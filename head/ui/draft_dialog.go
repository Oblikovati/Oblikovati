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
	angle  float32
	seeded *app.DraftTool // the tool the fields were seeded from (nil = none)
}{angle: 3}

// drawDraftDialog shows the Draft property panel while the Draft tool is active —
// creating a draft or re-editing a committed one (the same panel serves both).
func drawDraftDialog(s *app.Session) {
	d := s.ActiveDraft()
	if d == nil {
		draftUI.seeded = nil
		return
	}
	if draftUI.seeded != d {
		draftUI.angle = float32(d.AngleDegrees())
		draftUI.seeded = d
	}
	dialogSizeOnce(340, 230)
	if native.Begin("Draft") {
		title := "Draft"
		if name := d.EditingName(); name != "" {
			title = name // re-editing a committed draft: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Faces", "draft-faces", countChipText(d.FaceCount(), "Face", "Select Faces"),
				d.FaceCount() > 0, "Click faces in the viewport to draft", d.ClearFaces)
		}
		if propertySection("Behavior") {
			angleDegRowHint(s, "Angle", "draft-angle", " (+out / −in)", &draftUI.angle)
			d.SetAngleDegrees(float64(draftUI.angle))
		}
		native.Separator()
		drawCommitCancelButtons(s, d.CanCommit())
	}
	native.End()
}
