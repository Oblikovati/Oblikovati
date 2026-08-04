// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// A setGroup write carries only the fields its wire view exposes. Building a fresh option
// struct from those fields zeroes every option the view leaves out, so an add-in that set the
// grid spacing used to switch the whole heads-up display off behind the user's back — and
// persist it. These lock the merge (#2016).

func TestSetSketchOptionsKeepsUnexposedFlags(t *testing.T) {
	r, s := seededSession(t)
	before := s.Options().Sketch
	if !before.HeadsUpDisplay || !before.PointerInput || !before.DimensionInput ||
		!before.CreateDimensionsOnValueInput {
		t.Fatalf("precondition: the HUD flags must start on, got %+v", before)
	}

	call(t, r, s, "options.setGroup",
		`{"group":"sketch","sketch":{"gridSpacingCm":2,"gridVisible":true,"gridMajorEvery":5,"snapToPoints":true,"snapToGrid":true,"autoProjectOrigin":true}}`, nil)

	after := s.Options().Sketch
	if !after.HeadsUpDisplay || !after.PointerInput || !after.DimensionInput ||
		!after.CreateDimensionsOnValueInput {
		t.Errorf("a grid-only write cleared the HUD flags: %+v", after)
	}
	if after.GridSpacingCm != 2 {
		t.Errorf("grid spacing = %v, want the written 2", after.GridSpacingCm)
	}
}

func TestSetPartOptionsKeepsUnexposedFlags(t *testing.T) {
	r, s := seededSession(t)
	if !s.Options().Part.TangentChainSelect {
		t.Fatal("precondition: tangent-chain select must start on")
	}

	call(t, r, s, "options.setGroup", `{"group":"part","part":{"chamferFlatCorners":false}}`, nil)

	after := s.Options().Part
	if !after.TangentChainSelect {
		t.Error("a chamfer-only write cleared TangentChainSelect")
	}
	if after.ChamferFlatCorners {
		t.Error("chamferFlatCorners = true, want the written false")
	}
}

// The sketch view carries autoProjectOrigin, so it round-trips like any exposed field.
func TestAutoProjectOriginRoundTripsOverWire(t *testing.T) {
	r, s := seededSession(t)
	var view wire.OptionGroupView
	call(t, r, s, "options.getGroup", `{"group":"sketch"}`, &view)
	if view.Sketch == nil || !view.Sketch.AutoProjectOrigin {
		t.Fatalf("sketch = %+v, want autoProjectOrigin on by default", view.Sketch)
	}

	call(t, r, s, "options.setGroup",
		`{"group":"sketch","sketch":{"gridSpacingCm":1,"gridVisible":true,"gridMajorEvery":5,"snapToPoints":true,"snapToGrid":true,"autoProjectOrigin":false}}`, nil)

	call(t, r, s, "options.getGroup", `{"group":"sketch"}`, &view)
	if view.Sketch == nil || view.Sketch.AutoProjectOrigin {
		t.Errorf("sketch = %+v, want autoProjectOrigin off after the write", view.Sketch)
	}
}
