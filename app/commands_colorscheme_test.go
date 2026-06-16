// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestColorSchemeRibbonPanel checks the View tab carries a "Color Scheme" selection box with
// the built-in gallery as its options, and that the active scheme drives the selected index.
func TestColorSchemeRibbonPanel(t *testing.T) {
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab("View")
	if !ok {
		t.Fatal("View tab missing")
	}
	panel, ok := tab.Panel("Color Scheme")
	if !ok || panel.Selector == nil {
		t.Fatalf("View tab has no Color Scheme selector panel")
	}
	if len(panel.Selector.Options) < 2 {
		t.Errorf("Color Scheme selector has %d options, want the built-in gallery", len(panel.Selector.Options))
	}
	// "Default" is active out of the box, so the selector points at its option.
	if got := panel.Selector.Options[panel.Selector.SelectedIndex].Label; got != "Default" {
		t.Errorf("selected option = %q, want Default", got)
	}
}

// TestColorSchemeCommandActivates checks executing a color-scheme command activates the scheme
// and moves the selector's selected index.
func TestColorSchemeCommandActivates(t *testing.T) {
	s := registeredSession(t)
	if err := s.Execute("View.ColorScheme.High Contrast"); err != nil {
		t.Fatalf("execute color-scheme command: %v", err)
	}
	if s.ActiveColorScheme().Name != "High Contrast" {
		t.Errorf("active scheme = %q, want High Contrast", s.ActiveColorScheme().Name)
	}
	// Activating a scheme is an explicit background choice, so the sky/environment image is
	// turned off and the scheme's screen color becomes the viewport background.
	if s.Environment().ShowImage {
		t.Error("activating a color scheme should turn off the environment-image background")
	}
	tab, _ := BuildRibbon(s).Tab("View")
	panel, _ := tab.Panel("Color Scheme")
	if got := panel.Selector.Options[panel.Selector.SelectedIndex].Label; got != "High Contrast" {
		t.Errorf("selector now points at %q, want High Contrast", got)
	}
}
