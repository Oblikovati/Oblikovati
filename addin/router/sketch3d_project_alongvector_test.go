// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// projectSourceOverBlock extrudes a block and adds a 3D source line, returning a face key and the
// source line's entity id — the operands of a project-to-surface curve.
func projectSourceOverBlock(t *testing.T, r *Router, s *app.Session) (faceRef string, source uint64) {
	t.Helper()
	faceRef = blockFaceRef(t, r, s)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var line wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,20],[2,0,20]]}`, &line)
	return faceRef, line.EntityID
}

// TestSketch3DProjectAlongVectorOverWire projects a source curve onto a face along a direction
// (#1841).
func TestSketch3DProjectAlongVectorOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	faceRef, source := projectSourceOverBlock(t, r, s)

	var res wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "projectToSurface", FaceRefs: []string{faceRef},
		SourceEntityID: source, ProjectionType: "alongVector", ProjectDirection: []float64{0, 0, -1},
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &res)
	if !res.Healthy || res.EntityID == 0 {
		t.Fatalf("alongVector projection = %+v, want healthy with an entity id", res)
	}
}

// TestSketch3DProjectWrapOverWire wraps a source curve onto a face through an origin work plane as
// the flattening frame (#1841 part 2).
func TestSketch3DProjectWrapOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	faceRef, source := projectSourceOverBlock(t, r, s)

	var res wire.AddSketch3DSurfaceCurveResult
	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "projectToSurface", FaceRefs: []string{faceRef},
		SourceEntityID: source, ProjectionType: "wrap", WrapPlaneRef: "origin/plane/xy",
	})
	call(t, r, s, "sketch3d.addSurfaceCurve", string(args), &res)
	if !res.Healthy || res.EntityID == 0 {
		t.Fatalf("wrap projection = %+v, want healthy with an entity id", res)
	}
}

// TestSketch3DProjectWrapNeedsPlaneRef: wrap without a wrapPlaneRef errors (no flattening frame).
func TestSketch3DProjectWrapNeedsPlaneRef(t *testing.T) {
	r, s := emptyPartSession(t)
	faceRef, source := projectSourceOverBlock(t, r, s)

	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "projectToSurface", FaceRefs: []string{faceRef},
		SourceEntityID: source, ProjectionType: "wrap",
	})
	if err := tryCall(t, r, s, "sketch3d.addSurfaceCurve", string(args)); err == nil {
		t.Error("wrap projection without wrapPlaneRef should error")
	}
}

// TestSketch3DProjectUnknownTypeErrors rejects an unknown projection type (#1841).
func TestSketch3DProjectUnknownTypeErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	faceRef, source := projectSourceOverBlock(t, r, s)

	args, _ := json.Marshal(wire.AddSketch3DSurfaceCurveArgs{
		SketchIndex: 0, Kind: "projectToSurface", FaceRefs: []string{faceRef},
		SourceEntityID: source, ProjectionType: "sideways",
	})
	if err := tryCall(t, r, s, "sketch3d.addSurfaceCurve", string(args)); err == nil {
		t.Error("an unknown projectionType should error")
	}
}
