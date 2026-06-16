// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestOrientRibbonButton checks the View tab's Navigate panel carries the Orient split button
// with a dropdown of the standard orientations.
func TestOrientRibbonButton(t *testing.T) {
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab("View")
	if !ok {
		t.Fatal("View tab missing")
	}
	nav, ok := tab.Panel("Navigate")
	if !ok {
		t.Fatal("View tab has no Navigate panel")
	}
	btn, ok := buttonNamed(nav, "Orient")
	if !ok {
		t.Fatal("Navigate panel has no Orient button")
	}
	if len(btn.Variants) < 6 {
		t.Errorf("Orient has %d variants, want the standard orientations", len(btn.Variants))
	}
}

// TestOrientCommandsMoveCamera checks the front and iso orientation commands move the camera
// onto the expected side of the model.
func TestOrientCommandsMoveCamera(t *testing.T) {
	s := registeredSession(t)
	if err := s.Execute("View.Orient.Front"); err != nil {
		t.Fatalf("View.Orient.Front: %v", err)
	}
	front := s.Camera()
	if !(front.Eye.Z > front.Target.Z) {
		t.Errorf("front eye %v not in front (+Z) of target %v", front.Eye, front.Target)
	}
	if err := s.Execute("View.Orient.Iso"); err != nil {
		t.Fatalf("View.Orient.Iso: %v", err)
	}
	iso := s.Camera()
	if !(iso.Eye.X > iso.Target.X && iso.Eye.Y > iso.Target.Y && iso.Eye.Z > iso.Target.Z) {
		t.Errorf("iso eye %v not on +++ octant of target %v", iso.Eye, iso.Target)
	}
}

// TestNamedViewsCommandOpensPanel checks the View-tab Named Views button is present and its
// command opens the panel.
func TestNamedViewsCommandOpensPanel(t *testing.T) {
	s := registeredSession(t)
	tab, _ := BuildRibbon(s).Tab("View")
	nav, ok := tab.Panel("Navigate")
	if !ok {
		t.Fatal("View tab has no Navigate panel")
	}
	if _, ok := buttonNamed(nav, "Named Views…"); !ok {
		t.Error("Navigate panel has no Named Views button")
	}
	if s.NamedViewsPanelOpen() {
		t.Fatal("panel should start closed")
	}
	if err := s.Execute("View.NamedViews"); err != nil {
		t.Fatalf("View.NamedViews: %v", err)
	}
	if !s.NamedViewsPanelOpen() {
		t.Error("Named Views command should open the panel")
	}
}

// buttonNamed returns the named button of a panel.
func buttonNamed(p RibbonPanel, name string) (RibbonButton, bool) {
	for _, b := range p.Buttons {
		if b.Command.DisplayName() == name {
			return b, true
		}
	}
	return RibbonButton{}, false
}
