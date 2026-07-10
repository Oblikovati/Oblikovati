// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// offsetDimSketch adds a line (0,0)-(4,0) and a point (2,3) to sketch 0 and returns the point id
// and line id — the operands of an offset dimension whose perpendicular distance is 3.
func offsetDimSketch(t *testing.T, r *Router, s *app.Session) (point, line uint64) {
	t.Helper()
	var l, p wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &l)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"point","points":[[2,3]]}`, &p)
	return p.EntityID, l.EntityID
}

// TestSketchDimensionDrivenAtCreate creates a driven (reference) offset dimension in one call and
// verifies it neither constrains (DOF unchanged) nor enumerates as driving (#1875).
func TestSketchDimensionDrivenAtCreate(t *testing.T) {
	r, s := seededSession(t)
	p, l := offsetDimSketch(t, r, s)

	var free wire.SolveSketchResult
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &free)

	var res wire.AddDimensionResult
	call(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "offsetDim", Entities: []uint64{p, l}, Expression: "1 cm", Driven: true,
	}), &res)
	if res.DOF != free.DOF {
		t.Errorf("driven dimension changed DOF %d→%d; a reference dim must not constrain", free.DOF, res.DOF)
	}

	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 1 || !list.Dimensions[0].Driven {
		t.Errorf("enumerated dims = %+v, want one driven dim", list.Dimensions)
	}
}

// TestSketchDimensionLinearDiameterDoublesValue creates an offset dimension with LinearDiameter and
// verifies the reported value is twice the 3-unit perpendicular distance, and that enumeration
// echoes the flag (#1875).
func TestSketchDimensionLinearDiameterDoublesValue(t *testing.T) {
	r, s := seededSession(t)
	p, l := offsetDimSketch(t, r, s)

	var res wire.AddDimensionResult
	call(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "offsetDim", Entities: []uint64{p, l}, Expression: "1 cm", LinearDiameter: true,
	}), &res)
	if math.Abs(res.Value-6) > 1e-9 {
		t.Errorf("linear-diameter offset value = %g, want 6 (2× the 3-unit distance)", res.Value)
	}

	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 1 || !list.Dimensions[0].LinearDiameter {
		t.Errorf("enumerated dims = %+v, want one linearDiameter dim", list.Dimensions)
	}
}

// TestSketchDimensionTextPointRoundTrips stores a text placement at create and reads it back from
// enumeration; a malformed textPoint is a clean error (#1875).
func TestSketchDimensionTextPointRoundTrips(t *testing.T) {
	r, s := seededSession(t)
	p, l := offsetDimSketch(t, r, s)

	call(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "offsetDim", Entities: []uint64{p, l}, Expression: "3 cm", TextPoint: []float64{1.5, 2},
	}), &wire.AddDimensionResult{})

	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 1 || len(list.Dimensions[0].TextPoint) != 2 ||
		list.Dimensions[0].TextPoint[0] != 1.5 || list.Dimensions[0].TextPoint[1] != 2 {
		t.Errorf("enumerated textPoint = %+v, want [1.5,2]", list.Dimensions)
	}

	if err := tryCall(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "offsetDim", Entities: []uint64{p, l}, Expression: "3 cm", TextPoint: []float64{1},
	})); err == nil {
		t.Error("a one-element textPoint should error")
	}
}
