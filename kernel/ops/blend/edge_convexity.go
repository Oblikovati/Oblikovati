// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Edge convexity classification (M07-F07, Oblikovati/Oblikovati#630): the
// reference ConcaveEdges/ConvexEdges collections. At each manifold edge the
// dihedral sign of the two material-outward face normals about the edge
// tangent (as traversed by the first face) decides: positive triple product =
// convex (material angle < π), negative = concave, near-parallel normals =
// tangentially connected.

// EdgeConvexity is one edge's dihedral class.
type EdgeConvexity uint8

const (
	EdgeConvex EdgeConvexity = iota
	EdgeConcave
	EdgeTangent
	EdgeConvexityUnknown // boundary or non-manifold edges have no dihedral
)

// String returns a stable name for diagnostics.
func (c EdgeConvexity) String() string {
	switch c {
	case EdgeConvex:
		return "convex"
	case EdgeConcave:
		return "concave"
	case EdgeTangent:
		return "tangent"
	default:
		return "unknown"
	}
}

// tangentDihedralTol is the normal-angle band (radians) treated as
// tangentially connected rather than convex/concave.
const tangentDihedralTol = 1e-3

// ClassifyEdgeConvexity returns one edge's dihedral class.
func ClassifyEdgeConvexity(e *topo.Edge) EdgeConvexity {
	uses := e.Uses()
	if len(uses) != 2 {
		return EdgeConvexityUnknown
	}
	mid := edgeMidpoint(e)
	n1, ok1 := outwardFaceNormal(uses[0].Loop().Face(), mid)
	n2, ok2 := outwardFaceNormal(uses[1].Loop().Face(), mid)
	if !ok1 || !ok2 {
		return EdgeConvexityUnknown
	}
	if float64(n1.AngleTo(n2)) < tangentDihedralTol {
		return EdgeTangent
	}
	t := edgeUseTangent(uses[0])
	triple := float64(n1.Cross(n2).Dot(t))
	if triple > 0 {
		return EdgeConvex
	}
	return EdgeConcave
}

// BodyEdgeConvexity classifies every edge, bucketed by class.
//
// Example: byClass := ops.BodyEdgeConvexity(b); concave := byClass[ops.EdgeConcave]
func BodyEdgeConvexity(b *topo.Body) map[EdgeConvexity][]*topo.Edge {
	out := map[EdgeConvexity][]*topo.Edge{}
	for _, e := range b.Edges() {
		c := ClassifyEdgeConvexity(e)
		out[c] = append(out[c], e)
	}
	return out
}

// edgeMidpoint evaluates the edge curve at mid-parameter.
func edgeMidpoint(e *topo.Edge) math.Point3 {
	lo, hi := e.Geometry().Domain()
	return e.Geometry().PointAt((lo + hi) / 2)
}

// outwardFaceNormal is the face's material-outward normal nearest p (surface
// normal at the inverted parameters, negated for a reversed face).
func outwardFaceNormal(f *topo.Face, p math.Point3) (math.Vector3, bool) {
	u, v := f.Geometry().ParamAt(p)
	n := f.Geometry().NormalAt(u, v)
	l := float64(n.Length())
	if l == 0 || stdmath.IsNaN(l) {
		return math.Vector3{}, false
	}
	n = n.Scale(math.Scalar(1 / l))
	if f.Reversed() {
		n = n.Negate()
	}
	return n, true
}

// edgeUseTangent is the mid-edge curve tangent, oriented the way the use
// traverses it.
func edgeUseTangent(u *topo.EdgeUse) math.Vector3 {
	c := u.Edge().Geometry()
	lo, hi := c.Domain()
	t := c.TangentAt((lo + hi) / 2)
	if l := float64(t.Length()); l > 0 {
		t = t.Scale(math.Scalar(1 / l))
	}
	if u.Reversed() {
		t = t.Negate()
	}
	return t
}

func WireExtend(box math.Box, w *topo.Wire) math.Box {
	for _, u := range w.Uses() {
		c := u.Edge.Geometry()
		lo, hi := c.Domain()
		for i := 0; i <= 32; i++ {
			box = box.ExtendPoint(c.PointAt(lo + (hi-lo)*float64(i)/32))
		}
	}
	return box
}
