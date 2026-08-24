// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Constrained triangulation of one face, in exact arithmetic. A Triangulation
// starts as a single triangle and is refined by inserting the vertices and
// constraint segments produced by co-refinement, so the face ends up subdivided in
// conformance with the other operand. Point insertion (this file) splits the
// triangles a point lands in; because both triangles sharing a split edge detect
// the point on their own boundary, the mesh stays conforming with no adjacency
// bookkeeping. Constraint-edge insertion builds on this in a later layer.

// vertexLoc is where a point lies relative to one triangle.
type vertexLoc int

const (
	locOutside  vertexLoc = iota // strictly outside this triangle
	locInterior                  // strictly inside
	locEdge                      // on one edge (edgeIndex identifies which)
)

// Triangulation is a set of triangles over the vertices of one planar face,
// indexed for compact edges. All orientation tests use the face's fixed
// projection axis, so they stay exact and consistent.
type Triangulation struct {
	verts []Point
	tris  [][3]int
	axis  int
}

// NewTriangulation starts from a single non-degenerate triangle.
func NewTriangulation(t [3]Point) *Triangulation {
	return &Triangulation{
		verts: []Point{t[0], t[1], t[2]},
		tris:  [][3]int{{0, 1, 2}},
		axis:  planeAxis(t),
	}
}

// Triangles returns the current triangles as point triples (output order matches
// tris). Vertices keep the exact rational coordinates until the caller rounds.
func (tr *Triangulation) Triangles() [][3]Point {
	out := make([][3]Point, len(tr.tris))
	for i, t := range tr.tris {
		out[i] = [3]Point{tr.verts[t[0]], tr.verts[t[1]], tr.verts[t[2]]}
	}
	return out
}

// InsertPoint refines the triangulation so p becomes a vertex. A point strictly
// inside a triangle splits it into three; a point on an edge splits each incident
// triangle into two (keeping the shared edge conforming). A point equal to an
// existing vertex, or outside the face, is a no-op.
func (tr *Triangulation) InsertPoint(p Point) {
	if tr.indexOf(p) >= 0 {
		return
	}
	vi := len(tr.verts)
	tr.verts = append(tr.verts, p)
	next := make([][3]int, 0, len(tr.tris)+2)
	split := false
	for _, t := range tr.tris {
		rep := tr.splitAround(t, vi, p)
		if len(rep) > 1 { // interior→3 or edge→2 replacements means p landed here
			split = true
		}
		next = append(next, rep...)
	}
	if !split { // p was outside every triangle — drop the orphan vertex
		tr.verts = tr.verts[:vi]
		return
	}
	tr.tris = next
}

// splitAround returns triangle t's replacement(s) for inserting vertex vi at p:
// three triangles if p is interior, two if p is on an edge, or t unchanged if p is
// outside t.
func (tr *Triangulation) splitAround(t [3]int, vi int, p Point) [][3]int {
	loc, edge := tr.classify(t, p)
	switch loc {
	case locInterior:
		return [][3]int{{t[0], t[1], vi}, {t[1], t[2], vi}, {t[2], t[0], vi}}
	case locEdge:
		a, b, c := t[edge], t[(edge+1)%3], t[(edge+2)%3]
		return [][3]int{{a, vi, c}, {vi, b, c}}
	default:
		return [][3]int{t}
	}
}

// classify locates p relative to triangle t. p is inside/on t iff it is on the
// same rotational side of all three directed edges as t's own orientation; a zero
// on one edge means p lies on that edge.
func (tr *Triangulation) classify(t [3]int, p Point) (vertexLoc, int) {
	v0, v1, v2 := tr.verts[t[0]], tr.verts[t[1]], tr.verts[t[2]]
	o := orient2(v0, v1, v2, tr.axis) // triangle orientation (nonzero: t is non-degenerate)
	d := [3]int{
		orient2(v0, v1, p, tr.axis),
		orient2(v1, v2, p, tr.axis),
		orient2(v2, v0, p, tr.axis),
	}
	zeroEdge, zeros := 0, 0
	for i := 0; i < 3; i++ {
		if d[i] == 0 {
			zeroEdge, zeros = i, zeros+1
			continue
		}
		if (d[i] > 0) != (o > 0) {
			return locOutside, 0 // opposite side of an edge → outside
		}
	}
	switch zeros {
	case 0:
		return locInterior, 0
	case 1:
		return locEdge, zeroEdge
	default:
		return locOutside, 0 // two zeros = a vertex coincidence, caught by indexOf
	}
}

// indexOf returns the index of an existing vertex exactly equal to p, or -1.
func (tr *Triangulation) indexOf(p Point) int {
	for i, v := range tr.verts {
		if v.Equal(p) {
			return i
		}
	}
	return -1
}
