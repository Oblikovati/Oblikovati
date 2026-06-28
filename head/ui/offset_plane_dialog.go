//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Offset Plane flow in the head: while the Offset Plane tool runs, a modeless
// property panel (the reference panel schema) shows the base-plane chip and an editable
// offset distance, then OK/Cancel. Without it the tool could pick a plane but never get
// a distance, so OK stayed disabled. The distance is in the document's length unit.

// offsetPlaneUI holds the panel's distance field across frames and whether the panel
// was open last frame (so it seeds the field once when the tool opens).
var offsetPlaneUI = struct {
	distance float32
	open     bool
}{distance: 10}

// drawOffsetPlaneDialog shows the offset editor while the Offset Plane tool is active
// and keeps the tool's distance in sync with the field each frame; OK commits, Cancel
// aborts.
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
	native.SetNextWindowSizeOnce(340, 210)
	if native.Begin("Offset Plane") {
		drawFeatureBreadcrumb("Offset Plane", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("From", "offset-plane-base", pickChipText(t.BasePicked(), "1 Plane", "Select Plane"),
				t.BasePicked(), "Click the plane or planar face to offset from", t.ClearBase)
		}
		if propertySection("Behavior") {
			drawOffsetPlaneDistanceRow(s, t)
		}
		native.Separator()
		drawCommitCancelButtons(s, t.CanCommit())
	}
	native.End()
}

// drawOffsetPlaneDistanceRow renders the offset field, greyed until the base is picked
// (the offset has no meaning without a plane to measure from).
func drawOffsetPlaneDistanceRow(s *app.Session, t *app.OffsetWorkPlaneTool) {
	native.BeginDisabled(!t.BasePicked())
	parameterFloatRow(s, "Offset", "offset-plane-distance", paramLength, "", &offsetPlaneUI.distance)
	native.EndDisabled()
	if t.BasePicked() {
		s.SetOffsetDistanceDisplay(float64(offsetPlaneUI.distance)) // keep the tool in sync
	}
}
