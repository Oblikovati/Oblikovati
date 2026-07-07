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
			drawDraftPullRow(d)
			drawDraftNeutralRow(d)
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

// drawDraftPullRow shows the pull-direction chip (the face whose normal is the mould-pull axis,
// defaulting to +Z when unset) and the toggle that routes the next viewport click to it.
func drawDraftPullRow(d *app.DraftTool) {
	drawPickChipRow("Pull direction", "draft-pull", pickChipText(d.PullSet(), "1 Face", "+Z (default)"),
		d.PullSet(), "The face whose normal is the mould-pull direction", d.ClearPull)
	propertyRow("")
	pickPull := d.PickingPull()
	if native.Checkbox("Pick pull face", &pickPull) {
		d.SetPickingPull(pickPull)
	}
}

// drawDraftNeutralRow shows the neutral (parting) plane chip and the toggle that routes the next
// viewport click to it. With no neutral plane each face pivots on its implicit lowest-vertex hinge.
func drawDraftNeutralRow(d *app.DraftTool) {
	drawPickChipRow("Neutral plane", "draft-neutral", pickChipText(d.NeutralSet(), "1 Face", "None"),
		d.NeutralSet(), "The parting plane the taper pivots about", d.ClearNeutral)
	propertyRow("")
	pickNeutral := d.PickingNeutral()
	if native.Checkbox("Pick neutral plane", &pickNeutral) {
		d.SetPickingNeutral(pickNeutral)
	}
}
