// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// hasButton reports whether a ribbon panel contains a button with the given display name.
func hasButton(p RibbonPanel, name string) bool {
	for _, b := range p.Buttons {
		if b.Command.DisplayName() == name {
			return true
		}
	}
	return false
}

// TestSketchTabPanelsMatchInventor locks the Sketch tab's panel placement to the canonical
// ribbon (architecture/mapping/inventor-ribbon-structure.md): Fillet in Create, Mirror in
// Pattern, Dimension/Auto Dimension in Constrain, and NO standalone "Dimension" panel.
func TestSketchTabPanelsMatchInventor(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	tab, ok := BuildRibbon(s).Tab("Sketch")
	if !ok {
		t.Fatal("no Sketch tab")
	}

	create, ok := tab.Panel("Create")
	if !ok || !hasButton(create, "Fillet") {
		t.Error("Sketch Create panel should contain Fillet")
	}

	modify, ok := tab.Panel("Modify")
	if !ok {
		t.Fatal("no Modify panel")
	}
	if hasButton(modify, "Mirror") || hasButton(modify, "Fillet") {
		t.Error("Modify panel must not contain Mirror or Fillet (moved to Pattern/Create)")
	}
	for _, want := range []string{"Move", "Copy", "Rotate", "Scale", "Stretch", "Offset", "Trim", "Extend", "Split"} {
		if !hasButton(modify, want) {
			t.Errorf("Modify panel missing %q", want)
		}
	}

	pattern, ok := tab.Panel("Pattern")
	if !ok {
		t.Fatal("no Pattern panel")
	}
	for _, want := range []string{"Rectangular", "Circular", "Mirror"} {
		if !hasButton(pattern, want) {
			t.Errorf("Pattern panel missing %q", want)
		}
	}

	constrain, ok := tab.Panel("Constrain")
	if !ok || !hasButton(constrain, "Dimension") || !hasButton(constrain, "Auto Dimension") {
		t.Error("Constrain panel should contain Dimension and Auto Dimension")
	}

	if _, exists := tab.Panel("Dimension"); exists {
		t.Error("there must be no standalone 'Dimension' panel (Inventor has none)")
	}
}
