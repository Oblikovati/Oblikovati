//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Sweep flow in the head: while the Sweep tool runs, a modeless options window shows
// the profile/path pick state, the output operation and a twist field (degrees), then
// OK/Cancel. The picked profile is outlined by the tool's preview.
var sweepUI = struct {
	twistDeg float32
	open     bool
}{}

// drawSweepDialog shows the sweep options window while the Sweep tool is active.
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
	native.SetNextWindowSize(300, 220)
	if native.Begin("Sweep") {
		native.Text(sweepPickStatus(sw))
		sweepOperationCombo(sw)
		native.Text("Twist (deg)")
		native.InputFloat("##sweep-twist", &sweepUI.twistDeg)
		sw.SetTwist(float64(sweepUI.twistDeg) * stdmath.Pi / 180)
		drawSweepButtons(s, sw)
	}
	native.End()
}

func sweepPickStatus(sw *app.SweepTool) string {
	_, hasProfile := sw.PickedProfile()
	_, hasPath := sw.PickedPath()
	switch {
	case !hasProfile:
		return "Click a region to sweep"
	case !hasPath:
		return "Click a path to sweep along"
	default:
		return "Profile and path selected"
	}
}

func sweepOperationCombo(sw *app.SweepTool) {
	preview := "New Solid"
	for _, o := range extrudeOperations {
		if o.op == sw.Operation() {
			preview = o.label
		}
	}
	if native.BeginCombo("Output", preview) {
		for _, o := range extrudeOperations {
			if native.Selectable(o.label, o.op == sw.Operation()) {
				sw.SetOperation(o.op)
			}
		}
		native.EndCombo()
	}
}

func drawSweepButtons(s *app.Session, sw *app.SweepTool) {
	native.BeginDisabled(!sw.CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}
