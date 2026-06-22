// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/scene"
)

// downLookingBoxWithPlanes builds the box and installs a real RayPicker (bodies AND work planes)
// looking straight down — the production pick setup, so the create-sketch hover/pick path runs
// against genuine geometry with the origin planes hidden behind the solid (the bug's setting).
func downLookingBoxWithPlanes(t *testing.T) *Session {
	t.Helper()
	s := extrudedBox(t, 2, 4) // [0,2]×[0,2]×[0,4]
	cam := scene.NewCamera(400, 400)
	cam.Eye, cam.Target, cam.Up = math.P3(1, 1, 20), math.P3(1, 1, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	s.SetPicker(NewRayPicker(cam, partBodies(s)).
		WithPlanes(func() []*feature.WorkPlane { return s.PickableWorkPlanes() }))
	return s
}

// TestCreateSketchToolAcceptsFaceAndPlaneHosts pins that Create 2D Sketch lets BOTH valid sketch
// hosts highlight/pick: a work plane and a planar face. Before the fix the filter admitted only
// work planes, so a face never highlighted and a click over it picked the origin plane behind it.
func TestCreateSketchToolAcceptsFaceAndPlaneHosts(t *testing.T) {
	s, _ := emptyPartSession(t)
	tool := NewCreateSketchTool()
	tool.Start(s)
	f := s.Selection().Filter()
	if !f.Accepts(SelectWorkPlane) || !f.Accepts(SelectFace) {
		t.Errorf("create-sketch filter must accept work planes AND faces; accepts(plane)=%v accepts(face)=%v",
			f.Accepts(SelectWorkPlane), f.Accepts(SelectFace))
	}
	if f.Accepts(SelectEdge) {
		t.Error("create-sketch filter should not accept edges (not a sketch host)")
	}
}

// TestCreateSketchPicksFaceNotPlaneBehind is the regression for the reported bug: with the tool
// active and a solid present, a pick over the top face must resolve to the FACE — not the XY origin
// plane hidden behind the box — and clicking it enters a sketch on that face's plane.
func TestCreateSketchPicksFaceNotPlaneBehind(t *testing.T) {
	s := downLookingBoxWithPlanes(t)
	s.StartTool(NewCreateSketchTool())

	sel, ok := s.PickAt(200, 200, s.Selection().Filter())
	if !ok {
		t.Fatal("create-sketch pick over the top face returned nothing")
	}
	if _, isFace := sel.(FaceHandle); !isFace {
		t.Fatalf("pick over a face = %T, want FaceHandle (not the plane behind the box)", sel)
	}

	s.Click(200, 200) // auto-commit onto the face
	if s.ActiveTool() != nil || !s.InSketch() {
		t.Fatal("clicking a planar face must end the tool and enter the sketch environment")
	}
	pl := s.ActiveSketch().Plane()
	if n := pl.Normal().AsVector(); stdmath.Abs(float64(n.Z)) < 0.999 {
		t.Errorf("sketch plane normal = %v, want the top cap (±Z)", n)
	}
	if z := float64(pl.Origin().Z); stdmath.Abs(z-4) > 1e-6 {
		t.Errorf("sketch plane origin z = %v, want 4 (on the top cap)", z)
	}
}

// TestCreateSketchToolIgnoresNonPlanarFace: a cylindrical face cannot host a sketch, so feeding it
// to the tool must not arm a commit (and committing then errors rather than sketching on garbage).
func TestCreateSketchToolIgnoresNonPlanarFace(t *testing.T) {
	s, body := newPartWithCylinder(t)
	tool := NewCreateSketchTool()
	tool.Start(s)
	tool.Pick(s, FaceHandle{Face: cylinderFaceOf(t, body), Body: body})
	if tool.CanCommit() {
		t.Fatal("a non-planar (cylindrical) face must not arm the sketch commit")
	}
	if err := tool.Commit(s); err == nil {
		t.Error("committing with only a non-planar face picked should error")
	}
}

// TestSketchPlaneFromFace: a planar face yields its plane (matching normal); a curved one does not.
func TestSketchPlaneFromFace(t *testing.T) {
	s := downLookingBoxWithPlanes(t)
	sel, _ := s.PickAt(200, 200, NewSelectionFilter(SelectFace))
	pl, ok := sketchPlaneFromFace(sel.(FaceHandle))
	if !ok {
		t.Fatal("a planar face must yield a sketch plane")
	}
	faceNormal := sel.(FaceHandle).Face.Geometry().(geom.Plane).Normal()
	if !sameDir(pl.Normal().AsVector(), faceNormal) {
		t.Errorf("derived plane normal %v != face normal %v", pl.Normal().AsVector(), faceNormal)
	}

	_, body := newPartWithCylinder(t)
	if _, ok := sketchPlaneFromFace(FaceHandle{Face: cylinderFaceOf(t, body), Body: body}); ok {
		t.Error("a cylindrical face must not yield a sketch plane")
	}
}

// TestSelectedSketchHostPlaneFromFace: a pre-selected planar face is a sketch host, so running
// Create 2D Sketch with a face selected sketches on it immediately (no tool) — Inventor's
// "select a face, then Create Sketch" flow.
func TestSelectedSketchHostPlaneFromFace(t *testing.T) {
	s := downLookingBoxWithPlanes(t)
	sel, _ := s.PickAt(200, 200, NewSelectionFilter(SelectFace))
	s.Selection().Add(sel)

	if _, ok := s.SelectedSketchHostPlane(); !ok {
		t.Fatal("a selected planar face must be reported as a sketch host")
	}
	sk, err := s.CreateSketchOnSelectedPlane()
	if err != nil {
		t.Fatalf("CreateSketchOnSelectedPlane on a face: %v", err)
	}
	if z := float64(sk.Plane().Origin().Z); stdmath.Abs(z-4) > 1e-6 {
		t.Errorf("sketch on selected face: origin z = %v, want 4", z)
	}
}

// sameDir reports whether two vectors point the same or exactly opposite way (a plane's normal
// sign is orientation-dependent; either matches the host face).
func sameDir(a, b math.Vector3) bool {
	d := float64(a.AsUnit().Dot(b.AsUnit()))
	return stdmath.Abs(stdmath.Abs(d)-1) < 1e-6
}
