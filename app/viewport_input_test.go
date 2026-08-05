// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"
)

// The wire's synthesised input (viewport.click / viewport.key) resolves its button, modifiers and
// editing keys through these, so each mapping is pinned here rather than only through the router.

// TestPointerButtonNamed covers every accepted spelling and the rejection.
func TestPointerButtonNamed(t *testing.T) {
	for name, want := range map[string]PointerButton{
		"":       LeftButton, // omitted ⇒ the only button that reaches a tool
		"left":   LeftButton,
		"right":  RightButton,
		"middle": MiddleButton,
	} {
		got, err := PointerButtonNamed(name)
		if err != nil {
			t.Errorf("PointerButtonNamed(%q): %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("PointerButtonNamed(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestPointerButtonNamedRejectsATypo: the error must name the offending value and the accepted
// ones, or a caller cannot tell a typo from an unsupported button.
func TestPointerButtonNamedRejectsATypo(t *testing.T) {
	_, err := PointerButtonNamed("centre")
	if err == nil {
		t.Fatal("an unknown button should be rejected")
	}
	if !strings.Contains(err.Error(), "centre") || !strings.Contains(err.Error(), "middle") {
		t.Errorf("error %q should name the offending value and the accepted ones", err)
	}
}

// TestModifierForPacksEveryHeldKey: the selection and snapping paths read this mask, so a dropped
// bit silently changes what a click does (extend vs replace the selection).
func TestModifierForPacksEveryHeldKey(t *testing.T) {
	if got := ModifierFor(false, false, false); got != 0 {
		t.Errorf("no modifiers held = %v, want 0", got)
	}
	all := ModifierFor(true, true, true)
	for _, m := range []Modifier{ShiftMod, CtrlMod, AltMod} {
		if !all.Has(m) {
			t.Errorf("all-held mask %v is missing %v", all, m)
		}
	}
	if only := ModifierFor(false, true, false); only != CtrlMod {
		t.Errorf("ctrl only = %v, want just CtrlMod", only)
	}
}

// TestPlacementEditKeyDrivesTheBoxes: the wire's equivalent of the head's per-frame editing-key
// read. Without it a client could type a value into an in-place box but never LOCK it.
func TestPlacementEditKeyDrivesTheBoxes(t *testing.T) {
	s := placingRectangle(t)
	for _, r := range "25" {
		s.PlacementFieldInput(r)
	}

	if !s.placementEditKey("Tab") {
		t.Fatal("Tab was not consumed while placing")
	}

	fields := s.PlacementFields()
	if !fields[0].Locked {
		t.Error("Tab did not lock the active box")
	}
	if !fields[1].Active {
		t.Error("Tab did not advance to the next box")
	}
}

// TestPlacementEditKeyBackspaceEdits: a mistyped value must be correctable without cancelling the
// shape, which is Backspace's whole job here.
func TestPlacementEditKeyBackspaceEdits(t *testing.T) {
	s := placingRectangle(t)
	for _, r := range "25" {
		s.PlacementFieldInput(r)
	}

	if !s.placementEditKey("Backspace") {
		t.Fatal("Backspace was not consumed while placing")
	}
	if got := s.PlacementFields()[0].Value; got != "2" {
		t.Errorf("after Backspace the box holds %q, want %q", got, "2")
	}
}

// TestPlacementEditKeyIgnoresOtherKeys: only the editing keys are claimed. Escape must fall
// through to the tool's own cancel rather than being swallowed here.
func TestPlacementEditKeyIgnoresOtherKeys(t *testing.T) {
	s := placingRectangle(t)

	if s.placementEditKey("Escape") {
		t.Error("Escape was swallowed; it must reach the tool's cancel")
	}
	if s.placementEditKey("q") {
		t.Error("an ordinary key was claimed as an editing key")
	}
}

// TestPlacementEditKeyOnlyWhilePlacing: with no placement under way the keys belong to whatever
// else is running, so nothing may be consumed.
func TestPlacementEditKeyOnlyWhilePlacing(t *testing.T) {
	s, _ := sketchSession(t)

	if s.placementEditKey("Tab") {
		t.Error("Tab was consumed with no shape being placed")
	}
}
