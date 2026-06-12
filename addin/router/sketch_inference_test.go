// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestInferenceOptionsOverWire: get/set round-trips the preferences; unknown
// priorities are rejected (M06-F10, #625).
func TestInferenceOptionsOverWire(t *testing.T) {
	r, s := seededSession(t)
	var got wire.InferenceOptionsView
	call(t, r, s, "sketch.getInferenceOptions", ``, &got)
	if !got.InferEnabled || !got.ConstrainEnabled || got.Priority != "horizontalVertical" {
		t.Fatalf("defaults = %+v, want everything on with horizontal/vertical priority", got)
	}

	call(t, r, s, "sketch.setInferenceOptions",
		`{"inferEnabled":true,"constrainEnabled":false,"priority":"parallelPerpendicular"}`, &got)
	if got.ConstrainEnabled || got.Priority != "parallelPerpendicular" {
		t.Errorf("after set = %+v, want constrain off / parallelPerpendicular", got)
	}

	if _, err := r.Handle(s, "sketch.setInferenceOptions",
		[]byte(`{"inferEnabled":true,"priority":"psychic"}`)); err == nil {
		t.Error("an unknown priority must be rejected")
	}
}

// TestAddEntityReportsAppliedInference: adding a nearly horizontal line over
// the wire applies the inferred constraint and reports it on the result.
func TestAddEntityReportsAppliedInference(t *testing.T) {
	r, s := seededSession(t)
	var added wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity",
		`{"sketchIndex":0,"kind":"line","points":[[0,8],[4,8.02]]}`, &added)
	if len(added.InferredConstraints) != 1 || added.InferredConstraints[0].Kind != "horizontal" {
		t.Fatalf("inferred constraints = %+v, want one horizontal", added.InferredConstraints)
	}

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	idx := added.InferredConstraints[0].ConstraintIndex
	if idx < 0 || idx >= len(cons.Constraints) || cons.Constraints[idx].Kind != "horizontal" {
		t.Errorf("constraint index %d does not address the applied horizontal", idx)
	}

	// Disabled inference adds nothing and reports nothing.
	call(t, r, s, "sketch.setInferenceOptions", `{"inferEnabled":false}`, &wire.InferenceOptionsView{})
	var quiet wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity",
		`{"sketchIndex":0,"kind":"line","points":[[0,12],[4,12.02]]}`, &quiet)
	if len(quiet.InferredConstraints)+len(quiet.InferredPoints) != 0 {
		t.Errorf("disabled inference reported %+v", quiet)
	}
}
