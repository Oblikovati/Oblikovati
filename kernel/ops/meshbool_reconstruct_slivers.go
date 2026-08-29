// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// Near-tangent sliver collapse for reconstruction (ADR-0056, #2247). A gluing boolean of
// two operands that share a coincident wall — a lap-seam tab over a flange, two flush
// coplanar solids — imprints each operand's edge vertices onto the other along the shared
// surface. When those imprint points carry chained-construction precision dust (a vertex a
// sub-resolution 1e-19 off the wall's line for O(1) geometry), co-refinement emits a
// near-degenerate CAP triangle: three vertices spread along a near-straight line, its
// middle vertex sitting on the far edge. weldResultSoup collapses the caps whose vertices
// are within weld tolerance of EACH OTHER, but a cap that spans a large extent (the wall
// height) has vertices too far apart to vertex-weld, so it survives as a zero-area face
// whose whole arrangement boundary is one collinear "slit" run. Reconstruction cannot fit
// that run and declines, dropping the join to the faceted fallback (the proud tab's volume
// fell to 0.373× in TestCornerSeamOverlapAddsProudTab).
//
// The fix is combinatorial, not metric: a cap is a T-JUNCTION its own operand failed to
// resolve — its middle vertex m lies on a neighbour's long edge (p,q) that was not split
// there. collapseSlivers drops the cap and splits that neighbour at m, so the mesh keeps
// exact coordinates (m does not move; no vertex is invented) and stays watertight, while
// the degenerate face — and its slit — disappear. Detection reads a float distance against
// the weld tolerance (an approximation-quality gate, ADR-0042, separate from the exact
// topological predicates); the topology edit itself is exact vertex-index surgery.

// collapseSlivers removes near-degenerate cap triangles from a welded result soup by
// T-junction re-stitching, leaving a watertight mesh free of sub-resolution slivers.
func collapseSlivers(soup meshbool.TaggedSoup, tol float64) meshbool.TaggedSoup {
	verts, tris, tags := indexSoupExact(soup)
	// Bounded by the triangle count: each pass splits one neighbour (+1 tri) and drops one cap
	// (−1 tri), a net wash, so the guard is a generous ceiling. A cap that cannot be
	// re-stitched (no interior-foot neighbour) is left in place so the mesh is never torn; the
	// caller still declines it, exactly as before.
	for guard := 0; guard < 4*len(tris)+8; guard++ {
		si, m, a, b := findSliverCap(verts, tris, tol)
		if si < 0 {
			break
		}
		// The cap owns its long edge (a→b); the neighbour to split owns the REVERSE (b→a).
		// Splitting it at m into (b,m,apex),(m,a,apex) re-pairs the cap's short edges and
		// leaves the mesh watertight once the cap is dropped.
		ni := neighborAcross(tris, si, b, a)
		if ni < 0 {
			break // no neighbour owns the far edge (a torn input) — stop rather than worsen it
		}
		first, second, ok := splitTriangle(tris[ni], b, a, m)
		if !ok {
			break
		}
		tris[ni] = first
		tris = append(tris, second)
		tags = append(tags, tags[ni])
		tris = append(tris[:si], tris[si+1:]...)
		tags = append(tags[:si], tags[si+1:]...)
	}
	return rebuildSoup(verts, tris, tags)
}

// findSliverCap returns the index of a cap triangle plus its middle vertex m and the two
// endpoints p,q of its long edge (m lies strictly between p and q, within tol of the line),
// or si=-1 when none remains.
func findSliverCap(verts []meshbool.Point, tris [][3]int, tol float64) (si, m, p, q int) {
	for i, t := range tris {
		for k := 0; k < 3; k++ {
			mv := t[k]
			pv, qv := t[(k+1)%3], t[(k+2)%3]
			if vertexOnSegmentInterior(verts[mv], verts[pv], verts[qv], tol) {
				return i, mv, pv, qv
			}
		}
	}
	return -1, 0, 0, 0
}

// vertexOnSegmentInterior reports whether m lies within tol of segment (p,q) AND its foot
// is strictly interior — a T-junction, not a near-duplicate endpoint (vertex welding owns
// those). The long edge must exceed tol so a wholly sub-resolution triangle (all three
// vertices within tol) is not mistaken for a cap.
func vertexOnSegmentInterior(m, p, q meshbool.Point, tol float64) bool {
	mp, pp, qp := m.Round(), p.Round(), q.Round()
	seg := pp.VectorTo(qp)
	segLen := seg.Length()
	if segLen <= tol {
		return false
	}
	u := seg.Scale(1 / segLen)
	t := pp.VectorTo(mp).Dot(u)
	if t <= tol || t >= segLen-tol {
		return false // foot at or beyond an endpoint
	}
	foot := math.P3(pp.X+u.X*t, pp.Y+u.Y*t, pp.Z+u.Z*t)
	return foot.DistanceTo(mp) <= tol
}

// neighborAcross returns the triangle owning the directed edge (p→q) — the neighbour on the
// far side of the cap's long edge, whose reverse edge (q→p) the cap owns — or -1.
func neighborAcross(tris [][3]int, capIdx int, p, q int) int {
	for i, t := range tris {
		if i == capIdx {
			continue
		}
		for k := 0; k < 3; k++ {
			if t[k] == p && t[(k+1)%3] == q {
				return i
			}
		}
	}
	return -1
}

// splitTriangle splits triangle t, which owns directed edge (p→q), into two triangles
// meeting at m on that edge — (p,m,apex) and (m,q,apex) — preserving winding. Returns
// ok=false when t does not actually own (p→q).
func splitTriangle(t [3]int, p, q, m int) (first, second [3]int, ok bool) {
	for k := 0; k < 3; k++ {
		if t[k] == p && t[(k+1)%3] == q {
			apex := t[(k+2)%3]
			return [3]int{p, m, apex}, [3]int{m, q, apex}, true
		}
	}
	return first, second, false
}

// indexSoupExact deduplicates the soup's exact vertices (welded points are exact-equal, so a
// rational-string key is collision-free) and returns the indexed mesh.
func indexSoupExact(soup meshbool.TaggedSoup) (verts []meshbool.Point, tris [][3]int, tags []int) {
	index := map[string]int{}
	idOf := func(p meshbool.Point) int {
		key := p.X.RatString() + "|" + p.Y.RatString() + "|" + p.Z.RatString()
		if id, ok := index[key]; ok {
			return id
		}
		id := len(verts)
		index[key] = id
		verts = append(verts, p)
		return id
	}
	for i, t := range soup.Tris {
		tris = append(tris, [3]int{idOf(t[0]), idOf(t[1]), idOf(t[2])})
		tags = append(tags, soup.Tags[i])
	}
	return verts, tris, tags
}

// rebuildSoup materializes an indexed mesh back into a TaggedSoup.
func rebuildSoup(verts []meshbool.Point, tris [][3]int, tags []int) meshbool.TaggedSoup {
	out := meshbool.TaggedSoup{Tris: make([][3]meshbool.Point, len(tris)), Tags: make([]int, len(tris))}
	for i, t := range tris {
		out.Tris[i] = [3]meshbool.Point{verts[t[0]], verts[t[1]], verts[t[2]]}
		out.Tags[i] = tags[i]
	}
	return out
}
