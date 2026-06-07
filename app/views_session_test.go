// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/doc"
)

func addPart(t *testing.T, s *Session, name string) *doc.Document {
	t.Helper()
	d, err := s.Workspace().Add(doc.Part, name, true)
	if err != nil {
		t.Fatalf("add %s: %v", name, err)
	}
	d.SetContent(compdef.NewPartComponentDefinition())
	return d
}

// TestCameraIsPerDocumentAcrossSwitch is the regression for the reported bug: switching
// the active document must restore that document's camera, not reset it.
func TestCameraIsPerDocumentAcrossSwitch(t *testing.T) {
	s := NewSession()
	a := addPart(t, s, "a.obk")
	b := addPart(t, s, "b.obk")

	s.Workspace().SetActiveDocument(a)
	ca := s.Camera()
	ca.Eye = math.P3(11, 0, 0)
	s.SetCamera(ca)

	s.Workspace().SetActiveDocument(b)
	cb := s.Camera()
	cb.Eye = math.P3(0, 22, 0)
	s.SetCamera(cb)

	s.Workspace().SetActiveDocument(a)
	if got := s.Camera().Eye; got != math.P3(11, 0, 0) {
		t.Fatalf("doc A camera not restored after switch: eye=%v, want (11,0,0)", got)
	}
	s.Workspace().SetActiveDocument(b)
	if got := s.Camera().Eye; got != math.P3(0, 22, 0) {
		t.Fatalf("doc B camera not restored after switch: eye=%v, want (0,22,0)", got)
	}
}

func TestAddViewAndDocumentByID(t *testing.T) {
	s := NewSession()
	a := addPart(t, s, "a.obk")
	s.Workspace().SetActiveDocument(a)

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
