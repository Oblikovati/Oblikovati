//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Sweep flow in the head follows Inventor's Sweep dialog: while the Sweep tool runs, a modeless
// property panel drives the tool through Input Geometry (the Profile, Path, and optional Guide
// Rail), Behavior (how the profile rides the path — orientation, taper, twist, and a guide rail's
// profile scaling), and Output (the boolean), then OK/Cancel. The picked profile is outlined by the
// tool's preview.
var sweepUI = struct {
	twistDeg float32
	taperDeg float32
	open     bool
}{}

// sweepOrientationLabels maps the orientation combo to the tool's orientations (index order).
var sweepOrientationLabels = []string{"Follow Path", "Parallel"}

// sweepScalingLabels maps the profile-scaling combo to the tool's scalings (index order).
var sweepScalingLabels = []string{"X & Y", "X", "None"}

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
		sweepUI.taperDeg = float32(sw.Taper() * 180 / stdmath.Pi)
		sweepUI.open = true
	}
	dialogSizeOnce(340, 420)
	if native.Begin("Sweep") {
		drawFeatureBreadcrumb("Sweep", sw.SourceSketchName())
		drawSweepInputGeometry(sw)
		drawSweepBehavior(s, sw)
		drawSweepOutput(sw)
		native.Separator()
		drawCommitCancelButtons(s, sw.CanCommit())
	}
	native.End()
}

// drawSweepInputGeometry is the Input Geometry section: the required Profile and Path chips, plus
// the optional Guide Rail.
func drawSweepInputGeometry(sw *app.SweepTool) {
	if !propertySection("Input Geometry") {
		return
	}
	propertyRow("Profile")
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
	drawSweepGuideRail(sw)
}

// drawSweepGuideRail is the optional Guide Rail slot: a curve the profile follows and scales to as
// it rides the path (Inventor's Path & Guide Rail sweep). Clicking the chip arms viewport picking;
// the × clears it.
func drawSweepGuideRail(sw *app.SweepTool) {
	propertyRow("Guide Rail")
	_, has := sw.PickedGuideRail()
	arm, clear := propertyOptionalArmableSlotChip(
		"sweep-guiderail", pickChipText(has, "1 Rail", "Optional"), has, sw.GuideRailArmed(), has)
	if arm {
		sw.ArmGuideRailPicking()
	}
	if clear {
		sw.ClearGuideRail()
	}
	native.SetItemTooltip("Click a curve that steers and scales the profile along the path")
}

// drawSweepBehavior is the Behavior section: how the profile rides the path — its orientation, the
// taper and twist along the length, and (with a guide rail) the profile scaling.
func drawSweepBehavior(s *app.Session, sw *app.SweepTool) {
	if !propertySection("Behavior") {
		return
	}
	drawSweepOrientation(sw)
	angleDegRow(s, "Taper", "sweep-taper", &sweepUI.taperDeg)
	sw.SetTaper(float64(sweepUI.taperDeg) * stdmath.Pi / 180)
	angleDegRow(s, "Twist", "sweep-twist", &sweepUI.twistDeg)
	sw.SetTwist(float64(sweepUI.twistDeg) * stdmath.Pi / 180)
	if _, hasRail := sw.PickedGuideRail(); hasRail {
		drawSweepScaling(sw)
	}
}

// drawSweepOrientation is the orientation combo: Follow Path (the profile stays normal to the path)
// or Parallel (it stays parallel to its sketch plane).
func drawSweepOrientation(sw *app.SweepTool) {
	if chosen := propertyComboRow("Orientation", "sweep-orientation", sweepOrientationLabels, sweepOrientationIndex(sw.Orientation())); chosen >= 0 {
		sw.SetOrientation(sweepOrientationFromIndex(chosen))
	}
}

// drawSweepScaling is the guide rail's Profile Scaling combo (shown only with a rail): X & Y, X, or
// None — how the rail's distance scales the profile.
func drawSweepScaling(sw *app.SweepTool) {
	if chosen := propertyComboRow("Profile Scaling", "sweep-scaling", sweepScalingLabels, sweepScalingIndex(sw.Scaling())); chosen >= 0 {
		sw.SetScaling(sweepScalingFromIndex(chosen))
	}
}

// drawSweepOutput is the Output section: the shared Boolean toggle row.
func drawSweepOutput(sw *app.SweepTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("sweep-boolean", sw.Operation(), sw.SetOperation)
}

// sweepOrientationIndex / sweepOrientationFromIndex map the orientation combo index (the
// sweepOrientationLabels order) to and from the tool's orientation enum.
func sweepOrientationIndex(o types.SweepProfileOrientation) int {
	if o == types.ParallelToOriginalProfile {
		return 1
	}
	return 0
}

func sweepOrientationFromIndex(i int) types.SweepProfileOrientation {
	if i == 1 {
		return types.ParallelToOriginalProfile
	}
	return types.NormalToPath
}

// sweepScalingIndex / sweepScalingFromIndex map the profile-scaling combo index (the
// sweepScalingLabels order: X&Y, X, None) to and from the tool's scaling enum.
func sweepScalingIndex(s types.SweepProfileScaling) int {
	switch s {
	case types.XProfileScaling:
		return 1
	case types.NoProfileScaling:
		return 2
	default:
		return 0
	}
}

func sweepScalingFromIndex(i int) types.SweepProfileScaling {
	switch i {
	case 1:
		return types.XProfileScaling
	case 2:
		return types.NoProfileScaling
	default:
		return types.XYProfileScaling
	}
}
