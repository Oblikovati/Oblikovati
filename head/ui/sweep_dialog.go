//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Sweep flow in the head: while the Sweep tool runs, a modeless property panel
// (the reference panel schema) drives the tool — the Profiles and Path chips, the
// twist, and the boolean output — then OK/Cancel. The picked profile is outlined by
// the tool's preview.
var sweepUI = struct {
	twistDeg float32
	open     bool
}{}

// drawSweepDialog shows the Sweep property panel while the Sweep tool is active,
// syncing every control with the tool each frame; OK commits, Cancel aborts.
func drawSweepDialog(s *app.Session) {
	sw := s.ActiveSweep()
	if sw == nil {
		sweepUI.open = false
		return
	}
	if !sweepUI.open {
		sweepUI.twistDeg = float32(sw.Twist() * 180 / stdmath.Pi)
		sweepUI.open = true
	}
	native.SetNextWindowSizeOnce(340, 320)
	if native.Begin("Sweep") {
		drawFeatureBreadcrumb("Sweep", sw.SourceSketchName())
		drawSweepInputGeometry(sw)
		drawSweepBehavior(sw)
		drawSweepOutput(sw)
		native.Separator()
		drawCommitCancelButtons(s, sw.CanCommit())
	}
	native.End()
}

// drawSweepInputGeometry is the Input Geometry section: the required Profiles chip and
// the required Path chip, each clearable on its own.
func drawSweepInputGeometry(sw *app.SweepTool) {
	if !propertySection("Input Geometry") {
		return
	}
	propertyRow("Profiles")
	_, hasProfile := sw.PickedProfile()
	if propertySelectorChip("sweep-profiles", pickChipText(hasProfile, "1 Profile", "Select Profile"), hasProfile, true) {
		sw.ClearProfile()
	}
	native.SetItemTooltip("Click a region in the viewport to sweep")
	propertyRow("Path")
	_, hasPath := sw.PickedPath()
	if propertySelectorChip("sweep-path", pickChipText(hasPath, "1 Path", "Select Path"), hasPath, true) {
		sw.ClearPath()
	}
	native.SetItemTooltip("Click the curve to sweep along")
}

// drawSweepBehavior is the Behavior section: the twist spread along the path.
func drawSweepBehavior(sw *app.SweepTool) {
	if !propertySection("Behavior") {
		return
	}
	propertyFloatRow("Twist", "sweep-twist", "deg", &sweepUI.twistDeg)
	sw.SetTwist(float64(sweepUI.twistDeg) * stdmath.Pi / 180)
}

// drawSweepOutput is the Output section: the shared Boolean toggle row.
func drawSweepOutput(sw *app.SweepTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("sweep-boolean", sw.Operation(), sw.SetOperation)
}
