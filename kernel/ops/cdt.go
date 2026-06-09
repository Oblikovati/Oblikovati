// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati/math"
	"oblikovati/math/predicate"
)

// 2D constrained Delaunay triangulation (CDT) of a planar polygon (outer loop minus holes) with
// optional interior Steiner points. Built for trimmed curved-face tessellation in (u,v): the trim
// boundary loops become hard constraints, interior grid points refine the patch, and the domain is
// extracted by flooding across the constrained edges — so a CONCAVE trim is respected exactly (no
// triangles bridge a concavity) and a slightly self-intersecting boundary still meshes (the
// boundary segments are recovered by edge flips, then walls for the flood). The orientation and
// in-circle tests use the adaptive-EXACT predicates (see [orient2d]/[inCircle]): a planar face's
// boundary is sampled into near-collinear and near-cocircular points where a naive float determinant
// mis-signs and tangles the mesh.
//
// Algorithm: Bowyer–Watson incremental Delaunay over all points (connected-cavity insertion, which
// is robust to circumcircle round-off), then each boundary segment is recovered by flipping the
// edges it crosses, then triangles are 2-coloured inside/outside by flooding triangle adjacency and
// toggling at every constrained edge.

// cdtTri is one triangle: CCW vertices v, and n[i] = the neighbour across the edge opposite v[i]
// (the edge v[(i+1)%3]–v[(i+2)%3]), or -1 at the convex-hull boundary.
type cdtTri struct {
	v [3]int
	n [3]int
}

type cdt struct {
	pts  [][2]float64
	tris []cdtTri
	dead []bool          // tris[t] was deleted by an insertion
	con  map[[2]int]bool // constrained undirected edges (sorted vertex pair)
	nsup int             // index of the first super-triangle vertex (points >= nsup are super)
}

func conKey(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

// orient2d > 0 when c is left of the directed line a→b (triangle abc is CCW). It delegates to the
// adaptive-exact predicate: the naive float determinant suffers catastrophic cancellation on near-
// collinear points (e.g. a face boundary discretized into many almost-straight samples), returning the
// WRONG sign — which made the Bowyer–Watson cavity collection grab the wrong triangles and emit
// inverted/overlapping triangles. The exact predicate's sign is always correct.
func orient2d(a, b, c [2]float64) float64 {
	return predicate.Orient2D(p2(a), p2(b), p2(c))
}

// inCircle > 0 when d is strictly inside the circumcircle of CCW triangle abc. Delegates to the
// adaptive-exact predicate for the same reason as orient2d: a planar face's circular hole is sampled
// into (near-)cocircular points, where the naive float in-circle determinant (radius² terms swamping
// their tiny difference) returns the wrong sign and tangles the triangulation.
func inCircle(a, b, c, d [2]float64) float64 {
	return predicate.InCircle(p2(a), p2(b), p2(c), p2(d))
}

func p2(p [2]float64) math.Point2 { return math.Point2{X: p[0], Y: p[1]} }

// newCDT builds the initial triangulation: a single super-triangle large enough to contain every
// input point (the 3 super vertices are appended to pts at indices nsup, nsup+1, nsup+2).
func newCDT(pts [][2]float64) *cdt {
	minx, miny := pts[0][0], pts[0][1]
	maxx, maxy := minx, miny
	for _, p := range pts {
		minx, maxx = min(minx, p[0]), max(maxx, p[0])
		miny, maxy = min(miny, p[1]), max(maxy, p[1])
	}
	d := max(maxx-minx, maxy-miny)
	if d <= 0 {
		d = 1
	}
	cx, cy := (minx+maxx)/2, (miny+maxy)/2
	nsup := len(pts)
	all := append([][2]float64(nil), pts...)
	all = append(all,
		[2]float64{cx - 20*d, cy - d}, [2]float64{cx + 20*d, cy - d}, [2]float64{cx, cy + 20*d})
	m := &cdt{pts: all, con: map[[2]int]bool{}, nsup: nsup}
	m.tris = []cdtTri{{v: [3]int{nsup, nsup + 1, nsup + 2}, n: [3]int{-1, -1, -1}}}
	m.dead = []bool{false}
	return m
}

// localEdge returns the local index i (0..2) of the edge of triangle t between vertices a and b
// (unordered), or -1 if t has no such edge.
func (m *cdt) localEdge(t, a, b int) int {
	tri := m.tris[t]
	for i := 0; i < 3; i++ {
		u, w := tri.v[(i+1)%3], tri.v[(i+2)%3]
		if (u == a && w == b) || (u == b && w == a) {
			return i
		}
	}
	return -1
}

// relinkOpposite finds the edge of s shared with t and points it back at nt (used after t is
// replaced by a new triangle nt across that shared edge).
func (m *cdt) relinkOpposite(s, a, b, nt int) {
	if s < 0 {
		return
	}
	if i := m.localEdge(s, a, b); i >= 0 {
		m.tris[s].n[i] = nt
	}
}
