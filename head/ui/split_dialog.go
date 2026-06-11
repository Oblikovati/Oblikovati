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

// splitModeToggles is the Behavior section's Mode row: which side(s) the split keeps.
var splitModeToggles = propertyToggleSet{
	keys: []string{"split-keep-both", "split-keep-front", "split-keep-back"},
	tips: []string{
		"Split — keep both sides as separate bodies",
		"Trim — keep the front side only",
		"Trim — keep the back side only",
	},
}

// splitModeSides lists the keep sides in the Mode toggle row's order.
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

// drawSplitModeRow renders the keep-mode toggles, mapped onto the tool's SplitSide.
func drawSplitModeRow(t *app.SplitTool) {
	propertyRow("Mode")
	if i := propertyIconToggles("split-mode", splitModeToggles.keys, splitModeToggles.tips, splitModeIndex(t)); i >= 0 {
		t.SetKeep(splitModeSides[i])
	}
}

// splitModeIndex maps the tool's keep side onto the Mode toggle row.
func splitModeIndex(t *app.SplitTool) int {
	for i, side := range splitModeSides {
		if side == t.Keep() {
			return i
		}
	}
	return 0
}
