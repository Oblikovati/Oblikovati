// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
)

// The viewport was always perspective because nothing read the Application Options ▸ Display
// NewWindowProjection setting when a view was made — the option shipped orthographic and was
// simply ignored, so every view fell to doc.ProjectionMode's zero value.

// newDocSession returns a session with one freshly created part document.
func newDocSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if _, err := s.Workspace().Add(doc.Part, "proj.obk", true); err != nil {
		t.Fatalf("Add part: %v", err)
	}
	return s
}

// TestNewDocumentOpensOrthographic is the regression: a brand-new part must open in a parallel
// projection, which is what the shipped display option has always said.
func TestNewDocumentOpensOrthographic(t *testing.T) {
	s := newDocSession(t)

	if got := s.ActiveViewProjection(); got != doc.ProjOrthographic {
		t.Errorf("new document projection = %v, want ProjOrthographic", got)
	}
	if !s.Camera().Orthographic {
		t.Error("the camera handed to the renderer is still a perspective one")
	}
}

// TestNewWindowProjectionOptionDrivesNewDocuments: the setting is the source of the default, not a
// second opinion — choosing perspective must actually give perspective.
func TestNewWindowProjectionOptionDrivesNewDocuments(t *testing.T) {
	s := NewSession()
	s.displayOptions.NewWindowProjection = types.PerspectiveProjection
	if _, err := s.Workspace().Add(doc.Part, "persp.obk", true); err != nil {
		t.Fatalf("Add part: %v", err)
	}

	if got := s.ActiveViewProjection(); got != doc.ProjPerspective {
		t.Errorf("projection = %v, want ProjPerspective — the display option was ignored", got)
	}
	if s.Camera().Orthographic {
		t.Error("camera is orthographic though the option asked for perspective")
	}
}

// TestNewViewFollowsTheProjectionOption: View ▸ New View starts in the configured projection when
// it does not inherit a camera.
func TestNewViewFollowsTheProjectionOption(t *testing.T) {
	s := newDocSession(t)
	s.displayOptions.NewWindowProjection = types.PerspectiveWithOrthoFacesProjection

	i, err := s.AddView(0, "Second", false)
	if err != nil {
		t.Fatalf("AddView: %v", err)
	}
	if got := s.ActiveDocument().Views().All()[i].Projection; got != doc.ProjPerspectiveOrthoFaces {
		t.Errorf("new view projection = %v, want ProjPerspectiveOrthoFaces", got)
	}
}

// TestNewViewCopyingACameraCopiesItsProjection: "copy the active camera" has to include how that
// camera projects, or splitting a view would silently change what the copy shows.
func TestNewViewCopyingACameraCopiesItsProjection(t *testing.T) {
	s := newDocSession(t)
	s.SetActiveViewProjection(doc.ProjPerspective)

	i, err := s.AddView(0, "Copy", true)
	if err != nil {
		t.Fatalf("AddView: %v", err)
	}
	if got := s.ActiveDocument().Views().All()[i].Projection; got != doc.ProjPerspective {
		t.Errorf("copied view projection = %v, want the source view's ProjPerspective", got)
	}
}

// TestOpenedDocumentKeepsItsSavedProjection: the global default applies to NEW documents only. A
// document restored with a saved projection carries the user's per-document choice, which must
// outrank it.
func TestOpenedDocumentKeepsItsSavedProjection(t *testing.T) {
	s := newDocSession(t)
	s.SetActiveViewProjection(doc.ProjPerspective)

	s.applyNewWindowProjection(nil) // a nil document must be a no-op, not a panic
	if got := s.ActiveViewProjection(); got != doc.ProjPerspective {
		t.Errorf("projection = %v, want the explicitly chosen ProjPerspective to survive", got)
	}
}
