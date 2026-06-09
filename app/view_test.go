// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// extrudedBoxSession builds a part with a real solid via the synthetic-input extrude
// flow (square profile → distance → OK), returning the session for view-command tests.
func extrudedBoxSession(t *testing.T, side, depth float64) *Session {
	t.Helper()
	s, profile := newPartWithSquare(t, side)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(120, 90) // picks the profile
	ext.SetDistance(depth)
	if err := s.OK(); err != nil {
		t.Fatalf("extrude OK: %v", err)
	}
	return s
}

func TestFitViewFramesModelFromAnywhere(t *testing.T) {
	s := extrudedBoxSession(t, 4, 5)

	// Start from a camera pointed away from the model, then Zoom All.
	cam := scene.NewCamera(800, 600)
	cam.Eye = math.P3(0, 0, 500)
	cam.Target = math.P3(0, 0, 490) // looking −Z, far from the box near the origin
	s.SetCamera(cam)
	s.FitView()

	got := s.Camera()
	box := s.modelBounds()
	if box.IsEmpty() {
		t.Fatal("model bounds empty — extrude produced no body")
	}
	if !got.Target.IsEqualTo(box.Center(), 1e-9) {
		t.Errorf("after Zoom All target = %v, want model center %v", got.Target, box.Center())
	}
	for _, corner := range box.Corners() {
		if v := got.Eye.VectorTo(corner); float64(v.Dot(got.Forward())) <= 0 {
			t.Errorf("model corner %v is behind the camera after Zoom All", corner)
		}
	}
}

func TestHomeViewIsIsometricAndFramesModel(t *testing.T) {
	s := extrudedBoxSession(t, 4, 5)
	s.SetCamera(scene.NewCamera(800, 600))
	s.HomeView()

	got := s.Camera()
	if !got.Up.IsEqualTo(math.V3(0, 1, 0), 1e-9) {
		t.Errorf("home Up = %v, want +Y", got.Up)
	}
	if !got.Forward().IsEqualTo(unitV(math.V3(-1, -1, -1)), 1e-9) {
		t.Errorf("home forward = %v, want iso (−1,−1,−1)", got.Forward())
	}
	if !got.Target.IsEqualTo(s.modelBounds().Center(), 1e-9) {
		t.Error("home should target the model center")
	}
}

// TestZoomAllCommandReframesViaExecute exercises the full ribbon path a button uses:
// a registered "Zoom All" command, run through Session.Execute, reframes the view.
func TestZoomAllCommandReframesViaExecute(t *testing.T) {
	s := extrudedBoxSession(t, 4, 5)
	if err := s.Commands().Add(NewCommand("View.ZoomAll", "Zoom All", "Navigate",
		func(sess *Session) error { sess.FitView(); return nil })); err != nil {
		t.Fatalf("register Zoom All: %v", err)
	}
	cam := scene.NewCamera(800, 600)
	cam.Eye = math.P3(0, 0, 500)
	cam.Target = math.P3(0, 0, 490)
	s.SetCamera(cam)

	if err := s.Execute("View.ZoomAll"); err != nil {
		t.Fatalf("execute Zoom All: %v", err)
	}
	if !s.Camera().Target.IsEqualTo(s.modelBounds().Center(), 1e-9) {
		t.Error("running the Zoom All command did not reframe the view to the model center")
	}
}

func TestFitViewEmptyModelIsNoOp(t *testing.T) {
	s, _ := emptyPartSession(t)
	cam := scene.NewCamera(640, 480)
	s.SetCamera(cam)
	s.FitView()
	if s.Camera() != cam {
		t.Error("Zoom All on an empty model should leave the camera unchanged")
	}
}

// unitV normalizes a vector for the test's expected-direction comparison.
func unitV(v math.Vector3) math.Vector3 { return v.Scale(1 / float64(v.Length())) }
