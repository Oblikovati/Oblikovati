// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestColorStylesCommandOpensPanel checks the View-tab Color Styles button is present and its
// command opens the panel.
func TestColorStylesCommandOpensPanel(t *testing.T) {
	s := registeredSession(t)
	tab, _ := BuildRibbon(s).Tab("View")
	panel, ok := tab.Panel("Appearance")
	if !ok {
		t.Fatal("View tab has no Appearance panel")
	}
	if _, ok := buttonNamed(panel, "Color Styles…"); !ok {
		t.Error("Appearance panel has no Color Styles button")
	}
	if err := s.Execute("View.ColorStyles"); err != nil {
		t.Fatalf("View.ColorStyles: %v", err)
	}
	if !s.ColorStylesPanelOpen() {
		t.Error("Color Styles command should open the panel")
	}
}

// TestSelectedBodyKeyEmptyWithoutSelection checks no body key when nothing is selected.
func TestSelectedBodyKeyEmptyWithoutSelection(t *testing.T) {
	s := NewSession()
	if _, ok := s.SelectedBodyKey(); ok {
		t.Error("no body should be selected on a fresh session")
	}
}
