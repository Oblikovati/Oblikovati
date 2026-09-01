// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	s := newDocSession(t)
	s.SetActiveViewProjection(doc.ProjPerspective)

	s.applyNewWindowProjection(nil) // a nil document must be a no-op, not a panic
	if got := s.ActiveViewProjection(); got != doc.ProjPerspective {
		t.Errorf("projection = %v, want the explicitly chosen ProjPerspective to survive", got)
	}
}

// TestViewProjectionRoundTripsOverTheWireEnum: a client capturing a viewport has to be able to
// tell whether what it sees is foreshortened, and to change it. Before this the projection was
// reachable only as the global new-window default.
func TestViewProjectionRoundTripsOverTheWireEnum(t *testing.T) {
	t.Parallel()
	s := newDocSession(t)

	got, err := s.ViewProjection(0)
	if err != nil {
		t.Fatalf("ViewProjection: %v", err)
	}
	if got != types.OrthographicProjection {
		t.Errorf("projection = %v, want OrthographicProjection", got)
	}
	if err := s.SetViewProjection(0, types.PerspectiveProjection); err != nil {
		t.Fatalf("SetViewProjection: %v", err)
	}
	if got, _ := s.ViewProjection(0); got != types.PerspectiveProjection {
		t.Errorf("after setting perspective, projection = %v", got)
	}
	if s.Camera().Orthographic {
		t.Error("the rendered camera did not follow the projection change")
	}
}

// TestSetViewProjectionRejectsAnUnknownEnum: an out-of-vocabulary value must be reported with the
// offending value and the values it could have been, not silently coerced to a default.
func TestSetViewProjectionRejectsAnUnknownEnum(t *testing.T) {
	t.Parallel()
	s := newDocSession(t)

	err := s.SetViewProjection(0, types.ProjectionTypeEnum(42))
	if err == nil {
		t.Fatal("an unknown projection enum should be rejected")
	}
	if !strings.Contains(err.Error(), "42") || !strings.Contains(err.Error(), "86273") {
		t.Errorf("error %q should name the offending value and the accepted ones", err)
	}
}
