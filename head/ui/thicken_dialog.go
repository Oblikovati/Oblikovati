//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Thicken flow in the head: while the tool runs, a modeless property panel (the
// reference panel schema) takes the wall thickness applied to the active surface body,
// then OK/Cancel. There is no pick — the tool operates on the active surface.
var thickenUI = struct {
	thickness float32
	open      bool
}{thickness: 1}

// drawThickenDialog shows the Thicken property panel while the Thicken tool is active.
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
	native.SetNextWindowSizeOnce(340, 170)
	if native.Begin("Thicken") {
		drawFeatureBreadcrumb("Thicken", "")
		if propertySection("Behavior") {
			propertyFloatRow("Thickness", "thicken-thickness", s.LengthUnitName(), &thickenUI.thickness)
			th.SetThickness(float64(thickenUI.thickness))
		}
		native.Separator()
		drawCommitCancelButtons(s, th.CanCommit())
	}
	native.End()
}
