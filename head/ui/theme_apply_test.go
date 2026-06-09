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
	refreshThemeColors(theme.DefaultDark())
	dark := sketchColor
	refreshThemeColors(theme.DefaultLight())
	if sketchColor == dark {
		t.Errorf("sketchColor unchanged after switching Dark->Light (%v); not reading the theme", sketchColor)
	}
	// pointMarkerColor must track sketchColor (placed points match the wireframe).
	if pointMarkerColor != sketchColor {
		t.Errorf("pointMarkerColor %v != sketchColor %v", pointMarkerColor, sketchColor)
	}
	refreshThemeColors(theme.DefaultDark()) // restore the package default for other tests
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
