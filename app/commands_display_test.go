// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestDisplaySettingsCommandOpensPanel checks the View-tab Display Settings button is present
// and its command opens the dialog.
func TestDisplaySettingsCommandOpensPanel(t *testing.T) {
	s := registeredSession(t)
	tab, _ := BuildRibbon(s).Tab("View")
	panel, ok := tab.Panel("Appearance")
	if !ok {
		t.Fatal("View tab has no Appearance panel")
	}
	if _, ok := buttonNamed(panel, "Display Settings…"); !ok {
		t.Error("Appearance panel has no Display Settings button")
	}
	if err := s.Execute("View.DisplaySettings"); err != nil {
		t.Fatalf("View.DisplaySettings: %v", err)
	}
	if !s.DisplaySettingsOpen() {
		t.Error("Display Settings command should open the dialog")
	}
	s.CloseDisplaySettings()
	if s.DisplaySettingsOpen() {
		t.Error("dialog should be closed")
	}
}

// TestGroundPlaneToggleCommand checks the View-tab Ground Plane command toggles the active
// document's display-settings ground visibility, end to end through the ribbon command.
func TestGroundPlaneToggleCommand(t *testing.T) {
	s := registeredSession(t)
	if !s.GroundPlaneVisible() {
		t.Fatal("ground plane should start visible by default")
	}
	if err := s.Execute("View.GroundPlane"); err != nil {
		t.Fatalf("View.GroundPlane: %v", err)
	}
	if s.GroundPlaneVisible() {
		t.Error("ground plane should be hidden after one toggle")
	}
	if err := s.Execute("View.GroundPlane"); err != nil {
		t.Fatalf("View.GroundPlane (back on): %v", err)
	}
	if !s.GroundPlaneVisible() {
		t.Error("ground plane should be visible again after the second toggle")
	}
}
