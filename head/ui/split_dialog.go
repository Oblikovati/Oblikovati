//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Split flow in the head: while the Split tool runs, a modeless property panel
// (the reference panel schema) shows the cutting-plane chip and the keep-mode toggle
// row (split into two / trim either side), then OK/Cancel.

// splitModeToggles is the Behavior section's Mode row: which side(s) the split keeps,
// plus the faces-only imprint mode (#330).
var splitModeToggles = propertyToggleSet{
	keys: []string{"split-keep-both", "split-keep-front", "split-keep-back", "split-faces"},
	tips: []string{
		"Split — keep both sides as separate bodies",
		"Trim — keep the front side only",
		"Trim — keep the back side only",
		"Split Faces — imprint the plane onto the faces, removing nothing",
	},
}

// splitModeSides lists the keep sides for the first three Mode toggles; the fourth
// toggle is the faces-only mode.
var splitModeSides = []feature.SplitSide{feature.SplitBoth, feature.SplitPositive, feature.SplitNegative}

// drawSplitDialog shows the Split property panel while the Split tool is active.
func drawSplitDialog(s *app.Session) {
	t := s.ActiveSplit()
	if t == nil {
		return
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Split") {
		drawFeatureBreadcrumb("Split", "")
		if propertySection("Input Geometry") {
			_, picked := t.PickedPlane()
			drawPickChipRow("Plane", "split-plane", pickChipText(picked, "1 Plane", "Select Plane"),
				picked, "Click the work plane to cut with", t.ClearPlane)
		}
		if propertySection("Behavior") {
			drawSplitModeRow(t)
		}
		native.Separator()
		drawCommitCancelButtons(s, t.CanCommit())
	}
	native.End()
}

// drawSplitModeRow renders the keep-mode + faces-only toggles, mapped onto the tool.
func drawSplitModeRow(t *app.SplitTool) {
	propertyRow("Mode")
	i := propertyIconToggles("split-mode", splitModeToggles.keys, splitModeToggles.tips, splitModeIndex(t))
	if i < 0 {
		return
	}
	if i == len(splitModeSides) {
		t.SetSplitFaces()
		return
	}
	t.SetKeep(splitModeSides[i])
}

// splitModeIndex maps the tool's mode onto the toggle row (faces-only is the last slot).
func splitModeIndex(t *app.SplitTool) int {
	if t.FacesOnly() {
		return len(splitModeSides)
	}
	for i, side := range splitModeSides {
		if side == t.Keep() {
			return i
		}
	}
	return 0
}
