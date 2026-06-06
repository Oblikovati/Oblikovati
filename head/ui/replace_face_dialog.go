//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Replace Face flow in the head: while the tool runs, a modeless options window lists
// the picked faces, a toggle to pick the target face, and the chosen target, then OK/Cancel.
var replaceFaceUI = struct{ open bool }{}

// drawReplaceFaceDialog shows the replace-face options window while the tool is active.
func drawReplaceFaceDialog(s *app.Session) {
	r := s.ActiveReplaceFace()
	if r == nil {
		replaceFaceUI.open = false
		return
	}
	replaceFaceUI.open = true
	native.SetNextWindowSize(320, 170)
	if native.Begin("Replace Face") {
		native.Text("Faces to replace: " + strconv.Itoa(len(r.Faces())))
		pickTarget := r.PickingTarget()
		if native.Checkbox("Pick target face", &pickTarget) {
			r.SetPickingTarget(pickTarget)
		}
		native.Text("Target: " + targetLabel(r))
		native.BeginDisabled(!r.CanCommit())
		if native.Button("OK") {
			_ = s.OK()
		}
		native.EndDisabled()
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelTool()
		}
	}
	native.End()
}

// targetLabel reports whether a target face has been chosen.
func targetLabel(r *app.ReplaceFaceTool) string {
	if _, ok := r.PickedTarget(); ok {
		return "selected"
	}
	return "none"
}
