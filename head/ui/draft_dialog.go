//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Draft flow in the head: while the Draft tool runs, a modeless property panel
// (the reference panel schema) shows the picked-faces chip and the signed draft angle,
// then OK/Cancel.
var draftUI = struct {
	angle float32
	open  bool
}{angle: 3}

// drawDraftDialog shows the Draft property panel while the Draft tool is active.
func drawDraftDialog(s *app.Session) {
	d := s.ActiveDraft()
	if d == nil {
		draftUI.open = false
		return
	}
	if !draftUI.open {
		draftUI.angle = float32(d.AngleDegrees())
		draftUI.open = true
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Draft") {
		drawFeatureBreadcrumb("Draft", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Faces", "draft-faces", countChipText(len(d.Faces()), "Face", "Select Faces"),
				len(d.Faces()) > 0, "Click faces in the viewport to draft", d.ClearFaces)
		}
		if propertySection("Behavior") {
			propertyFloatRow("Angle", "draft-angle", "deg (+out / −in)", &draftUI.angle)
			d.SetAngleDegrees(float64(draftUI.angle))
		}
		native.Separator()
		drawCommitCancelButtons(s, d.CanCommit())
	}
	native.End()
}
