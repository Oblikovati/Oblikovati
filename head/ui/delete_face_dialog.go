//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Delete Face flow in the head: while the tool runs, a modeless property panel
// (the reference panel schema) shows the picked-faces chip, then OK/Cancel (there is
// no parameter — the heal is automatic).
var deleteFaceUI = struct{ open bool }{}

// drawDeleteFaceDialog shows the Delete Face property panel while the tool is active.
func drawDeleteFaceDialog(s *app.Session) {
	d := s.ActiveDeleteFace()
	if d == nil {
		deleteFaceUI.open = false
		return
	}
	deleteFaceUI.open = true
	native.SetNextWindowSizeOnce(340, 190)
	if native.Begin("Delete Face") {
		drawFeatureBreadcrumb("Delete Face", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Faces", "delete-faces", countChipText(len(d.Faces()), "Face", "Select Faces"),
				len(d.Faces()) > 0, "Click faces in the viewport to delete", d.ClearFaces)
			native.Text("Neighbours heal to close the opening.")
		}
		native.Separator()
		drawCommitCancelButtons(s, d.CanCommit())
	}
	native.End()
}
