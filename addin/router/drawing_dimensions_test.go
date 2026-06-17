// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

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
