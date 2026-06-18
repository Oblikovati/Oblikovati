// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// drawingCylinderSession builds a 2 cm-radius cylinder part + a drawing of it — the fixture for
// radial (radius/diameter) dimension tests, since a box has no circular edges.
func drawingCylinderSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r := New(opregistry.Default())
	s := app.NewSession()
	part, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	call(t, r, s, "documents.create", `{"type":"drawing","name":"box.odd"}`, nil)
	call(t, r, s, "drawing.setModelReference", `{"fullDocumentName":"box.opd"}`, nil)
	return r, s
}

// TestDrawingRadialDimensionsOverWire drives the radial-dimension surface: a diameter dimension on
// a cylinder's circular edge re-measures the true 40 mm diameter through the live stack.
func TestDrawingRadialDimensionsOverWire(t *testing.T) {
	r, s := drawingCylinderSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"TOP","orientation":"top","scale":1,"centerXmm":100,"centerYmm":100}`, nil)

	var dim wire.DimensionResult
	call(t, r, s, "drawingDimensions.addRadial",
		`{"name":"D1","viewName":"TOP","type":"diameter","pickXmm":100,"pickYmm":100}`, &dim)
	if dim.Dimension.Type != "diameter" || math.Abs(dim.Dimension.ValueMM-40) > 1e-6 || dim.Dimension.CurveCount == 0 {
		t.Fatalf("radial dimension = %+v, want a diameter ⌀40 with glyph", dim.Dimension)
	}
	if _, err := r.Handle(s, "drawingDimensions.addRadial", []byte(`{"viewName":"TOP","type":"bogus","pickXmm":100,"pickYmm":100}`)); err == nil {
		t.Error("addRadial with a bad type = ok, want error")
	}
}

// TestDrawingAngularDimensionOverWire drives the angular-dimension surface: the angle between two
// perpendicular box edges re-derives 90° through the live stack, reported in valueDeg.
func TestDrawingAngularDimensionOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":1,"centerXmm":100,"centerYmm":100}`, nil)

	var dim wire.DimensionResult
	call(t, r, s, "drawingDimensions.addAngular", `{"name":"A1","viewName":"FRONT","x1":100,"y1":80,"x2":120,"y2":100}`, &dim)
	if dim.Dimension.Type != "angular" || math.Abs(dim.Dimension.ValueDeg-90) > 1e-6 {
		t.Fatalf("angular dimension = %+v, want a 90° angle", dim.Dimension)
	}
	if dim.Dimension.CurveCount == 0 || dim.Dimension.Text == "" {
		t.Errorf("angular dimension = %+v, want arc glyph + text", dim.Dimension)
	}
}

// TestDrawingDimensionsOverWire drives the linear-dimension surface: add a horizontal dimension
// across a base view, list it, and delete it — through the live router→model→kernel stack.
func TestDrawingDimensionsOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	var dim wire.DimensionResult
	call(t, r, s, "drawingDimensions.addLinear",
		`{"name":"D1","viewName":"FRONT","type":"horizontal","x1":60,"y1":100,"x2":180,"y2":100,"offsetMm":-12}`, &dim)
	if dim.Dimension.ViewName != "FRONT" || dim.Dimension.Type != "horizontal" {
		t.Fatalf("dimension = %+v, want a horizontal dim on FRONT", dim.Dimension)
	}
	if dim.Dimension.ValueMM <= 0 || dim.Dimension.CurveCount == 0 || dim.Dimension.Text == "" {
		t.Fatalf("dimension = %+v, want a positive measured value with glyph + text", dim.Dimension)
	}

	var list wire.ListDrawingDimensionsResult
	call(t, r, s, "drawingDimensions.list", "{}", &list)
	if len(list.Dimensions) != 1 {
		t.Fatalf("dimensions = %d, want 1", len(list.Dimensions))
	}
	var after wire.ListDrawingDimensionsResult
	call(t, r, s, "drawingDimensions.delete", `{"name":"D1"}`, &after)
	if len(after.Dimensions) != 0 {
		t.Errorf("dimensions after delete = %d, want 0", len(after.Dimensions))
	}
}

// TestDrawingDimensionsRejectBadArgs: an unknown type, an unknown view and a missing-name delete
// all error rather than panic.
func TestDrawingDimensionsRejectBadArgs(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)
	for method, args := range map[string]string{
		"drawingDimensions.addLinear": `{"viewName":"FRONT","type":"bogus","x1":0,"y1":0,"x2":10,"y2":0}`,
		"drawingDimensions.delete":    `{"name":"missing"}`,
	} {
		if _, err := r.Handle(s, method, []byte(args)); err == nil {
			t.Errorf("%s(%s) = ok, want error", method, args)
		}
	}
	if _, err := r.Handle(s, "drawingDimensions.addLinear", []byte(`{"viewName":"NOPE","x1":0,"y1":0,"x2":10,"y2":0}`)); err == nil {
		t.Error("addLinear on a missing view = ok, want error")
	}
}
