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
	typeIndex    int // setback mode combo: 0 distance, 1 two distances, 2 distance and angle
	distance     float32
	distance2    float32 // two-distance mode: the second face's setback
	angleDeg     float32 // distance-and-angle mode: the chamfer-face angle
	flatCorners  bool
	concaveIndex int              // concave-edge strategy combo: 0 outward (fill), 1 inward (relief)
	tangentChain bool             // select the whole tangent chain on a plain pick (#1947)
	seeded       *app.ChamferTool // the tool the fields were seeded from (nil = none)
}{distance: 1, distance2: 1, angleDeg: 45, flatCorners: true}

// drawChamferDialog shows the Chamfer property panel while the Chamfer tool is active —
// creating a chamfer or re-editing a committed one (the same panel serves both).
func drawChamferDialog(s *app.Session) {
	c := s.ActiveChamfer()
	if c == nil {
		chamferUI.seeded = nil
		return
	}
	if chamferUI.seeded != c {
		seedChamferUI(c)
	}
	dialogSizeOnce(340, 250)
	if native.Begin("Chamfer") {
		drawFeatureBreadcrumb(chamferTitle(c), "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Edges", "chamfer-edges", countChipText(c.EdgeCount(), "Edge", "Select Edges"),
				c.EdgeCount() > 0, "Click edges to bevel; with Tangent chain on, a click selects the whole connected loop (Shift+click always does)", c.ClearEdges)
			drawTangentChainRow("chamfer", &chamferUI.tangentChain, c.SetTangentChain)
		}
		if propertySection("Behavior") {
			drawChamferBehaviorRows(s, c)
		}
		native.Separator()
		drawCommitCancelButtons(s, c.CanCommit())
	}
	native.End()
}

// chamferTitle is the panel/breadcrumb title: a committed chamfer's name when re-editing, else
// "Chamfer".
func chamferTitle(c *app.ChamferTool) string {
	if name := c.EditingName(); name != "" {
		return name
	}
	return "Chamfer"
}

// seedChamferUI loads the panel buffers from the tool the first frame it appears (creation
// defaults, or a committed chamfer's values in edit mode) — mirrors seedFilletUI.
func seedChamferUI(c *app.ChamferTool) {
	chamferUI.typeIndex = c.ChamferTypeIndex()
	chamferUI.distance = float32(c.Distance())
	chamferUI.distance2 = float32(c.Distance2())
	chamferUI.angleDeg = float32(c.AngleDegrees())
	chamferUI.flatCorners = c.FlatCorners()
	chamferUI.concaveIndex = c.ConcaveStrategyIndex()
	chamferUI.tangentChain = c.TangentChain()
	chamferUI.seeded = c
}

// drawChamferBehaviorRows renders the setback mode and its inputs, the concave-edge strategy
// combo, and the flat-corner toggle. The mode's second input replaces the flat-corner toggle:
// an asymmetric chamfer leaves a three-edge corner pointy, so the toggle would do nothing (#2045).
func drawChamferBehaviorRows(s *app.Session, c *app.ChamferTool) {
	if i := propertyComboRow("Type", "chamfer-type", app.ChamferTypeNames(), chamferUI.typeIndex); i >= 0 {
		chamferUI.typeIndex = i
	}
	c.SetChamferTypeIndex(chamferUI.typeIndex)
	lengthCmRow(s, "Distance", "chamfer-distance", &chamferUI.distance)
	c.SetDistance(float64(chamferUI.distance))
	switch chamferUI.typeIndex { // the input the selected mode adds; equal-distance adds none
	case chamferTypeTwoDistances:
		lengthCmRow(s, "Distance 2", "chamfer-distance2", &chamferUI.distance2)
		c.SetDistance2(float64(chamferUI.distance2))
	case chamferTypeDistanceAndAngle:
		angleDegRow(s, "Angle", "chamfer-angle", &chamferUI.angleDeg)
		c.SetAngleDegrees(float64(chamferUI.angleDeg))
	}
	drawChamferCornerRows(c)
}

// The setback modes' indices in app.ChamferTypeNames.
const (
	chamferTypeDistance = iota
	chamferTypeTwoDistances
	chamferTypeDistanceAndAngle
)

// drawChamferCornerRows renders the concave-edge strategy combo and — for the equal-distance
// mode only — the flat-corner toggle: an asymmetric chamfer leaves a three-edge corner pointy,
// so the toggle would have nothing to blend (#2045).
func drawChamferCornerRows(c *app.ChamferTool) {
	if i := propertyComboRow("Concave edge", "chamfer-concave", app.ConcaveStrategyNames(), chamferUI.concaveIndex); i >= 0 {
		chamferUI.concaveIndex = i
	}
	c.SetConcaveStrategyIndex(chamferUI.concaveIndex)
	if chamferUI.typeIndex != chamferTypeDistance {
		return
	}
	propertyRow("")
	if native.Checkbox("Flat corner (3 edges)", &chamferUI.flatCorners) {
		c.SetFlatCorners(chamferUI.flatCorners)
	}
}
