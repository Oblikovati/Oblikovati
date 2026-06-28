//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

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
	rho         float32         // conic cross-section fullness (#1284)
	seeded      *app.FilletTool // the tool the fields were seeded from (nil = none)
}{radius: 1, startRadius: 1, endRadius: 1, rho: 0.5}

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
	native.SetNextWindowSizeOnce(340, 300)
	if native.Begin("Fillet") {
		drawFilletPanelBody(s, f)
	}
	native.End()
}

// drawFilletPanelBody draws the Fillet panel's sections (the Begin/End wrapper stays in
// drawFilletDialog): the breadcrumb, the edge picker, and the radius/corner behavior rows.
func drawFilletPanelBody(s *app.Session, f *app.FilletTool) {
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
		drawFilletCrossRows(f)
		drawFilletCornerRows(f)
	}
	native.Separator()
	drawCommitCancelButtons(s, f.CanCommit())
}

// drawFilletCrossRows draws the cross-section dropdown (circular/G2/conic) and, for a conic, the
// fullness (rho) slider — the Class-A blend shape controls (#1284).
func drawFilletCrossRows(f *app.FilletTool) {
	if i := propertyComboRow("Cross-section", "fillet-cross", app.FilletCrossSectionOptions(), f.CrossSectionIndex()); i >= 0 {
		f.SetCrossSectionIndex(i)
	}
	if app.FilletCrossSectionOptions()[f.CrossSectionIndex()] == "Conic" {
		propertyRow("Fullness (ρ)")
		if native.SliderFloat("##fillet-rho", &filletUI.rho, 0.1, 0.9) {
			f.SetRho(float64(filletUI.rho))
		}
		native.SetItemTooltip("Conic shoulder fullness: 0.5 = parabola, lower = flatter, higher = fuller")
	}
}

// drawFilletCornerRows draws the shared-corner treatment and concave-edge strategy dropdowns.
func drawFilletCornerRows(f *app.FilletTool) {
	if i := propertyComboRow("Corner", "fillet-corner", app.FilletCornerOptions(), f.CornerTypeIndex()); i >= 0 {
		f.SetCornerTypeIndex(i)
	}
	if i := propertyComboRow("Concave edge", "fillet-concave", app.FilletConcaveOptions(), f.ConcaveStrategyIndex()); i >= 0 {
		f.SetConcaveStrategyIndex(i)
	}
}

// seedFilletUI loads the panel buffers from the tool the first frame it appears
// (creation defaults, or a committed fillet's values in edit mode).
func seedFilletUI(f *app.FilletTool) {
	filletUI.radius = float32(f.Radius())
	filletUI.startRadius = float32(f.StartRadius())
	filletUI.endRadius = float32(f.EndRadius())
	filletUI.rho = float32(f.Rho())
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
		lengthCmRow(s, "Radius", "fillet-radius", &filletUI.radius)
		f.SetRadius(float64(filletUI.radius))
		return
	}
	lengthCmRow(s, "Start radius", "fillet-start-radius", &filletUI.startRadius)
	f.SetStartRadius(float64(filletUI.startRadius))
	lengthCmRow(s, "End radius", "fillet-end-radius", &filletUI.endRadius)
	f.SetEndRadius(float64(filletUI.endRadius))
	drawFilletMidPointRows(s, f)
}

// drawFilletMidPointRows renders the intermediate radius stops of a variable fillet (#695): one
// editable Position/Radius pair per stop with a remove (×), plus an Add button. Position is a
// 0–1 fraction along the edge; the radius follows the document length unit. The tool owns the
// stops, so each row reads/writes through it — no panel-side buffer to keep in sync.
func drawFilletMidPointRows(s *app.Session, f *app.FilletTool) {
	for i, p := range f.MidPoints() {
		if drawFilletMidPointRow(s, f, i, p) {
			break // the slice shifted under us when a stop was removed; redraw next frame
		}
	}
	if native.Button("+ Add intermediate point") {
		f.AddMidPoint()
	}
	native.SetItemTooltip("Add a radius stop between the start and end radius of each edge")
}

// drawFilletMidPointRow draws one stop. It returns true when the stop was removed this frame so
// the caller stops iterating the now-stale slice.
func drawFilletMidPointRow(s *app.Session, f *app.FilletTool, i int, p app.FilletMidPoint) bool {
	tBuf := float32(p.T)
	if parameterFloatRow(s, "Position", filletMidID("t", i), paramUnitless, "(0–1)", &tBuf) {
		f.SetMidPointT(i, float64(tBuf))
	}
	native.SameLine()
	if native.Button("×##" + filletMidID("rm", i)) {
		f.RemoveMidPoint(i)
		return true
	}
	rBuf := float32(p.Radius)
	lengthCmRow(s, "Radius", filletMidID("r", i), &rBuf)
	f.SetMidPointR(i, float64(rBuf))
	return false
}

// filletMidID gives a stop's widget a stable, per-index ImGui id.
func filletMidID(field string, i int) string {
	return "fillet-mid-" + field + "-" + strconv.Itoa(i)
}
