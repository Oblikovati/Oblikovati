// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchDistanceDimensionOrientation drives sketch.addDimension with a horizontal orientation
// over two points at (0,0) and (3,4): the measured value is 3 (|Δx|), not the Euclidean 5, and
// sketch.dimensions reports the orientation. An unknown orientation is a clean error. #1869.
func TestSketchDistanceDimensionOrientation(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var l wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[3,4]]}`, &l)

	var res wire.AddDimensionResult
	call(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "distance", Orientation: "horizontal", Entities: l.PointIDs, Expression: "1 cm",
	}), &res)
	if math.Abs(res.Value-3) > 1e-9 {
		t.Errorf("horizontal distance measured = %g, want 3 (|Δx|, not the Euclidean 5)", res.Value)
	}

	var list wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &list)
	if len(list.Dimensions) != 1 || list.Dimensions[0].Orientation != "horizontal" {
		t.Errorf("enumerated orientation = %+v, want one horizontal dim", list.Dimensions)
	}

	if err := tryCall(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "distance", Orientation: "diagonal", Entities: l.PointIDs, Expression: "1 cm",
	})); err == nil {
		t.Error("an unknown orientation should error")
	}
}
