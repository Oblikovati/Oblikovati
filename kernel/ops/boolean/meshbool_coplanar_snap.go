// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"math/big"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/ops/internal/probe"
)

// Cross-operand coincident-plane reconciliation for reconstruction (ADR-0056 / #2247).
//
// The exact mesh boolean classifies a coplanar-coincident face pair by an EXACT predicate
// (meshbool.faceOnPlane, Orient3D==0). That is correct only when the two operands' coincident
// faces are bit-coincident. But the two operands are tessellated INDEPENDENTLY, so a planar face
// carried from a prior boolean result and a freshly-built coincident face differ by a sub-ULP
// jitter (~1e-17) off their common plane. The exact predicate then reads them as two distinct
// near-parallel planes, the coincident-OPPOSITE interface of a gluing union survives
// classification, and reconstruction declines a join it should annihilate.
//
// The ground rule "classify a comparison by the origin of its operands: independent sources →
// Sew()" says this cross-operand coincidence is a tolerance decision, not an exact one. meshbool
// stays exact; the Sew tolerance lives here in the ops layer (ADR-0056). We detect the coincident
// PLANES analytically (from the operand surfaces, ignoring normal sign so an opposite-facing pair
// still groups) and project every tessellation vertex of the grouped faces EXACTLY, by rational
// arithmetic, onto one shared canonical rational plane. The two faces become bit-coincident, the
// exact predicate is then correct, and the coincident-opposite pair annihilates through the same
// general pipeline. No analytic surface and no output coordinate is touched — reconstruction still
// rebuilds each face on its exact surface — so the derived tessellation is only made faithful to
// the plane it already approximates, never perturbed to force a result.

// snapCoincidentPlanes projects, onto one shared canonical rational plane, the tessellation
// vertices of every planar face that a face of the OTHER operand is coincident with (same plane,
// either orientation). It mutates the two operands' soups in place so meshbool's exact coplanar
// classification sees the intended coincidence.
func snapCoincidentPlanes(a, b *meshbool.TaggedSoup, refs []faceSurfaceRef, naRefs int, res geom.Resolution) {
	planeOf := planeGroupReps(refs, naRefs, res.Weld())
	if len(planeOf) == 0 {
		return
	}
	remap := buildPlaneSnapRemap(a, b, planeOf)
	if len(remap) == 0 {
		return
	}
	applyVertexRemap(a, remap)
	applyVertexRemap(b, remap)
}

// planeGroupReps returns, for each tag that is a planar face belonging to a CROSS-operand
// coincident-plane group, the canonical rational plane its vertices snap onto (the plane of the
// smallest tag in the group). Only groups holding a face from operand A (tag < naRefs) AND a face
// from operand B (tag >= naRefs) are returned — same-operand coplanar faces are already welded and
// need no cross-operand reconciliation.
func planeGroupReps(refs []faceSurfaceRef, naRefs int, tol float64) map[int]ratPlane {
	rep := make([]int, len(refs))
	for i := range rep {
		rep[i] = i
	}
	for i := range refs {
		pi, ok := refs[i].surface.(geom.Plane)
		if !ok {
			continue
		}
		for j := i + 1; j < len(refs); j++ {
			pj, ok := refs[j].surface.(geom.Plane)
			if ok && planesShareSurface(pi, pj, tol) {
				if ri, rj := findRoot(rep, i), findRoot(rep, j); ri != rj {
					rep[max2(ri, rj)] = min2(ri, rj) // keep the smaller tag as the group's canonical
				}
			}
		}
	}
	return crossOperandPlanes(rep, refs, naRefs)
}

// crossOperandPlanes collapses the union-find to roots and returns the canonical plane for every
// tag whose group spans both operands.
func crossOperandPlanes(rep []int, refs []faceSurfaceRef, naRefs int) map[int]ratPlane {
	hasA, hasB := map[int]bool{}, map[int]bool{}
	for i := range rep {
		if _, ok := refs[i].surface.(geom.Plane); !ok {
			continue
		}
		if r := findRoot(rep, i); i < naRefs {
			hasA[r] = true
		} else {
			hasB[r] = true
		}
	}
	out := map[int]ratPlane{}
	for i := range rep {
		if _, ok := refs[i].surface.(geom.Plane); !ok {
			continue
		}
		r := findRoot(rep, i)
		if hasA[r] && hasB[r] {
			out[i] = ratPlaneOf(refs[r].surface.(geom.Plane)) // r is the smallest tag: deterministic canonical plane
		}
	}
	return out
}

// planesShareSurface reports whether two planes are the same UNORIENTED plane: parallel normals
// (either sign) and a common offset. It is planesCoincide without the same-side requirement, so an
// opposite-facing coincident pair (the annihilating interface of a gluing union) groups too.
func planesShareSurface(a, b geom.Plane, tol float64) bool {
	na, nb := probe.Unit(a.Normal()), probe.Unit(b.Normal())
	if probe.AbsFloat(na.Dot(nb)) < 1-directionTol {
		return false
	}
	return probe.AbsFloat(a.Origin.VectorTo(b.Origin).Dot(na)) <= tol
}

