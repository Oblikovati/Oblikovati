//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
	"testing"

	"oblikovati.org/app"
)

// TestHoleSeatToggleRoundTrip locks the Seat row's mapping: applying toggle i reads
// back as active index i, and the tool's seat flags stay mutually exclusive.
func TestHoleSeatToggleRoundTrip(t *testing.T) {
	h := app.NewHoleTool()
	for i := 0; i < 3; i++ {
		applyHoleSeat(h, i)
		if got := holeSeatIndex(h); got != i {
			t.Errorf("after applyHoleSeat(%d), index = %d, want %d", i, got, i)
		}
		if h.Counterbore() && h.Countersink() {
			t.Errorf("after applyHoleSeat(%d), both seat flags are set", i)
		}
	}
}

// TestHoleDrillPointToggle locks the Drill Point mapping: flat zeroes the angle, angle
// mode restores the panel's value and falls back to the 118° drill default.
func TestHoleDrillPointToggle(t *testing.T) {
	h := app.NewHoleTool()
	holeUI.pointAngleDeg = 0 // simulate a cleared field: angle mode must restore 118
	applyHoleDrillPoint(h, 1)
	if got := h.PointAngle() * 180 / stdmath.Pi; got < 117.9 || got > 118.1 {
		t.Errorf("angle mode with a zero field set %v°, want the 118° default", got)
	}
	applyHoleDrillPoint(h, 0)
	if h.PointAngle() != 0 {
		t.Errorf("flat mode left point angle %v, want 0", h.PointAngle())
	}
}

// TestToggleRevolveFullRoundTrip locks the full-revolution toggle: entering full mode
// zeroes the swept angle (the tool's full marker); leaving it restores the panel angle.
func TestToggleRevolveFullRoundTrip(t *testing.T) {
	rv := app.NewRevolveTool()
	revolveUI.angleDeg = 180
	if !rv.IsFullRevolution() {
		rv.SetFullRevolution()
	}
	toggleRevolveFull(rv) // leave full mode → 180° from the panel field
	if rv.IsFullRevolution() {
		t.Fatal("toggling out of full mode left the tool in full revolution")
	}
	if got := rv.Angle() * 180 / stdmath.Pi; got < 179.9 || got > 180.1 {
		t.Errorf("leaving full mode set angle %v°, want 180°", got)
	}
	toggleRevolveFull(rv) // back to full
	if !rv.IsFullRevolution() {
		t.Error("toggling into full mode did not set a full revolution")
	}
}

// TestPickChipText locks the single-pick chip captions.
func TestPickChipText(t *testing.T) {
	if got := pickChipText(false, "1 Path", "Select Path"); got != "Select Path" {
		t.Errorf("empty chip text = %q, want \"Select Path\"", got)
	}
	if got := pickChipText(true, "1 Path", "Select Path"); got != "1 Path" {
		t.Errorf("picked chip text = %q, want \"1 Path\"", got)
	}
}

// TestSplitModeToggleRoundTrip locks the Mode row's mapping onto the tool's keep side.
func TestSplitModeToggleRoundTrip(t *testing.T) {
	st := app.NewSplitTool()
	for i, side := range splitModeSides {
		st.SetKeep(side)
		if got := splitModeIndex(st); got != i {
			t.Errorf("keep side %v maps to toggle %d, want %d", side, got, i)
		}
	}
}

// TestCountChipText locks the multi-pick chip captions.
func TestCountChipText(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{{0, "Select Edges"}, {1, "1 Edge"}, {3, "3 Edges"}}
	for _, c := range cases {
		if got := countChipText(c.n, "Edge", "Select Edges"); got != c.want {
			t.Errorf("countChipText(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

// TestLoftGuideChipState locks the Guides chip caption per routing kind.
func TestLoftGuideChipState(t *testing.T) {
	l := app.NewLoftTool()
	l.SetGuideKind(0)
	if text, filled := loftGuideChipState(l); filled || text != "Select Paths" {
		t.Errorf("empty rails chip = (%q, %v), want (\"Select Paths\", false)", text, filled)
	}
	l.SetGuideKind(1)
	if text, filled := loftGuideChipState(l); filled || text != "Select Path" {
		t.Errorf("empty centerline chip = (%q, %v), want (\"Select Path\", false)", text, filled)
	}
}

// TestEditSlotChipText locks the generic editor's slot chip captions: the reference
// sheet's "N Selected" once filled, the slot's prompt while empty.
func TestEditSlotChipText(t *testing.T) {
	if got := editSlotChipText(0, "Edges"); got != "Select Edges" {
		t.Errorf("empty slot chip = %q, want \"Select Edges\"", got)
	}
	if got := editSlotChipText(2, "Edges"); got != "2 Selected" {
		t.Errorf("filled slot chip = %q, want \"2 Selected\"", got)
	}
}
