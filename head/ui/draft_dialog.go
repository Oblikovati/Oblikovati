//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// The Draft flow in the head: while the Draft tool runs, a modeless options window shows
// the picked-face count and the draft angle (degrees, signed), then OK/Cancel.
var draftUI = struct {
	angle float32
	open  bool
}{angle: 3}

// drawDraftDialog shows the draft options window while the Draft tool is active.
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
	native.SetNextWindowSize(300, 160)
	if native.Begin("Draft") {
		native.Text("Faces: " + strconv.Itoa(len(d.Faces())) + " (click faces to draft)")
		native.Text("Angle (degrees, +out / −in)")
		native.InputFloat("##draft-angle", &draftUI.angle)
		d.SetAngleDegrees(float64(draftUI.angle))
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
