//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Thicken flow in the head: while the tool runs, a modeless options window takes the
// wall thickness applied to the active surface body, then OK/Cancel.
var thickenUI = struct {
	thickness float32
	open      bool
}{thickness: 1}

// drawThickenDialog shows the thicken options window while the Thicken tool is active.
func drawThickenDialog(s *app.Session) {
	th := s.ActiveThicken()
	if th == nil {
		thickenUI.open = false
		return
	}
	if !thickenUI.open {
		thickenUI.thickness = float32(th.Thickness())
		thickenUI.open = true
	}
	native.SetNextWindowSize(300, 140)
	if native.Begin("Thicken") {
		native.Text("Thickness (" + s.LengthUnitName() + ")")
		native.InputFloat("##thicken-thickness", &thickenUI.thickness)
		th.SetThickness(float64(thickenUI.thickness))
		native.BeginDisabled(!th.CanCommit())
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
