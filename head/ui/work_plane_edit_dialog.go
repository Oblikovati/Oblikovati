//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Editing a placed offset work plane (browser double-click): a small dialog edits its offset
// distance and OK/Cancel, mirroring the Offset Plane creation dialog. While it is open the
// model is rolled back to the plane (edit-scope, issue #132), so the change previews against
// only the geometry that existed when the plane was made. The distance is in the document's
// length unit.

// workPlaneEditUI holds the dialog's distance field across frames and whether it was open last
// frame (so it seeds the field once when the edit opens).
var workPlaneEditUI = struct {
	distance float32
	open     bool
}{distance: 10}

// drawWorkPlaneEditDialog shows the offset editor while a work-plane edit is active, keeping
// the tool's distance in sync with the field each frame; OK commits, Cancel restores.
func drawWorkPlaneEditDialog(s *app.Session) {
	t := s.ActiveWorkPlaneEdit()
	if t == nil {
		workPlaneEditUI.open = false
		return
	}
	if !workPlaneEditUI.open { // edit just opened — seed the field from the plane's current offset
		workPlaneEditUI.distance = float32(s.EditPlaneOffsetDisplay())
		workPlaneEditUI.open = true
	}
	native.SetNextWindowSize(300, 110)
	if native.Begin("Edit Work Plane") {
		native.Text("Offset (" + s.LengthUnitName() + ")")
		native.InputFloat("##edit-plane-offset", &workPlaneEditUI.distance)
		s.SetEditPlaneOffsetDisplay(float64(workPlaneEditUI.distance)) // keep the tool in sync
		if native.Button("OK") {
			_ = s.OK()
		}
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelTool()
		}
	}
	native.End()
}
