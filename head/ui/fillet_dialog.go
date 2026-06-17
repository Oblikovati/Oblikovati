//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Fillet flow in the head: while the Fillet tool runs, a modeless property panel
// (the reference panel schema) shows the picked-edges chip and the blend radius, then
// OK/Cancel.
var filletUI = struct {
	radius      float32
	startRadius float32
	endRadius   float32
	seeded      *app.FilletTool // the tool the fields were seeded from (nil = none)
}{radius: 1, startRadius: 1, endRadius: 1}

// drawFilletDialog shows the Fillet property panel while the Fillet tool is active —
// creating a fillet or re-editing a committed one (the same panel serves both).
func drawFilletDialog(s *app.Session) {
	f := s.ActiveFillet()
	if f == nil {
		filletUI.seeded = nil
		return
	}
	if filletUI.seeded != f {
		seedFilletUI(f)
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Fillet") {
		title := "Fillet"
		if name := f.EditingName(); name != "" {
			title = name // re-editing a committed fillet: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Edges", "fillet-edges", countChipText(f.EdgeCount(), "Edge", "Select Edges"),
				f.EdgeCount() > 0, "Click convex edges in the viewport to round", f.ClearEdges)
		}
		if propertySection("Behavior") {
			drawFilletRadiusRows(s, f)
			if i := propertyComboRow("Corner", "fillet-corner", app.FilletCornerOptions(), f.CornerTypeIndex()); i >= 0 {
				f.SetCornerTypeIndex(i)
			}
			if i := propertyComboRow("Concave edge", "fillet-concave", app.FilletConcaveOptions(), f.ConcaveStrategyIndex()); i >= 0 {
				f.SetConcaveStrategyIndex(i)
			}
		}
		native.Separator()
		drawCommitCancelButtons(s, f.CanCommit())
	}
	native.End()
}

// seedFilletUI loads the panel buffers from the tool the first frame it appears
// (creation defaults, or a committed fillet's values in edit mode).
func seedFilletUI(f *app.FilletTool) {
	filletUI.radius = float32(f.Radius())
	filletUI.startRadius = float32(f.StartRadius())
	filletUI.endRadius = float32(f.EndRadius())
	filletUI.seeded = f
}

// drawFilletRadiusRows renders the constant-vs-variable mode (#323): a constant blend
// takes one radius; variable blends each picked edge from a start to an end radius.
func drawFilletRadiusRows(s *app.Session, f *app.FilletTool) {
	propertyRow("")
	variable := f.Variable()
	if native.Checkbox("Variable radius (start → end per edge)", &variable) {
		f.SetVariable(variable)
	}
	if !variable {
		propertyFloatRow("Radius", "fillet-radius", s.LengthUnitName(), &filletUI.radius)
		f.SetRadius(float64(filletUI.radius))
		return
	}
	propertyFloatRow("Start radius", "fillet-start-radius", s.LengthUnitName(), &filletUI.startRadius)
	f.SetStartRadius(float64(filletUI.startRadius))
	propertyFloatRow("End radius", "fillet-end-radius", s.LengthUnitName(), &filletUI.endRadius)
	f.SetEndRadius(float64(filletUI.endRadius))
}
