// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchOffsetSplineDimensionOverWire creates a spline, offsets it, dimensions the offset via
// sketch.addDimension kind=offsetSplineDim, and confirms the driven value and enumeration (#1874).
func TestSketchOffsetSplineDimensionOverWire(t *testing.T) {
	r, s := seededSession(t)

	var sp wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0],[1,1],[2,0]]}`, &sp)
	var off wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", mustJSON(t, wire.AddSketchEntityArgs{
		SketchIndex: 0, Kind: "offsetSpline", EntityRefs: []uint64{sp.EntityID}, Radius: "2 cm",
	}), &off)

	var res wire.AddDimensionResult
	call(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "offsetSplineDim", Entities: []uint64{off.EntityID}, Expression: "5 mm",
	}), &res)
	if res.Kind != "offsetSplineDim" {
		t.Errorf("created dim kind = %q, want offsetSplineDim", res.Kind)
	}

	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &wire.SolveSketchResult{})
	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 1 || list.Dimensions[0].Kind != "offsetSplineDim" {
		t.Fatalf("enumerated dims = %+v, want one offsetSplineDim", list.Dimensions)
	}
	if math.Abs(list.Dimensions[0].Value-0.5) > 1e-6 {
		t.Errorf("driven offset value = %g cm, want 0.5 (5 mm)", list.Dimensions[0].Value)
	}
}

// TestSketchOffsetSplineDimensionRejectsNonOffsetSpline: pointing the dimension at a plain spline is
// a clean error (#1874).
func TestSketchOffsetSplineDimensionRejectsNonOffsetSpline(t *testing.T) {
	r, s := seededSession(t)
	var sp wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0],[1,1],[2,0]]}`, &sp)

	if err := tryCall(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "offsetSplineDim", Entities: []uint64{sp.EntityID}, Expression: "5 mm",
	})); err == nil {
		t.Error("dimensioning a plain spline as an offset spline should error")
	}
}
