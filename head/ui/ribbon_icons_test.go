//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/icon"
)

// TestRibbonCommandIconsResolve guards that every standard ribbon command which names an
// icon has a bundled SVG asset for it. This pins the fix for "blank / text-only buttons":
// a command with WithIcon("foo") but no foo.svg falls back to a text button, so a typo'd
// or unbundled key now fails the build instead of shipping a broken-looking ribbon.
func TestRibbonCommandIconsResolve(t *testing.T) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	for _, c := range s.Commands().All() {
		key := c.Icon()
		if key == "" {
			continue // text-only by design (e.g. visual-style / gallery combo options)
		}
		if _, ok := icon.SVG(key); !ok {
			t.Errorf("command %q references icon %q with no bundled asset (head/icon/assets/%s.svg)", c.ID(), key, key)
		}
	}
}

// TestRibbonIconCommandsHaveButtonStyle guards that a command which names an icon also sets
// a (small/large) button style — the renderer only draws the glyph when both are present, so
// an icon with the default text-only style silently renders as a text button (the bug that
// left the 3D Sketch tab icon-less).
func TestRibbonIconCommandsHaveButtonStyle(t *testing.T) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	for _, c := range s.Commands().All() {
		if c.Icon() != "" && c.ButtonStyle() == types.TextOnlyButton {
			t.Errorf("command %q sets icon %q but a text-only button style — the glyph will not render; add WithButtonStyle", c.ID(), c.Icon())
		}
	}
}
