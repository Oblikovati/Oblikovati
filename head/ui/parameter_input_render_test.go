//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// TestParameterFieldEvalDispatch checks the head's evaluator dispatch (#1519): evalParamField routes
// a field's text — including a formula over the part's parameters ("len * 0.5", "n * 10.5 mm") — to
// the right document-unit evaluator, and paramFieldUnit reports each kind's unit + precision.
func TestParameterFieldEvalDispatch(t *testing.T) {
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "param-field.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	if _, err := def.Parameters().AddUserParameter("len", "20 mm"); err != nil {
		t.Fatalf("add len: %v", err)
	}
	if _, err := def.Parameters().AddUserParameter("n", "3"); err != nil {
		t.Fatalf("add n: %v", err)
	}
	cases := []struct {
		text string
		kind paramFieldKind
		want float64
	}{
		{"len * 0.5", paramLength, 10},     // length formula → mm
		{"n * 10.5 mm", paramLength, 31.5}, // the "D0 * 10.5 mm" shape
		{"45 deg", paramAngle, 45},
		{"n * 2", paramUnitless, 6},
	}
	for _, c := range cases {
		got, ok := evalParamField(s, c.text, c.kind)
		if !ok {
			t.Errorf("evalParamField(%q, %v): did not resolve", c.text, c.kind)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("evalParamField(%q, %v) = %g, want %g", c.text, c.kind, got, c.want)
		}
	}
	if _, ok := evalParamField(s, "len *", paramLength); ok {
		t.Error("an incomplete formula should not resolve")
	}
	if u, p := paramFieldUnit(s, paramLength); u != "mm" || p != s.LengthPrecision() {
		t.Errorf("paramFieldUnit(length) = (%q,%d), want (mm,%d)", u, p, s.LengthPrecision())
	}
	if u, _ := paramFieldUnit(s, paramUnitless); u != "" {
		t.Errorf("paramFieldUnit(unitless) unit = %q, want empty", u)
	}
}

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
