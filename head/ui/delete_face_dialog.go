//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Delete Face flow in the head: while the tool runs, a modeless options window shows the
// picked-face count, then OK/Cancel (there is no parameter — the heal is automatic).
var deleteFaceUI = struct{ open bool }{}

// drawDeleteFaceDialog shows the delete-face options window while the tool is active.
func drawDeleteFaceDialog(s *app.Session) {
	d := s.ActiveDeleteFace()
	if d == nil {
		deleteFaceUI.open = false
		return
	}
	deleteFaceUI.open = true
	native.SetNextWindowSize(300, 130)
	if native.Begin("Delete Face") {
		native.Text("Faces: " + strconv.Itoa(len(d.Faces())) + " (click faces to delete)")
		native.Text("Neighbours heal to close the opening.")
		native.BeginDisabled(!d.CanCommit())
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
