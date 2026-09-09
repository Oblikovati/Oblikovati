// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "oblikovati.org/math"

// The vertex accumulator every COVERING-SPACE mesher shares (#1510, ADR-0061).
//
// A face on a periodic surface whose region wraps the seam has no simple polygon in one branch of the
// surface's parameters, so it is meshed in the covering space instead: the boundary and the interior
// grid are replicated a whole period either way, ONE constrained triangulation covers the lot, and each
// triangle is kept when its centroid falls in the canonical window and on the material side. Period
// copies of a boundary point are the SAME 3D point, so welding the kept triangles closes every seam.
//
// Two meshers do that — the periodic B-spline band cover and the chart-driven analytic mesher — and
// they differ in exactly three things: how a covering point's parameters fold back onto the surface for
// its normal, whether a replicated point is worth carrying at all, and what "material" means. Those are
// the three functions this struct takes; everything else below is the construction itself, held once.

// coverVertices accumulates a covering's vertices: exact 3D positions with their surface normals, the
// metric-scaled (u,v) the triangulation runs in, and the UNFOLDED (u,v) the canonical selection reads.
type coverVertices struct {
	// normalAt is the surface normal at a covering point. The owner folds the parameters back onto the
	// surface: the branch a replica lives on is bookkeeping, not geometry.
	normalAt func(u, v float64) math.Vector3
	// carry decides whether a replicated point is close enough to the canonical window to be worth
	// triangulating. nil carries every replica, which is what a cover of only three copies wants.
	carry  func(u, v float64) bool
	su, sv float64 // the (u,v) metric, so the triangulation runs in a space isometric to 3D
	pos    []math.Point3
	nrm    []math.Vector3
	xy     [][2]float64
	uu, vv []float64
}

// add records one covering vertex and returns its index.
func (c *coverVertices) add(p math.Point3, u, v float64) int {
	i := len(c.pos)
	c.pos = append(c.pos, p)
	c.nrm = append(c.nrm, c.normalAt(u, v))
	c.xy = append(c.xy, [2]float64{u * c.su, v * c.sv})
	c.uu, c.vv = append(c.uu, u), append(c.vv, v)
	return i
}

// place records a vertex the owner may not want at this replica, returning -1 when it declines.
func (c *coverVertices) place(p math.Point3, u, v float64) int {
	if c.carry != nil && !c.carry(u, v) {
		return -1
	}
	return c.add(p, u, v)
}

// addChain lays one boundary chain into the covering at a whole-period offset and returns its
// PER-SEGMENT constraints. Per-segment rather than as a closed loop because a boundary that wraps a
// period does not close in the covering space, and a spurious closing edge would constrain a chord the
// face does not have; the owner classifies triangles itself rather than by the loop-parity flood.
func (c *coverVertices) addChain(p3 []math.Point3, uv []math.Point2, du, dv float64) [][]int {
	idx := make([]int, len(p3))
	for i := range p3 {
		idx[i] = c.place(p3[i], float64(uv[i].X)+du, float64(uv[i].Y)+dv)
	}
	return chainConstraints(idx)
}

// addRing lays a chain that DOES close in the covering space as one loop constraint (constrain wraps
// its last edge back to its first), returning its vertex index sequence.
func (c *coverVertices) addRing(p3 []math.Point3, uv []math.Point2, du, dv float64) []int {
	idx := make([]int, len(p3))
	for i := range p3 {
		idx[i] = c.add(p3[i], float64(uv[i].X)+du, float64(uv[i].Y)+dv)
	}
	return idx
}

// chainConstraints is a chain's consecutive index pairs, skipping any segment an end of which the
// carry gate declined — that segment's own replica a period along carries it.
func chainConstraints(idx []int) [][]int {
	var segs [][]int
	for i := 0; i+1 < len(idx); i++ {
		if idx[i] >= 0 && idx[i+1] >= 0 {
			segs = append(segs, []int{idx[i], idx[i+1]})
		}
	}
	return segs
}

// centroid is a triangle's mean UNFOLDED (u,v) — the point the canonical selection reads.
func (c *coverVertices) centroid(t [3]int) (u, v float64) {
	return (c.uu[t[0]] + c.uu[t[1]] + c.uu[t[2]]) / 3, (c.vv[t[0]] + c.vv[t[1]] + c.vv[t[2]]) / 3
}

// keepCanonical keeps each triangle whose centroid the predicate accepts. Period replication gives
// every seam-spanning triangle exactly one translate whose centroid is in the canonical window, so a
// predicate that is half-open there de-duplicates the seam without ever cutting the mesh at it.
func (c *coverVertices) keepCanonical(tris [][3]int, material func(u, v float64) bool) [][3]int {
	out := make([][3]int, 0, len(tris))
	for _, t := range tris {
		if u, v := c.centroid(t); material(u, v) {
			out = append(out, t)
		}
	}
	return out
}
