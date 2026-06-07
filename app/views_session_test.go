// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/api/types"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/doc"
)

func TestSetViewLayoutCreatesAndKeepsViews(t *testing.T) {
	s := NewSession()
	a := addPart(t, s, "a.obk")
	activate(t, s, a)

	if err := s.SetViewLayout(types.LayoutFour); err != nil {
		t.Fatalf("SetViewLayout(Four): %v", err)
	}
	if got := a.Views().Count(); got < 4 {
		t.Fatalf("after Four layout, view count = %d, want >= 4", got)
	}
	if a.Views().Layout() != types.LayoutFour {
		t.Errorf("layout = %v, want four", a.Views().Layout())
	}
	// Switching to a smaller layout keeps the extra views for later.
	if err := s.SetViewLayout(types.LayoutSingle); err != nil {
		t.Fatalf("SetViewLayout(Single): %v", err)
	}
	if a.Views().Count() < 4 {
		t.Errorf("single layout dropped views: count = %d, want >= 4", a.Views().Count())
	}
}

func addPart(t *testing.T, s *Session, name string) *doc.Document {
	t.Helper()
	d, err := s.Workspace().Add(doc.Part, name, true)
	if err != nil {
		t.Fatalf("add %s: %v", name, err)
	}
	d.SetContent(compdef.NewPartComponentDefinition())
	return d
}

func activate(t *testing.T, s *Session, d *doc.Document) {
	t.Helper()
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		t.Fatalf("activate %s: %v", d.DisplayName(), err)
	}
}

// TestCameraIsPerDocumentAcrossSwitch is the regression for the reported bug: switching
// the active document must restore that document's camera, not reset it.
func TestCameraIsPerDocumentAcrossSwitch(t *testing.T) {
	s := NewSession()
	a := addPart(t, s, "a.obk")
	b := addPart(t, s, "b.obk")

	activate(t, s, a)
	ca := s.Camera()
	ca.Eye = math.P3(11, 0, 0)
	s.SetCamera(ca)

	activate(t, s, b)
	cb := s.Camera()
	cb.Eye = math.P3(0, 22, 0)
	s.SetCamera(cb)

	activate(t, s, a)
	if got := s.Camera().Eye; got != math.P3(11, 0, 0) {
		t.Fatalf("doc A camera not restored after switch: eye=%v, want (11,0,0)", got)
	}
	activate(t, s, b)
	if got := s.Camera().Eye; got != math.P3(0, 22, 0) {
		t.Fatalf("doc B camera not restored after switch: eye=%v, want (0,22,0)", got)
	}
}

func TestAddViewAndDocumentByID(t *testing.T) {
	s := NewSession()
	a := addPart(t, s, "a.obk")
	activate(t, s, a)

	// Distinct active-view camera, then a copied view should start at it.
	ca := s.Camera()
	ca.Eye = math.P3(5, 6, 7)
	s.SetCamera(ca)
	i, err := s.AddView(0, "Iso", true)
	if err != nil || i != 1 {
		t.Fatalf("AddView: i=%d err=%v", i, err)
	}
	if got := a.Views().Active().Eye; got != math.P3(5, 6, 7) {
		t.Errorf("copied view eye = %v, want (5,6,7)", got)
	}
	// DocumentByID(0) is the active document; an unknown id errors.
	if d, err := s.DocumentByID(0); err != nil || d != a {
		t.Errorf("DocumentByID(0) = %v,%v want active doc", d, err)
	}
	if _, err := s.DocumentByID(999999); err == nil {
		t.Error("DocumentByID(unknown) should error")
	}
}
