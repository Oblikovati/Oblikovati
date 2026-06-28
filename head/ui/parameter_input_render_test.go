//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// TestParameterInputDialogsRender renders the feature dialogs whose dimensioned fields moved onto
// ParameterInput (#1519) — Extrude (Distance A/B + Taper), Revolve (Angle A), and Offset Plane
// (Offset) — through real frames, so the unit-in-field rows actually execute (and are credited by the
// xvfb+lavapipe CI head job). It asserts nothing about pixels; it guards that the rows draw without
// panicking and exercises the document-unit/precision path. Skips cleanly with no display/Vulkan.
func TestParameterInputDialogsRender(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = newIconCache(win)
	frame := func(draw func()) {
		win.BeginFrame()
		draw()
		win.EndFrame(0.1, 0.1, 0.1)
	}

	// Extrude: Distance A (the field), then asymmetric so Distance B renders, plus the Advanced Taper.
	es := framedSession()
	ext := app.NewExtrudeTool()
	es.StartTool(ext)
	extrudeUI.seeded = nil
	frame(func() { drawExtrudeDialog(es) })
	applyExtrudeDirection(ext, 3) // asymmetric → Distance B row
	ext.SetExtentType(feature.DistanceExtent)
	frame(func() { drawExtrudeDialog(es) })

	// Revolve: the Angle A field (in the document angle unit).
	rs := framedSession()
	rv := app.NewRevolveTool()
	rs.StartTool(rv)
	revolveUI.seeded = nil
	frame(func() { drawRevolveDialog(rs) })

	// Offset Plane: the Offset field (in the document length unit).
	ofs := framedSession()
	ofs.StartTool(app.NewOffsetWorkPlaneTool())
	offsetPlaneUI.open = false
	frame(func() { drawOffsetPlaneDialog(ofs) })
	if ofs.ActiveOffsetPlane() == nil {
		t.Fatal("offset plane tool did not start")
	}

	// Loft Conditions tab with a shaped end condition, so the Impact ParameterInput row renders.
	ls := loftThreeSectionSession(t)
	lf := ls.ActiveLoft()
	loftUI.first.cond, loftUI.last.cond = 1, 2 // Angle / Direction → the angle + impact rows
	frame(func() {
		if native.Begin("##loft-conditions") {
			drawLoftConditionsTab(ls, lf)
		}
		native.End()
	})
}
