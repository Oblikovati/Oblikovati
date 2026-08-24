// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"sort"
)

// RefineFace triangulates a face in conformance with a set of constraint segments
// — the segments where the other operand's faces cross this one (from
// IntersectTriangles). Every segment endpoint and every pairwise crossing is
// inserted as a vertex, then each segment is forced as a chain of edges, so the
// result subdivides the face along the intersection curve with no segment left
// crossed by a triangle. This is the per-face core of co-refinement.
//
// PRECONDITION: each segment lies in the face's plane with endpoints on or inside
// the face (as IntersectTriangles guarantees). Points keep exact coordinates; the
// caller rounds the returned triangles.
func RefineFace(face [3]Point, segments [][2]Point) [][3]Point {
	tr := NewTriangulation(face)
	for _, v := range constraintVertices(segments, tr.axis) {
		tr.InsertPoint(v)
	}
	for _, s := range segments {
		tr.forceSegment(s[0], s[1])
	}
	return tr.Triangles()
}

// constraintVertices returns every segment endpoint plus every exact pairwise
// proper-crossing point, so co-refined sub-segments meet only at shared vertices.
func constraintVertices(segments [][2]Point, axis int) []Point {
	out := make([]Point, 0, 2*len(segments))
	for _, s := range segments {
		out = append(out, s[0], s[1])
	}
	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			if segmentsProperlyCross(segments[i][0], segments[i][1], segments[j][0], segments[j][1], axis) {
				out = append(out, SegSegCross(segments[i][0], segments[i][1], segments[j][0], segments[j][1], axis))
			}
		}
	}
	return out
}

// forceSegment forces the whole segment [a,b] as triangulation edges, splitting it
// at every vertex that lies on it so each consecutive sub-edge satisfies ForceEdge's
// no-vertex-strictly-between precondition.
func (tr *Triangulation) forceSegment(a, b Point) {
	on := tr.verticesOnSegment(a, b)
	for k := 0; k+1 < len(on); k++ {
		tr.ForceEdge(on[k], on[k+1])
	}
}

// verticesOnSegment returns the indices of all vertices lying on the closed
// segment [a,b], ordered from a to b by their exact parameter.
func (tr *Triangulation) verticesOnSegment(a, b Point) []int {
	lenSq := segParam(a, b, b) // (b-a)·(b-a)
	type onVert struct {
		idx int
		t   *big.Rat
	}
	var on []onVert
	for i, v := range tr.verts {
		if !rcollinear(a, b, v) {
			continue
		}
		t := segParam(a, b, v)
		if t.Sign() < 0 || t.Cmp(lenSq) > 0 {
			continue // beyond an endpoint
		}
		on = append(on, onVert{i, t})
	}
	sort.Slice(on, func(i, j int) bool { return on[i].t.Cmp(on[j].t) < 0 })
	idx := make([]int, len(on))
	for i, e := range on {
		idx[i] = e.idx
	}
	return idx
}
