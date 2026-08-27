// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// Same-surface adjacent-face merge for reconstruction (ADR-0054). A gluing boolean of two
// bodies that share a surface — the #2167 cocylindrical join, or two coplanar-faced solids
// welded along a face — leaves the shared surface as TWO tagged regions (e.g. a lower
// cylinder wall and an upper cocylindrical wall) meeting along a false seam that is not a
// real geometric edge: the surface is smooth across it. Reconstructing the two regions as
// separate faces fails, because the seam run is neither an original edge nor a clean
// surface-surface intersection.
//
// The fix is to relabel every tag to a canonical representative of its COINCIDENT-surface
// group (same plane with the same normal, or the same cylinder axis+radius, and the same
// material side) BEFORE the arrangement is traced. The boundary walker only draws an edge
// between DIFFERENT tags, so a seam between two now-equal tags becomes interior and
// vanishes; adjacency is still honoured because the arrangement groups by tag-CONNECTED
// component, so two disjoint coincident faces stay separate. The merged region is then one
// analytic face carrying the shared surface, with a boundary of only real edges.

// directionTol is the dot-product slack within which two unit directions count as parallel.
const directionTol = 1e-7

// mergeCoincidentTags returns, for each tag, the canonical (smallest) tag of its
// coincident-surface group, plus the size of each group keyed by that representative. Tags
// merge only when their surfaces coincide geometrically AND bound material on the same side
// (same reversed sense), so opposite-facing coplanar caps never fuse.
func mergeCoincidentTags(refs []faceSurfaceRef, res geom.Resolution) (rep []int, size map[int]int) {
	rep = make([]int, len(refs))
	for i := range refs {
		rep[i] = i
	}
	for i := range refs {
		for j := i + 1; j < len(refs); j++ {
			if surfacesCoincide(refs[i], refs[j], res) {
				if ri, rj := findRoot(rep, i), findRoot(rep, j); ri != rj {
					rep[max2(ri, rj)] = min2(ri, rj) // keep the smaller tag as representative
				}
			}
		}
	}
	size = make(map[int]int, len(refs))
	for i := range refs {
		rep[i] = findRoot(rep, i)
		size[rep[i]]++
	}
	return rep, size
}

// findRoot returns the union-find root of i in rep, compressing the path it walks.
func findRoot(rep []int, i int) int {
	for rep[i] != i {
		rep[i] = rep[rep[i]]
		i = rep[i]
	}
	return i
}

// relabelTags rewrites each tag in place to its group representative.
func relabelTags(tags []int, rep []int) {
	for i, t := range tags {
		if t >= 0 && t < len(rep) {
			tags[i] = rep[t]
		}
	}
}

// surfacesCoincide reports whether two operand faces lie on the same surface and bound
// material on the same side — the condition under which their two regions are one face.
func surfacesCoincide(a, b faceSurfaceRef, res geom.Resolution) bool {
	if a.reversed != b.reversed {
		return false
	}
	switch sa := a.surface.(type) {
	case geom.Plane:
		sb, ok := b.surface.(geom.Plane)
		return ok && planesCoincide(sa, sb, res.Weld())
	case geom.Cylinder:
		sb, ok := b.surface.(geom.Cylinder)
		return ok && cylindersCoincide(sa, sb, res.Weld())
	}
	return false // sphere/cone merge not needed by the current gluing cases (SSI layer)
}

// planesCoincide reports whether two planes have the same oriented normal and pass through
// the same points (b's origin lies on a within tol).
func planesCoincide(a, b geom.Plane, tol float64) bool {
	na, nb := unit3(a.Normal()), unit3(b.Normal())
	if na.Dot(nb) < 1-directionTol {
		return false // different or opposite normal → not the same oriented plane
	}
	return stdmath.Abs(a.Origin.VectorTo(b.Origin).Dot(na)) <= tol
}

// cylindersCoincide reports whether two cylinders share an axis line and radius (the axis
// SIGN is irrelevant — the surface and its outward radial normal are the same either way).
func cylindersCoincide(a, b geom.Cylinder, tol float64) bool {
	if stdmath.Abs(a.Radius-b.Radius) > tol {
		return false
	}
	da, db := a.AxisDir.AsVector(), b.AxisDir.AsVector()
	if stdmath.Abs(da.Dot(db)) < 1-directionTol {
		return false // axes not parallel
	}
	return a.Origin.VectorTo(b.Origin).Cross(da).Length() <= tol // b's axis point lies on a's axis
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}
