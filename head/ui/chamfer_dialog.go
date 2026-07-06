//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Chamfer flow in the head: while the Chamfer tool runs, a modeless property panel
// (the reference panel schema) shows the picked-edges chip, the setback distance, and
// the flat-corner toggle, then OK/Cancel.
var chamferUI = struct {
	distance     float32
	flatCorners  bool
	concaveIndex int              // concave-edge strategy combo: 0 outward (fill), 1 inward (relief)
	seeded       *app.ChamferTool // the tool the fields were seeded from (nil = none)
}{distance: 1, flatCorners: true}

// drawChamferDialog shows the Chamfer property panel while the Chamfer tool is active —
// creating a chamfer or re-editing a committed one (the same panel serves both).
func drawChamferDialog(s *app.Session) {
	c := s.ActiveChamfer()
	if c == nil {
		chamferUI.seeded = nil
		return
	}
	if chamferUI.seeded != c {
		chamferUI.distance = float32(c.Distance())
		chamferUI.flatCorners = c.FlatCorners()
		chamferUI.concaveIndex = c.ConcaveStrategyIndex()
		chamferUI.seeded = c
	}
	dialogSizeOnce(340, 250)
	if native.Begin("Chamfer") {
		title := "Chamfer"
		if name := c.EditingName(); name != "" {
			title = name // re-editing a committed chamfer: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Edges", "chamfer-edges", countChipText(c.EdgeCount(), "Edge", "Select Edges"),
				c.EdgeCount() > 0, "Click edges in the viewport to bevel", c.ClearEdges)
		}
		if propertySection("Behavior") {
			drawChamferBehaviorRows(s, c)
		}
		native.Separator()
		drawCommitCancelButtons(s, c.CanCommit())
	}
	native.End()
}

// drawChamferBehaviorRows renders the setback distance, the concave-edge strategy combo, and the
// flat-corner toggle.
func drawChamferBehaviorRows(s *app.Session, c *app.ChamferTool) {
	lengthCmRow(s, "Distance", "chamfer-distance", &chamferUI.distance)
	c.SetDistance(float64(chamferUI.distance))
	if i := propertyComboRow("Concave edge", "chamfer-concave", app.ConcaveStrategyNames(), chamferUI.concaveIndex); i >= 0 {
		chamferUI.concaveIndex = i
	}
	c.SetConcaveStrategyIndex(chamferUI.concaveIndex)
	propertyRow("")
	if native.Checkbox("Flat corner (3 edges)", &chamferUI.flatCorners) {
		c.SetFlatCorners(chamferUI.flatCorners)
	}
}
