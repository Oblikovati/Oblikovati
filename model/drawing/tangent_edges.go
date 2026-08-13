// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Tangent-edge classification (#1984). A drawing curve is tagged "tangent" when its source model edge
// is smooth — its two adjacent faces meet with (near-)equal outward normals at the edge, i.e. a
// fillet/blend runout rather than a sharp crease. A view can then suppress these edges. The role is
// re-derived from the model each recompute, so it stays associative.

// cosTangentTol: two faces whose outward normals agree to within ~1° at the shared edge read as a
// smooth (tangent) transition.
const cosTangentTol = 0.9998

// tangentEdgeKeys returns the set (by string reference key) of the body's smooth/tangent edges.
func tangentEdgeKeys(body *topo.Body) map[string]bool {
	out := map[string]bool{}
	for _, e := range body.Edges() {
		if edgeIsTangent(e) {
			out[string(e.ReferenceKey())] = true
		}
	}
	return out
}

// edgeIsTangent reports whether an edge's two faces meet tangentially at a representative point on
// the edge (equal outward normals) — the smooth-transition test.
func edgeIsTangent(e *topo.Edge) bool {
	faces := e.Faces()
	if len(faces) != 2 {
		return false
	}
	p := edgePoint(e)
	n1, ok1 := faceOutwardNormalAt(faces[0], p)
	n2, ok2 := faceOutwardNormalAt(faces[1], p)
	if !ok1 || !ok2 {
		return false
	}
	return float64(n1.Dot(n2)) >= cosTangentTol
}

// edgePoint returns a point on the edge to sample the adjacent surfaces at — its start vertex, or its
// range-box centre when the edge has no start vertex.
func edgePoint(e *topo.Edge) math.Point3 {
	if v := e.StartVertex(); v != nil {
		return v.Point()
	}
	return e.RangeBox().Center()
}

// faceOutwardNormalAt evaluates a face's outward unit normal at a 3D point (flipped when the face is
// reversed), or ok=false when the normal is degenerate there.
func faceOutwardNormalAt(f *topo.Face, p math.Point3) (math.Vector3, bool) {
	surf := f.Geometry()
	u, v := surf.ParamAt(p)
	unit, err := math.UnitVector3FromVector(surf.NormalAt(u, v))
	if err != nil {
		return math.Vector3{}, false
	}
	n := unit.AsVector()
	if f.Reversed() {
		n = n.Negate()
	}
	return n, true
}

// SetDisplayTangentEdges shows or hides this view's smooth tangent edges (#1984).
func (v *DrawingView) SetDisplayTangentEdges(show bool) { v.hideTangentEdges = !show }

// DisplayTangentEdges reports whether the view draws its smooth tangent edges (the default).
func (v *DrawingView) DisplayTangentEdges() bool { return !v.hideTangentEdges }
