//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"slices"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
)

// slotToken reverses chromeBinding to report which theme token drives a given Dear ImGui color slot.
func slotToken(slot string) (types.ThemeToken, bool) {
	for tok, slots := range chromeBinding {
		if slices.Contains(slots, slot) {
			return tok, true
		}
	}
	var none types.ThemeToken
	return none, false
}

// TestScrollbarGrabContrastsWithTrack is the regression guard for the #1471 follow-up: the scrollbar
// GRAB (draggable slider) and its TRACK background must be driven by different theme tokens, and
// resolve to visibly different colors — otherwise the grab is invisible against its track until you
// drag it (ScrollbarGrabActive, the bright accent, is the only state that shows). That was the bug:
// ScrollbarBg and ScrollbarGrab shared one token, so both painted the same color.
func TestScrollbarGrabContrastsWithTrack(t *testing.T) {
	bgTok, okBg := slotToken("ScrollbarBg")
	grabTok, okGrab := slotToken("ScrollbarGrab")
	if !okBg || !okGrab {
		t.Fatal("ScrollbarBg and ScrollbarGrab must both be bound in chromeBinding")
	}
	if bgTok == grabTok {
		t.Fatalf("ScrollbarBg and ScrollbarGrab share token %v — the grab is invisible against its "+
			"track until dragged (#1471 follow-up)", bgTok)
	}
	th := app.NewSession().Theme()
	bg := th.Color(bgTok).Array()
	grab := th.Color(grabTok).Array()
	if rgbDistance(bg, grab) < 0.12 { // perceptible at rest, not just while dragging
		t.Errorf("scrollbar track %v and grab %v are nearly identical (Δ=%.3f) — grab not visible at rest",
			bg, grab, rgbDistance(bg, grab))
	}
}

// rgbDistance is the summed absolute RGB difference of two RGBA colors (alpha ignored).
func rgbDistance(a, b [4]float32) float32 {
	var d float32
	for i := range 3 {
		if a[i] > b[i] {
			d += a[i] - b[i]
		} else {
			d += b[i] - a[i]
		}
	}
	return d
}
