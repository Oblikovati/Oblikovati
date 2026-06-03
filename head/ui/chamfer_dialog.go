//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// The Chamfer flow in the head: while the Chamfer tool runs, a modeless options window
// shows the picked-edge count and the setback distance (database units), then OK/Cancel.
var chamferUI = struct {
	distance    float32
	flatCorners bool
	open        bool
}{distance: 1, flatCorners: true}

// drawChamferDialog shows the chamfer options window while the Chamfer tool is active.
func drawChamferDialog(s *app.Session) {
	c := s.ActiveChamfer()
	if c == nil {
		chamferUI.open = false
		return
	}
	if !chamferUI.open {
		chamferUI.distance = float32(c.Distance())
		chamferUI.flatCorners = c.FlatCorners()
		chamferUI.open = true
	}
	native.SetNextWindowSize(300, 190)
	if native.Begin("Chamfer") {
		native.Text("Edges: " + strconv.Itoa(len(c.Edges())) + " (click edges to bevel)")
		native.Text("Distance (" + s.LengthUnitName() + ")")
		native.InputFloat("##chamfer-distance", &chamferUI.distance)
		c.SetDistance(float64(chamferUI.distance))
		if native.Checkbox("Flat corner (3 edges)", &chamferUI.flatCorners) {
			c.SetFlatCorners(chamferUI.flatCorners)
		}
		native.BeginDisabled(!c.CanCommit())
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
