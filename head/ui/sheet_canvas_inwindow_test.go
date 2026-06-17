//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/drawing"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// drawingWithViewsSession builds a boxed part and an active drawing carrying a base + projected
// view — the fixture the in-window canvas test renders.
func drawingWithViewsSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	part, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -3))
	c1 := sk.Points().Add(math.P2(2, -3))
	c2 := sk.Points().Add(math.P2(2, 3))
	c3 := sk.Points().Add(math.P2(-2, 3))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	if _, err := s.NewDrawing(); err != nil {
		t.Fatalf("NewDrawing: %v", err)
	}
	c, err := app.ActiveDrawing(s)
	if err != nil {
		t.Fatalf("ActiveDrawing: %v", err)
	}
	c.SetModelReference("box.opd")
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(drawing.BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 2, CenterX: 120, CenterY: 100}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if _, err := views.AddProjected(drawing.ProjectedViewSpec{Name: "RIGHT", BaseView: "FRONT", Direction: types.ProjectRight, CenterX: 240, CenterY: 100}); err != nil {
		t.Fatalf("AddProjected: %v", err)
	}
	// A linear dimension across FRONT so the canvas renders the dimension glyph + value text.
	if _, err := c.Sheets().Active().Dimensions().AddLinear("D1", "FRONT", types.HorizontalDimension, 100, 100, 140, 100, -14); err != nil {
		t.Fatalf("AddLinear: %v", err)
	}
	return s
}

// TestInWindowDrawingCanvasRenders drives the real chrome over a few frames on a drawing
// document — exercising the 2D sheet canvas: views (visible/hidden curves), border, title
// block, the selected-view highlight, and the placement-preview / selection input branches.
// Skips when no display/Vulkan is available (CI without a GPU).
func TestInWindowDrawingCanvasRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := drawingWithViewsSession(t)
	c, _ := app.ActiveDrawing(s)
	views := c.Sheets().Active().Views()

	// Frame 1–2: a base-view placement tool is active → the preview branch renders.
	s.StartTool(app.NewBaseViewTool())
	for i := 0; i < 2; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	s.CancelTool()

	// Frame 3–4: a view is selected → the highlight + selection branch render.
	s.Select(app.DrawingViewHandle{Views: views, View: views.Item(0)})
	for i := 0; i < 2; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	if views.Item(0).CurveCount() == 0 {
		t.Error("rendered base view lost its curves")
	}
}

// TestInWindowDrawingSheetTabsRender renders a drawing with two sheets so the sheet-tab strip
// (drawSheetTabs) actually executes — the affordance for switching the active sheet, which only
// appears when a drawing has more than one sheet. Skips when no display/Vulkan is available.
func TestInWindowDrawingSheetTabsRender(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := drawingWithViewsSession(t)
	c, _ := app.ActiveDrawing(s)
	if _, err := c.Sheets().Add(drawing.SheetSpec{Size: types.SheetSizeA3, Orientation: types.SheetLandscape}); err != nil {
		t.Fatalf("add second sheet: %v", err)
	}
	if c.Sheets().Count() != 2 {
		t.Fatalf("want 2 sheets for a tab strip, got %d", c.Sheets().Count())
	}

	// Several frames: the first force-selects the active tab; later frames let ImGui drive —
	// exercising the whole sheet-tab strip (force and steady-state paths).
	for i := 0; i < 4; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}
