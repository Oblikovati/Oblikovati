//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Replace Face flow in the head: while the tool runs, a modeless property panel
// (the reference panel schema) shows the faces-to-replace chip, the target-face chip,
// and the toggle routing the next viewport pick to the target, then OK/Cancel.
var replaceFaceUI = struct{ open bool }{}

// drawReplaceFaceDialog shows the Replace Face property panel while the tool is active.
func drawReplaceFaceDialog(s *app.Session) {
	r := s.ActiveReplaceFace()
	if r == nil {
		replaceFaceUI.open = false
		return
	}
	replaceFaceUI.open = true
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Replace Face") {
		drawFeatureBreadcrumb("Replace Face", "")
		if propertySection("Input Geometry") {
			drawReplaceFacePicks(r)
		}
		native.Separator()
		drawCommitCancelButtons(s, r.CanCommit())
	}
	native.End()
}

// drawReplaceFacePicks renders the two pick chips and the toggle that routes the next
// viewport click to the target face instead of the replace set.
func drawReplaceFacePicks(r *app.ReplaceFaceTool) {
	drawPickChipRow("Faces", "replace-faces", countChipText(len(r.Faces()), "Face", "Select Faces"),
		len(r.Faces()) > 0, "Click the faces to replace", r.ClearFaces)
	_, hasTarget := r.PickedTarget()
	drawPickChipRow("Target", "replace-target", pickChipText(hasTarget, "1 Face", "Select Face"),
		hasTarget, "The face the picked set is replaced with", r.ClearTarget)
	propertyRow("")
	pickTarget := r.PickingTarget()
	if native.Checkbox("Pick target face", &pickTarget) {
		r.SetPickingTarget(pickTarget)
	}
}
