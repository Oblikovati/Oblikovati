// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchDimensionKinds drives sketch.addDimension across distance/angle/radius/
// diameter/arcLength against freshly-added geometry.
func TestSketchDimensionKinds(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	add := func(args string) wire.AddSketchEntityResult {
		var res wire.AddSketchEntityResult
		call(t, r, s, "sketch.addEntity", args, &res)
		return res
	}
	dim := func(kind string, entities []uint64, expr string) {
		_ = tryCall(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
			SketchIndex: 0, Kind: kind, Entities: entities, Expression: expr,
		}))
	}

	l1 := add(`{"sketchIndex":0,"kind":"line","points":[[0,0],[5,0]]}`)
	dim("distance", l1.PointIDs, "5 cm")

	l2 := add(`{"sketchIndex":0,"kind":"line","points":[[0,0],[0,5]]}`)
	dim("angle", []uint64{l1.EntityID, l2.EntityID}, "90 deg")

	c := add(`{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,0]],"radius":"2 cm"}`)
	dim("radius", []uint64{c.EntityID}, "2 cm")
	dim("diameter", []uint64{c.EntityID}, "4 cm")

	arc := add(`{"sketchIndex":0,"kind":"arc","points":[[0,0],[2,0],[1,1]]}`)
	dim("arcLength", []uint64{arc.EntityID}, "3 cm")

	// Unknown entity refs are a clean error.
	if err := tryCall(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{
		SketchIndex: 0, Kind: "distance", Entities: []uint64{99998, 99999}, Expression: "1 cm",
	})); err == nil {
		t.Error("a distance dimension over unknown points should error")
	}
}
