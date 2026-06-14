//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/app"

// placeUI tracks which Place Component tool instance the file picker has been opened for, so the
// picker opens exactly once when a Place tool starts awaiting a component — not every frame — and
// the tool is abandoned if the user dismisses the picker without choosing a file. Keyed on the
// tool pointer (like the extrude dialog's seeded field) so a fresh Place invocation re-arms.
var placeUI struct {
	armedFor *app.PlaceComponentTool
}

// drawPlaceComponentDialog drives the Place Component tool's one piece of chrome: choosing the
// component document to instance (#763). When a Place tool is active and has no component yet it
// opens the file picker (in dialogPlaceComponent mode, which feeds the choice to the tool via
// SetPlaceComponentDocument); if the user cancels the picker without choosing, the tool is
// abandoned so it does not sit waiting forever. Placement itself is ground-plane clicks in the
// viewport (the app PlaceComponentTool), so there is no per-frame property panel here.
func drawPlaceComponentDialog(s *app.Session) {
	tool := s.ActivePlaceComponent()
	if tool == nil {
		placeUI.armedFor = nil // tool committed or cancelled; re-arm for the next invocation
		return
	}
	if !s.PlaceComponentAwaitingFile() {
		return // a component is chosen; the tool now drops instances on viewport clicks
	}
	if placeUI.armedFor != tool {
		fileModal.openFor(dialogPlaceComponent) // open the picker once for this tool
		placeUI.armedFor = tool
		return
	}
	if !fileModal.isOpen() {
		s.CancelTool() // the picker was dismissed without a choice — abandon the Place tool
		placeUI.armedFor = nil
	}
}
