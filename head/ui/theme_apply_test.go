//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/theme"
)

// TestChromeBindingSlotsUnique guards the apply contract that each Dear ImGui color slot
// is driven by exactly one token — overlap would make the last-applied token silently win
// and the result order-dependent.
func TestChromeBindingSlotsUnique(t *testing.T) {
	seen := map[string]types.ThemeToken{}
	for tok, slots := range chromeBinding {
		for _, slot := range slots {
			if other, dup := seen[slot]; dup {
				t.Errorf("ImGui slot %q bound by both %q and %q", slot, other, tok)
			}
			seen[slot] = tok
		}
	}
}

// TestRefreshThemeColorsReadsActiveTheme proves the overlay color vars are sourced from
// the active theme, not frozen at their seeded Dark values: switching to Light must change
// the sketch-geometry color.
func TestRefreshThemeColorsReadsActiveTheme(t *testing.T) {
	chromeTheme.refresh(theme.DefaultDark())
	dark := chromeTheme.sketchColor
	chromeTheme.refresh(theme.DefaultLight())
	if chromeTheme.sketchColor == dark {
		t.Errorf("chromeTheme.sketchColor unchanged after switching Dark->Light (%v); not reading the theme", chromeTheme.sketchColor)
	}
	// chromeTheme.pointMarkerColor must track chromeTheme.sketchColor (placed points match the wireframe).
	if chromeTheme.pointMarkerColor != chromeTheme.sketchColor {
		t.Errorf("chromeTheme.pointMarkerColor %v != chromeTheme.sketchColor %v", chromeTheme.pointMarkerColor, chromeTheme.sketchColor)
	}
	chromeTheme.refresh(theme.DefaultDark()) // restore the package default for other tests
}

// TestInputTextCursorBoundToText guards the light-mode caret fix: ImGui 1.92 draws the
// InputText caret with the dedicated ImGuiCol_InputTextCursor (not ImGuiCol_Text), so it
// must be remapped with the text color — otherwise it keeps the dark-theme default and is
// invisible on a light background.
func TestInputTextCursorBoundToText(t *testing.T) {
	bound := false
	for _, slot := range chromeBinding[types.TokenChromeText] {
		if slot == "InputTextCursor" {
			bound = true
		}
	}
	if !bound {
		t.Error("InputTextCursor must be bound to TokenChromeText so the caret is visible in light themes")
	}
}

func TestBufString(t *testing.T) {
	buf := make([]byte, 16)
	copy(buf, "My Dark\x00garbage")
	if got := bufString(buf); got != "My Dark" {
		t.Errorf("bufString = %q, want \"My Dark\"", got)
	}
}
