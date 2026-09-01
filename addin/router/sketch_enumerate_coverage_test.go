// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchEnumerateShapes builds a sketch with varied entity, constraint, and dimension
// kinds, then enumerates each — exercising the point/line/circular shape classifiers.
func TestSketchEnumerateShapes(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	add := func(args string) wire.AddSketchEntityResult {
		var res wire.AddSketchEntityResult
		call(t, r, s, "sketch.addEntity", args, &res)
		return res
	}

	l1 := add(`{"sketchIndex":0,"kind":"line","points":[[0,0],[5,0]]}`)
	l2 := add(`{"sketchIndex":0,"kind":"line","points":[[0,0],[0,5]]}`)
	c := add(`{"sketchIndex":0,"kind":"circle","variant":"center","points":[[3,3]],"radius":"1 cm"}`)
	_ = add(`{"sketchIndex":0,"kind":"arc","points":[[0,0],[2,0],[1,1]]}`)

	// A couple of geometric constraints over those entities.
	_ = tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{SketchIndex: 0, Kind: "perpendicular", Entities: []uint64{l1.EntityID, l2.EntityID}}))
	_ = tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{SketchIndex: 0, Kind: "coincident", Entities: l1.PointIDs}))
	// A radius dimension on the circle.
	_ = tryCall(t, r, s, "sketch.addDimension", mustJSON(t, wire.AddDimensionArgs{SketchIndex: 0, Kind: "radius", Entities: []uint64{c.EntityID}, Expression: "1 cm"}))

	// Enumerate each surface — the classifiers run for every shape.
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, nil)
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, nil)
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, nil)

	if err := tryCall(t, r, s, "sketch.entities", `{"sketchIndex":9}`); err == nil {
		t.Error("enumerating an out-of-range sketch should error")
	}
}
