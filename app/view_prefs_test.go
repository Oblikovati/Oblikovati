// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// TestSessionViewPreferences covers the view/ViewCube preference accessors and the no-document
// guards in app/views.go: each preference round-trips, and the active-view operations are safe
// no-ops when no document is open.
func TestSessionViewPreferences(t *testing.T) {
	s := NewSession()

	s.SetLockToSelection(true)
	if !s.LockToSelection() {
		t.Error("LockToSelection did not round-trip")
	}

	s.SetInactiveOpacity(0.4)
	if got := s.InactiveOpacity(); got <= 0 || got > 1 {
		t.Errorf("InactiveOpacity = %g, want in (0,1]", got)
	}

	s.SetCubeSize(72)
	if s.CubeSize() <= 0 {
		t.Errorf("CubeSize = %g, want > 0", s.CubeSize())
	}

	s.SetCubeCorner(2)
	if s.CubeCorner() != 2 {
		t.Errorf("CubeCorner = %d, want 2", s.CubeCorner())
	}

	_ = s.CubeOrientation()
	_ = s.ViewCubePivot() // no selection: falls back to a default pivot

	// No document open: the active-view operations must be safe no-ops.
	s.SetActiveViewProjection(doc.ProjOrthographic)
	if s.ActiveViewProjection() != doc.ProjPerspective {
		t.Errorf("ActiveViewProjection with no view = %v, want the perspective default", s.ActiveViewProjection())
	}
	s.SetActiveViewAsFront()
	s.ResetFront()
}

// TestSessionViewManagement covers the per-document view/camera management (NewView,
// CloseActiveView, ViewCamera/SetViewCamera, ViewCameraAt/SetViewCameraAt) against an open part.
func TestSessionViewManagement(t *testing.T) {
	s := NewSession()
	part, err := compdef.AddPart(s.Workspace(), "v.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(part); err != nil {
		t.Fatalf("SetActiveDocument: %v", err)
	}

	cam, err := s.ViewCamera(0) // 0 = active document
	if err != nil {
		t.Fatalf("ViewCamera: %v", err)
	}
	if err := s.SetViewCamera(0, cam); err != nil {
		t.Errorf("SetViewCamera: %v", err)
	}
	camAt, ok := s.ViewCameraAt(0) // scene.Camera (the tiled-viewport accessor the head uses)
	if !ok {
		t.Error("ViewCameraAt(0) should resolve the first open document's camera")
	}
	s.SetViewCameraAt(0, camAt)

	if err := s.NewView(); err != nil {
		t.Errorf("NewView: %v", err)
	}
	if err := s.CloseActiveView(); err != nil {
		t.Errorf("CloseActiveView: %v", err)
	}
}

// TestFaceAligned covers the axis-aligned view test used by the ViewCube snapping.
func TestFaceAligned(t *testing.T) {
	if !faceAligned(math.P3(0, 0, 10), math.P3(0, 0, 0)) {
		t.Error("a straight-down-Z view should be face-aligned")
	}
	if faceAligned(math.P3(1, 2, 3), math.P3(0, 0, 0)) {
		t.Error("an oblique view should not be face-aligned")
	}
}
