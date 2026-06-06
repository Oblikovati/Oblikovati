//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Offset Plane flow in the head: while the Offset Plane tool runs, a small dialog
// prompts for the base plane (picked in the view/browser) and then an editable offset
// distance, OK/Cancel. Without it the tool could pick a plane but never get a distance, so
// OK stayed disabled — Inventor asks for the distance rather than dropping a plane at a
// fixed offset. The distance is in the document's length unit (e.g. mm).

// offsetPlaneUI holds the dialog's distance field across frames and whether the dialog was
// open last frame (so it seeds the field once when the tool opens).
var offsetPlaneUI = struct {
	distance float32
	open     bool
}{distance: 10}

// drawOffsetPlaneDialog shows the offset editor while the Offset Plane tool is active and
// keeps the tool's distance in sync with the field each frame; OK commits, Cancel aborts.
func drawOffsetPlaneDialog(s *app.Session) {
	t := s.ActiveOffsetPlane()
	if t == nil {
		offsetPlaneUI.open = false
		return
	}
	if !offsetPlaneUI.open { // tool just opened — seed the field from its current offset
		if d := s.OffsetDistanceDisplay(); d != 0 {
			offsetPlaneUI.distance = float32(d)
		}
		offsetPlaneUI.open = true
	}
	native.SetNextWindowSize(300, 124)
	if native.Begin("Offset Plane") {
		if !t.BasePicked() {
			native.Text("Select a plane or planar face to offset from")
		} else {
			native.Text("Offset (" + s.LengthUnitName() + ")")
			native.InputFloat("##offset-distance", &offsetPlaneUI.distance)
			s.SetOffsetDistanceDisplay(float64(offsetPlaneUI.distance)) // keep the tool in sync
		}
		native.BeginDisabled(!t.CanCommit())
		if native.Button("OK") {
			_ = s.OK() // a failed commit keeps the tool open
		}
		native.EndDisabled()
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelTool()
		}
	}
	native.End()
}
