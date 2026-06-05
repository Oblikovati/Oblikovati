// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// createButton returns the Sketch Create-panel button with the given display name.
func createButton(t *testing.T, s *Session, name string) RibbonButton {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab("Sketch")
	if !ok {
		t.Fatal("no Sketch tab")
	}
	create, ok := tab.Panel("Create")
	if !ok {
		t.Fatal("no Create panel")
	}
	for _, b := range create.Buttons {
		if b.Command.DisplayName() == name {
			return b
		}
	}
	t.Fatalf("Create panel has no %q button", name)
	return RibbonButton{}
}

// The Create-panel heads carry their Inventor variant flyouts as dropdown entries.
func TestCreatePanelHeadsHaveVariants(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	wantVariants := map[string][]string{
		"Rectangle": {"Three Point Rectangle", "Two Point Center Rectangle"},
		"Circle":    {"Three Point Circle"},
		"Arc":       {"Center Point Arc"},
		"Slot":      {"Center Point Arc Slot", "Three Point Arc Slot"},
		"Spline":    {"Control Vertex Spline"},
	}
	for head, want := range wantVariants {
		btn := createButton(t, s, head)
		got := make([]string, len(btn.Variants))
		for i, v := range btn.Variants {
			got[i] = v.Label
		}
		if len(got) != len(want) {
			t.Errorf("%s head has variants %v, want %v", head, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s variant[%d] = %q, want %q", head, i, got[i], want[i])
			}
		}
	}
}

// A variant is registered for id dispatch but never appears as its own Create-panel button.
func TestVariantsDispatchButAreNotPanelButtons(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)

	const variantID = "Sketch.Rectangle.ThreePoint"
	if _, ok := s.Commands().ByID(variantID); !ok {
		t.Fatalf("variant %q not registered for dispatch", variantID)
	}
	if err := s.Execute(variantID); err != nil {
		t.Fatalf("Execute(%q): %v", variantID, err)
	}
	if s.ActiveTool() == nil {
		t.Error("executing a variant should start its tool")
	}

	tab, _ := BuildRibbon(s).Tab("Sketch")
	create, _ := tab.Panel("Create")
	for _, b := range create.Buttons {
		if b.Command.ID() == variantID {
			t.Errorf("variant %q must not be a standalone Create-panel button", variantID)
		}
	}
}

// Variant dropdown entries reflect the head's enable state (disabled outside a sketch).
func TestVariantsEnabledOnlyInSketch(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	for _, v := range createButton(t, s, "Rectangle").Variants {
		if !v.Enabled {
			t.Errorf("variant %q should be enabled inside a sketch", v.Label)
		}
	}
}
