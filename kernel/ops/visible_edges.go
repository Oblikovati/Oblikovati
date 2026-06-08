// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// VisibleEdges returns a body's display edge polylines with TANGENT edges suppressed: an edge
// whose two faces meet within creaseAngle (a smooth/tangent edge), or a parametric seam internal
// to a single smooth face, is omitted — matching CAD viewers (Inventor) that hide tangent edges
// so a faceted-smooth loft/sweep skin reads as one surface instead of a web of facet lines.
// Genuine edges (a box corner, a loft's section boundary, a chamfer) meet above the crease and
// are kept. Display only: b.Edges() and TessellateBody are unchanged, so picking, mass
// properties, and export still see every edge.
func VisibleEdges(b *topo.Body, q Quality, creaseAngle float64) [][]math.Point3 {
	cosThresh := stdmath.Cos(creaseAngle)
	var out [][]math.Point3
	for _, e := range b.Edges() {
		if isTangentEdge(e, cosThresh) {
			continue
		}
		out = append(out, TessellateEdge(e, q))
	}
	return out
}

// isTangentEdge reports whether an edge should be hidden as tangent: a seam internal to one
// smooth face, or a manifold edge whose two faces' outward normals (sampled at the edge
// midpoint) agree within the crease angle (dot ≥ cosThresh).
func isTangentEdge(e *topo.Edge, cosThresh float64) bool {
	faces := e.Faces()
	if len(e.Uses()) == 2 && len(faces) == 1 {
		return true // a parametric seam internal to one smooth face (e.g. a cylinder seam)
	}
	if len(faces) != 2 {
		return false // a boundary or non-manifold edge — always keep
	}
	mid := e.Geometry().PointAt(0.5)
	n0 := outwardFaceNormalAt(faces[0], mid)
	n1 := outwardFaceNormalAt(faces[1], mid)
	return float64(n0.Dot(n1)) > cosThresh
}

// outwardFaceNormalAt is a face's outward unit normal at point p on it (the surface normal,
// flipped when the face is reversed).
func outwardFaceNormalAt(f *topo.Face, p math.Point3) math.Vector3 {
	u, v := f.Geometry().ParamAt(p)
	n := f.Geometry().NormalAt(u, v)
	if f.Reversed() {
		n = n.Scale(-1)
	}
	return unitOr(n)
}
