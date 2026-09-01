// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestRelationshipsPanelExposesEveryConstraint: the Assemble tab's Relationships panel exposes
// one command per M12-F01 constraint kind (#770), each a compact icon button.
func TestRelationshipsPanelExposesEveryConstraint(t *testing.T) {
	t.Parallel()
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("an active assembly should show the Assemble tab")
	}
	panel, ok := tab.Panel("Relationships")
	if !ok {
		t.Fatal("Assemble tab has no Relationships panel")
	}
	want := []string{
		"Mate", "Flush", "Angle", "Tangent", "Insert", "Symmetry",
		"Rotate-Rotate", "Rotate-Translate", "Translate-Translate", "Transitional", "Custom",
	}
	for _, name := range want {
		if !hasButton(panel, name) {
			t.Errorf("Relationships panel is missing the %q command", name)
		}
		if got, ok := styleOf(panel, name); !ok || got != CompactIconButton {
			t.Errorf("%q button style = %v, want CompactIconButton", name, got)
		}
	}
}

// TestEveryAssembleRibbonCommandHasIcon enforces the requirement that every command shown on
// the Assemble ribbon carries an icon (so the bar never renders a blank button) — Place was
// the lone exception until M12.
func TestEveryAssembleRibbonCommandHasIcon(t *testing.T) {
	t.Parallel()
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("an active assembly should show the Assemble tab")
	}
	for _, panel := range tab.Panels {
		for _, b := range panel.Buttons {
			if b.Command.Icon() == "" {
				t.Errorf("Assemble command %q (panel %q) has no icon", b.Command.ID(), panel.Name)
			}
		}
	}
}
