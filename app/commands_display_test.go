// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

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
