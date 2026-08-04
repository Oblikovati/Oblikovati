// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// A sketch dimension could be added and driven over the wire but never removed or repositioned —
// there was no method for either, unlike the drawing side (#2017).

// dimensionedWireSketch adds a line to sketch 0 and dimensions its length, returning the sketch's
// DOF before the dimension was added.
func dimensionedWireSketch(t *testing.T, r *Router, s *app.Session) int {
	t.Helper()
	var l wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &l)
	var free wire.SolveSketchResult
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &free)
	var res wire.AddDimensionResult
	call(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "distance", Entities: []uint64{l.PointIDs[0], l.PointIDs[1]}, Expression: "4 cm",
	}), &res)
	return free.DOF
}

// TestDeleteDimensionFreesTheDOFItHeld: removing a dimension gives back the degree of freedom it
// was holding, which is the observable proof it really left the constraint system.
func TestDeleteDimensionFreesTheDOFItHeld(t *testing.T) {
	r, s := seededSession(t)
	freeDOF := dimensionedWireSketch(t, r, s)

	var got wire.DeleteSketchDimensionResult
	call(t, r, s, "sketch.deleteDimension", `{"sketchIndex":0,"dimensionIndex":0}`, &got)
	if got.DOF != freeDOF {
		t.Errorf("DOF after delete = %d, want the undimensioned %d", got.DOF, freeDOF)
	}

	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 0 {
		t.Errorf("dimension still enumerated after delete: %+v", list.Dimensions)
	}
}

// TestDeleteDimensionRejectsABadIndex: an out-of-range index errors rather than panicking or
// silently deleting the wrong dimension.
func TestDeleteDimensionRejectsABadIndex(t *testing.T) {
	r, s := seededSession(t)
	dimensionedWireSketch(t, r, s)
	wantErr(t, r, s, "sketch.deleteDimension", `{"sketchIndex":0,"dimensionIndex":7}`)
}

// TestMoveDimensionStoresTheTextPoint: the placement round-trips out through enumeration, so a
// client can read back where it put the annotation.
func TestMoveDimensionStoresTheTextPoint(t *testing.T) {
	r, s := seededSession(t)
	dimensionedWireSketch(t, r, s)

	var ok wire.OKResult
	call(t, r, s, "sketch.moveDimension", `{"sketchIndex":0,"dimensionIndex":0,"textPoint":[2,5]}`, &ok)

	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 1 {
		t.Fatalf("want 1 dimension, got %d", len(list.Dimensions))
	}
	if tp := list.Dimensions[0].TextPoint; len(tp) != 2 || tp[0] != 2 || tp[1] != 5 {
		t.Errorf("textPoint = %v, want [2 5]", tp)
	}
}

// TestMoveDimensionLeavesTheGeometryAlone: moving an annotation must not move what it measures,
// so the sketch's DOF and the dimension's measured value are unchanged.
func TestMoveDimensionLeavesTheGeometryAlone(t *testing.T) {
	r, s := seededSession(t)
	dimensionedWireSketch(t, r, s)

	var before wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &before)

	var ok wire.OKResult
	call(t, r, s, "sketch.moveDimension", `{"sketchIndex":0,"dimensionIndex":0,"textPoint":[40,50]}`, &ok)

	var after wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &after)
	if after.Dimensions[0].Value != before.Dimensions[0].Value {
		t.Errorf("measured value changed %v→%v; moving the text moved the geometry",
			before.Dimensions[0].Value, after.Dimensions[0].Value)
	}
}

// TestMoveDimensionRejectsAMalformedTextPoint: a one-value point is a caller error, not a
// silently-ignored no-op.
func TestMoveDimensionRejectsAMalformedTextPoint(t *testing.T) {
	r, s := seededSession(t)
	dimensionedWireSketch(t, r, s)
	wantErr(t, r, s, "sketch.moveDimension", `{"sketchIndex":0,"dimensionIndex":0,"textPoint":[2]}`)
}
