//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// showPreferences toggles the Preferences window (Tools ▸ Preferences). It is UI state,
// not model state, so it lives here in the head.
var showPreferences bool

// drawPreferencesWindow renders the Preferences window when open: the sketch-grid
// spacing (in the document's units), visibility, and major-line interval. Edits write
// straight back to the session's settings, so the grid updates next frame.
func drawPreferencesWindow(s *app.Session) {
	if !showPreferences {
		return
	}
	if native.Begin("Preferences") {
		native.SeparatorText("Sketch Grid")
		editGridSpacing(s)
		editGridVisibility(s)
		editGridMajor(s)
	}
	native.End()
}

// editGridSpacing draws the spacing field labeled with the document's length unit.
func editGridSpacing(s *app.Session) {
	value, unit := s.GridSpacingDisplay()
	v := float32(value)
	if native.InputFloat("Grid spacing ("+unit+")", &v) {
		_ = s.SetGridSpacingDisplay(float64(v)) // ignore non-positive entries
	}
}

// editGridVisibility draws the show-grid checkbox.
func editGridVisibility(s *app.Session) {
	vis := s.Grid().Visible
	if native.Checkbox("Show grid in sketch", &vis) {
		s.Grid().Visible = vis
	}
}

// editGridMajor draws the major-line interval field.
func editGridMajor(s *app.Session) {
	major := int32(s.Grid().MajorEvery)
	if native.InputInt("Major line every N", &major) && major >= 1 {
		s.Grid().MajorEvery = int(major)
	}
}
