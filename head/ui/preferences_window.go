//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// showPreferences toggles the Preferences window (Tools ▸ Preferences). It is UI state,
// not model state, so it lives here in the head.
var showPreferences bool

// drawPreferencesWindow renders the Preferences window when open, as tabs: Sketch Grid
// (spacing in the document's units, visibility, major-line interval) and Appearance (the
// theme picker + editor). Edits write straight back to the session, so the grid and theme
// update next frame.
func drawPreferencesWindow(s *app.Session) {
	if !showPreferences {
		return
	}
	native.SetNextWindowSizeOnce(640, 480) // sensible default; the user can still resize
	if native.Begin("Preferences") && native.BeginTabBar("##prefs-tabs") {
		if native.BeginTabItem("Sketch Grid") {
			drawGridTab(s)
			native.EndTabItem()
		}
		if native.BeginTabItem("Modeling") {
			drawModelingTab(s)
			native.EndTabItem()
		}
		if native.BeginTabItem("Theme") {
			drawAppearanceTab(s)
			native.EndTabItem()
		}
		native.EndTabBar()
	}
	native.End()
}

// drawModelingTab renders modeling-feature preferences (the default chamfer corner
// treatment for now).
func drawModelingTab(s *app.Session) {
	flat := s.ChamferFlatCorners()
	if native.Checkbox("Chamfer: flat triangular face at 3-edge corners", &flat) {
		s.SetChamferFlatCorners(flat)
	}
}

// drawGridTab renders the sketch-grid preference fields.
func drawGridTab(s *app.Session) {
	editGridSpacing(s)
	editGridVisibility(s)
	editGridMajor(s)
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
