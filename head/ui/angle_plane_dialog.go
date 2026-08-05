//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Angle to Plane flow in the head: while the tool runs, a modeless property panel shows the
// axis and base-plane chips and an editable angle, then OK/Cancel. Like the offset plane it does
// not auto-commit on the picks — a plane at no angle is just the base plane (#2044).

// anglePlaneUI holds the panel's angle field across frames, in the document's angle unit.
var anglePlaneUI = struct {
	angleDeg float32
	open     bool
}{angleDeg: 45}

// drawAnglePlaneDialog shows the angle editor while the Angle to Plane tool is active and keeps
// the tool's angle in sync with the field each frame.
func drawAnglePlaneDialog(s *app.Session) {
	t := s.ActiveAnglePlane()
	if t == nil {
		anglePlaneUI.open = false
		return
	}
	if !anglePlaneUI.open { // tool just opened — seed the field from its current angle
		if a := t.AngleDegrees(); a != 0 {
			anglePlaneUI.angleDeg = float32(a)
		}
		anglePlaneUI.open = true
	}
	dialogSizeOnce(340, 250)
	if native.Begin("Angle to Plane") {
		drawFeatureBreadcrumb("Angle to Plane", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("About", "angle-plane-axis", pickChipText(t.AxisPicked(), "1 Axis", "Select Axis"),
				t.AxisPicked(), "Click the axis or edge to rotate about", t.ClearPicks)
			drawPickChipRow("From", "angle-plane-base", pickChipText(t.BasePicked(), "1 Plane", "Select Plane"),
				t.BasePicked(), "Click the plane or planar face to measure the angle from", t.ClearPicks)
		}
		if propertySection("Behavior") {
			native.BeginDisabled(!t.AxisPicked() || !t.BasePicked())
			angleDegRow(s, "Angle", "angle-plane-angle", &anglePlaneUI.angleDeg)
			native.EndDisabled()
			t.SetAngleDegrees(float64(anglePlaneUI.angleDeg))
		}
		native.Separator()
		drawCommitCancelButtons(s, t.CanCommit())
	}
	native.End()
}
