//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// The Hole flow in the head: while the Hole tool runs, a modeless options window lets
// the user set the diameter and depth (database units), then OK/Cancel. The placement
// face is the one clicked in the viewport.
var holeUI = struct {
	diameter, depth float32
	open            bool
}{diameter: 1, depth: 2}

// drawHoleDialog shows the hole options window while the Hole tool is active.
func drawHoleDialog(s *app.Session) {
	h := s.ActiveHole()
	if h == nil {
		holeUI.open = false
		return
	}
	if !holeUI.open {
		holeUI.diameter = float32(h.Diameter())
		holeUI.depth = float32(h.Depth())
		holeUI.open = true
	}
	native.SetNextWindowSize(300, 180)
	if native.Begin("Hole") {
		if _, ok := h.PickedFace(); !ok {
			native.Text("Click a planar face to place the hole on")
		}
		native.Text("Diameter (" + s.LengthUnitName() + ")")
		native.InputFloat("##hole-diameter", &holeUI.diameter)
		h.SetDiameter(float64(holeUI.diameter))
		native.Text("Depth (" + s.LengthUnitName() + ")")
		native.InputFloat("##hole-depth", &holeUI.depth)
		h.SetDepth(float64(holeUI.depth))
		drawHoleButtons(s, h)
	}
	native.End()
}

func drawHoleButtons(s *app.Session, h *app.HoleTool) {
	native.BeginDisabled(!h.CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}
