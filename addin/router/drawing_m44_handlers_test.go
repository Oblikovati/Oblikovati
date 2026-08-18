// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestDrawingViewM44HandlersOverWire drives the view surfaces added in M44 — rotation (#1988),
// alignment locks (#1988), tangent-edge display (#1984), crop add/remove (#1987) and view label
// (#1983) — on a base + projected view of the boxed fixture.
func TestDrawingViewM44HandlersOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase",
		`{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	var rotated wire.ListDrawingViewsResult
	call(t, r, s, wire.MethodDrawingViewsRotate, mustJSON(t, wire.RotateViewArgs{Name: "FRONT", AngleDeg: 30}), &rotated)

	tangentOff := false
	call(t, r, s, wire.MethodDrawingViewsSetDisplay,
		mustJSON(t, wire.SetViewDisplayArgs{Name: "FRONT", DisplayTangentEdges: &tangentOff}), nil)

	label := "DETAIL A"
	call(t, r, s, wire.MethodDrawingViewsSetLabel, mustJSON(t, wire.SetViewLabelArgs{Name: "FRONT", Text: &label}), nil)

	// A rectangular crop, then its removal, exercise both crop handlers.
	call(t, r, s, wire.MethodDrawingViewsAddCrop,
		mustJSON(t, wire.AddViewCropArgs{View: "FRONT", Shape: "rectangle", X0: 100, Y0: 80, X1: 140, Y1: 120}), nil)
	call(t, r, s, wire.MethodDrawingViewsRemoveCrop, mustJSON(t, wire.RemoveViewCropArgs{View: "FRONT"}), nil)

	// A projected view aligned to the base exercises the alignment-lock handler.
	call(t, r, s, "drawingViews.addProjected", `{"name":"TOP","baseView":"FRONT","direction":"up"}`, nil)
	var aligned wire.ListDrawingViewsResult
	call(t, r, s, wire.MethodDrawingViewsAlign,
		mustJSON(t, wire.AlignViewArgs{Name: "TOP", AnchorView: "FRONT", Alignment: "vertical"}), &aligned)
	if len(aligned.Views) < 2 {
		t.Fatalf("aligned views = %d, want the base + projected", len(aligned.Views))
	}
}

// TestDrawingDimensionDecorationOverWire covers the dimension text/tolerance/inspection handlers
// (#1990, #1992, #1993, #1996) and the model-dimension retrieve handlers (#1991) — the latter on a
// box with no driven dimensions, so it returns an empty (but valid) set.
func TestDrawingDimensionDecorationOverWire(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase",
		`{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)
	call(t, r, s, "drawingDimensions.addLinear",
		`{"name":"D1","viewName":"FRONT","type":"horizontal","x1":60,"y1":100,"x2":180,"y2":100,"offsetMm":-12}`, nil)

	call(t, r, s, wire.MethodDrawingDimensionsSetTolerance,
		mustJSON(t, wire.SetDimensionToleranceArgs{Name: "D1",
			Tolerance: types.DimensionTolerance{Type: types.SymmetricTolerance, Plus: 0.1}}), nil)
	call(t, r, s, wire.MethodDrawingDimensionsSetInspection,
		mustJSON(t, wire.SetDimensionInspectionArgs{Name: "D1",
			Inspection: types.InspectionDimension{Shape: types.RoundedEndsInspectionBorder, Label: "A", Rate: "100%"}}), nil)
	var styled wire.ListDrawingDimensionsResult
	call(t, r, s, wire.MethodDrawingDimensionsSetTextStyle, `{"name":"D1","prefix":"~","suffix":" REF"}`, &styled)
	if len(styled.Dimensions) != 1 {
		t.Fatalf("dimensions after styling = %d, want 1", len(styled.Dimensions))
	}

	var retr wire.RetrievableDimensionsResult
	call(t, r, s, wire.MethodDrawingDimensionsListRetrievable, `{"viewName":"FRONT"}`, &retr)
	var got wire.RetrievedDimensionsResult
	call(t, r, s, wire.MethodDrawingDimensionsRetrieve, `{"viewName":"FRONT"}`, &got)
}

// TestDrawingNotesRejectPlainBoxEdges: the geometry-derived chamfer/bend notes (#1995) reject a
// plain box's edges (no straight chamfer, no bend cylinder), exercising the handlers' parse path
// and error return without panicking.
func TestDrawingNotesRejectPlainBoxEdges(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase",
		`{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)

	if _, err := r.Handle(s, wire.MethodDrawingAnnotationsAddChamferNote,
		[]byte(`{"viewName":"FRONT","edgeA":"deadbeef","edgeB":"cafe"}`)); err == nil {
		t.Error("chamfer note on non-chamfer edges = ok, want error")
	}
	if _, err := r.Handle(s, wire.MethodDrawingAnnotationsAddBendNote,
		[]byte(`{"viewName":"FRONT","bendEdge":"deadbeef"}`)); err == nil {
		t.Error("bend note on a non-bend edge = ok, want error")
	}
}