// buildPlaneSnapRemap maps each grouped-face vertex to its exact projection onto the group's
// canonical plane, keyed by the vertex's exact value so all welded copies (including a shared
// edge's copy on a neighbour face) remap identically. A vertex that lies on TWO different group
// planes is a corner where two coincident interfaces meet; snapping it onto one plane would move it
// off the other, so every group sharing such a corner is left unsnapped (a named decline — that
// interface stays faceted through the existing reconstruction fallback rather than shipping a
// distorted face). The common single-interface case snaps exactly.
func buildPlaneSnapRemap(a, b *meshbool.TaggedSoup, planeOf map[int]ratPlane) map[vertexKey]meshbool.Point {
	assign := map[vertexKey][]ratPlane{}
	src := map[vertexKey]meshbool.Point{}
	for _, s := range []*meshbool.TaggedSoup{a, b} {
		for i := range s.Tris {
			pl, ok := planeOf[s.Tags[i]]
			if !ok {
				continue
			}
			for _, v := range s.Tris[i] {
				k := keyOf(v)
				if _, seen := src[k]; !seen {
					src[k] = v
				}
				assign[k] = appendPlane(assign[k], pl)
			}
		}
	}
	conflicted := conflictedPlanes(assign)
	remap := make(map[vertexKey]meshbool.Point, len(assign))
	for k, planes := range assign {
		if len(planes) == 1 && !containsPlane(conflicted, planes[0]) {
			remap[k] = planes[0].project(src[k])
		}
	}
	return remap
}

// conflictedPlanes returns every plane that meets another group plane at a shared vertex — the
// planes whose groups must be left unsnapped.
func conflictedPlanes(assign map[vertexKey][]ratPlane) []ratPlane {
	var out []ratPlane
	for _, planes := range assign {
		if len(planes) > 1 {
			for _, pl := range planes {
				out = appendPlane(out, pl)
			}
		}
	}
	return out
}

// applyVertexRemap rewrites every soup vertex that appears in remap to its snapped value.
func applyVertexRemap(s *meshbool.TaggedSoup, remap map[vertexKey]meshbool.Point) {
	for i := range s.Tris {
		for j := range s.Tris[i] {
			if q, ok := remap[keyOf(s.Tris[i][j])]; ok {
				s.Tris[i][j] = q
			}
		}
	}
}

// vertexKey is a soup vertex's exact value as a comparable map key.
type vertexKey [3]string

func keyOf(p meshbool.Point) vertexKey {
	return vertexKey{p.X.RatString(), p.Y.RatString(), p.Z.RatString()}
}

// ratPlane is a plane through o with normal n, all exact rationals, so a projection onto it is
// exact and its result lies bit-exactly on the plane.
type ratPlane struct{ o, n [3]*big.Rat }

func ratPlaneOf(p geom.Plane) ratPlane {
	n := p.Normal()
	return ratPlane{
		o: [3]*big.Rat{ratF(p.Origin.X), ratF(p.Origin.Y), ratF(p.Origin.Z)},
		n: [3]*big.Rat{ratF(n.X), ratF(n.Y), ratF(n.Z)},
	}
}

// project returns the exact orthogonal projection of p onto pl: q = p - ((n·(p-o))/(n·n)) n, which
// satisfies n·(q-o)=0 exactly in rational arithmetic — so p lands bit-exactly on the plane.
func (pl ratPlane) project(p meshbool.Point) meshbool.Point {
	diff := [3]*big.Rat{
		new(big.Rat).Sub(p.X, pl.o[0]),
		new(big.Rat).Sub(p.Y, pl.o[1]),
		new(big.Rat).Sub(p.Z, pl.o[2]),
	}
	t := new(big.Rat).Quo(ratDot(pl.n, diff), ratDot(pl.n, pl.n))
	comp := [3]*big.Rat{p.X, p.Y, p.Z}
	for i := range comp {
		comp[i] = new(big.Rat).Sub(comp[i], new(big.Rat).Mul(t, pl.n[i]))
	}
	return meshbool.Point{X: comp[0], Y: comp[1], Z: comp[2]}
}

// appendPlane adds pl to planes unless an equal plane is already present.
func appendPlane(planes []ratPlane, pl ratPlane) []ratPlane {
	if containsPlane(planes, pl) {
		return planes
	}
	return append(planes, pl)
}

func containsPlane(planes []ratPlane, pl ratPlane) bool {
	for _, q := range planes {
		if q.equal(pl) {
			return true
		}
	}
	return false
}

func (pl ratPlane) equal(q ratPlane) bool {
	for i := range pl.o {
		if pl.o[i].Cmp(q.o[i]) != 0 || pl.n[i].Cmp(q.n[i]) != 0 {
			return false
		}
	}
	return true
}

func ratDot(a, b [3]*big.Rat) *big.Rat {
	s := new(big.Rat)
	for i := range a {
		s.Add(s, new(big.Rat).Mul(a[i], b[i]))
	}
	return s
}

func ratF(x float64) *big.Rat {
	return new(big.Rat).SetFloat64(x)
}
