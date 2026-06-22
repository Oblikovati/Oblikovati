//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
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
		drawPreferencesTabs(s)
		native.EndTabBar()
	}
	native.End()
}

// drawPreferencesTabs mounts each preference tab in order. Kept separate from the window
// open/close so the tab list reads as one table and the window function stays small.
func drawPreferencesTabs(s *app.Session) {
	tabs := []struct {
		name string
		draw func(*app.Session)
	}{
		{"General", drawGeneralTab},
		{"UI", drawUITab},
		{"Privacy", drawPrivacyTab},
		{"Sketch Grid", drawGridTab},
		{"Modeling", drawModelingTab},
		{"Theme", drawAppearanceTab},
	}
	for _, t := range tabs {
		if native.BeginTabItem(t.name) {
			t.draw(s)
			native.EndTabItem()
		}
	}
}

// drawPrivacyTab renders the anonymous usage-statistics opt-out (#1182). Telemetry is on by
// default; unchecking it stops the startup snapshot upload to stats.oblikovati.org.
func drawPrivacyTab(s *app.Session) {
	native.Text("Oblikovati can share anonymous usage statistics to guide development.")
	native.Text("No personal data is collected — only OS, hardware, version and installed add-ins.")
	native.Separator()
	share := s.TelemetryEnabled()
	if native.Checkbox("Share anonymous usage statistics", &share) {
		reportPrefError(s.SetTelemetryEnabled(share))
	}
}

// drawModelingTab renders modeling-feature preferences (the default chamfer corner
// treatment for now).
func drawModelingTab(s *app.Session) {
	flat := s.ChamferFlatCorners()
	if native.Checkbox("Chamfer: flat triangular face at 3-edge corners", &flat) {
		s.SetChamferFlatCorners(flat)
		persistOptions(s)
	}
}

// drawGeneralTab renders application-level options: what opens at startup (M05-F11).
func drawGeneralTab(s *app.Session) {
	current := s.Options().General.StartupAction
	native.Text("On startup")
	if native.BeginCombo("##startup-action", startupActionLabel(current)) {
		for _, action := range []types.StartupActionType{types.StartupNewPart, types.StartupEmptyWorkspace} {
			if native.Selectable(startupActionLabel(action), action == current) && action != current {
				general := s.Options().General
				general.StartupAction = action
				reportPrefError(s.SetGeneralOptions(general))
			}
		}
		native.EndCombo()
	}
}

// startupActionLabel is the user-facing name of a startup action.
func startupActionLabel(a types.StartupActionType) string {
	if a == types.StartupEmptyWorkspace {
		return "Empty workspace"
	}
	return "New part"
}

// persistOptions saves the live, tab-edited state into the per-user options file.
func persistOptions(s *app.Session) { reportPrefError(s.PersistLiveOptions()) }

// reportPrefError surfaces a failed preference save without interrupting the UI.
func reportPrefError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "preferences: %v\n", err)
	}
}

// drawGridTab renders the sketch-grid and sketch-input preference fields.
func drawGridTab(s *app.Session) {
	editGridSpacing(s)
	editGridVisibility(s)
	editGridMajor(s)
	native.Separator()
	editHUDEnabled(s)
}

// editHUDEnabled draws the dynamic-input HUD enable checkbox (#790): the in-canvas heads-up
// entry of coordinates/length/angle shown while drawing sketch geometry.
func editHUDEnabled(s *app.Session) {
	on := s.HUDEnabled()
	if native.Checkbox("Heads-up dynamic input while sketching", &on) {
		s.SetHUDEnabled(on)
	}
}

// editGridSpacing draws the spacing field labeled with the document's length unit.
func editGridSpacing(s *app.Session) {
	value, unit := s.GridSpacingDisplay()
	v := float32(value)
	if native.InputFloat("Grid spacing ("+unit+")", &v) {
		if s.SetGridSpacingDisplay(float64(v)) == nil { // ignore non-positive entries
			persistOptions(s)
		}
	}
}

// editGridVisibility draws the show-grid checkbox.
func editGridVisibility(s *app.Session) {
	vis := s.Grid().Visible
	if native.Checkbox("Show grid in sketch", &vis) {
		s.Grid().Visible = vis
		persistOptions(s)
	}
}

// editGridMajor draws the major-line interval field.
func editGridMajor(s *app.Session) {
	major := int32(s.Grid().MajorEvery)
	if native.InputInt("Major line every N", &major) && major >= 1 {
		s.Grid().MajorEvery = int(major)
		persistOptions(s)
	}
}
