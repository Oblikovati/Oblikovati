// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// blockFaceRef extrudes a block and returns a reference key for one of its faces.
func blockFaceRef(t *testing.T, r *Router, s *app.Session) string {
	t.Helper()
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct{}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatal(err)
	}
	bodies := part.SurfaceBodies().All()
	if len(bodies) == 0 || len(bodies[0].Faces()) == 0 {
		t.Fatal("extrude produced no faces")
	}
	return string(bodies[0].Faces()[0].ReferenceKey())
}

// TestSketch3DIntersectionFaceWorkPlaneOverWire intersects a part face with an origin work plane
// (#1854).
func TestSketch3DIntersectionFaceWorkPlaneOverWire(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	faceRef := blockFaceRef(t, r, s)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var res wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "intersection", FaceRefs: []string{faceRef}, WorkRefs: []string{"origin/plane/xz"},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &res)
	if !res.Healthy || res.EntityID == 0 {
		t.Fatalf("face ∩ work-plane intersection = %+v, want healthy with an entity id", res)
	}
}

// TestSketch3DIntersectionWrongOperandCountErrors: fewer or more than two operands is a clean error
// (#1854 AC2).
func TestSketch3DIntersectionWrongOperandCountErrors(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	faceRef := blockFaceRef(t, r, s)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "intersection", FaceRefs: []string{faceRef}, // only one operand
	})
	if err := tryCall(t, r, s, "sketch3d.addSurfaceCurve", string(args)); err == nil {
		t.Error("an intersection with one operand should error")
	}
}

// TestSketch3DIntersectionUnresolvedWorkPlaneIsUnhealthy: an unresolved work-plane ref reports
// unhealthy rather than erroring, matching the face path (#1854).
func TestSketch3DIntersectionUnresolvedWorkPlaneIsUnhealthy(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	faceRef := blockFaceRef(t, r, s)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var res wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "intersection", FaceRefs: []string{faceRef}, WorkRefs: []string{"origin/plane/bogus"},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &res)
	if res.Healthy {
		t.Errorf("intersection with an unresolved work plane = %+v, want unhealthy", res)
	}
}
