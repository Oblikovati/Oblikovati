// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchTangentAndPointCircle covers the circular-reference geometric constraints
// (tangent line-circle / circle-circle, concentric, point-on-circle).
func TestSketchTangentAndPointCircle(t *testing.T) {
	r, s := seededSession(t)
	add := func(args string) wire.AddSketchEntityResult {
		var res wire.AddSketchEntityResult
		call(t, r, s, "sketch.addEntity", args, &res)
		return res
	}
	con := func(kind string, ents []uint64) error {
		return tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
			SketchIndex: 0, Kind: kind, Entities: ents,
		}))
	}

	c1 := add(`{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,0]],"radius":"2 cm"}`)
	c2 := add(`{"sketchIndex":0,"kind":"circle","variant":"center","points":[[6,0]],"radius":"1 cm"}`)
	line := add(`{"sketchIndex":0,"kind":"line","points":[[0,5],[6,5]]}`)
	pt := add(`{"sketchIndex":0,"kind":"point","points":[[2,0]]}`)

	_ = con("tangent", []uint64{line.EntityID, c1.EntityID})
	_ = con("tangent", []uint64{c1.EntityID, c2.EntityID})
	_ = con("concentric", []uint64{c1.EntityID, c2.EntityID})
	_ = con("pointOnCircle", []uint64{pt.EntityID, c1.EntityID})

	if err := con("tangent", []uint64{c1.EntityID}); err == nil {
		t.Error("tangent with a single ref should error")
	}
	if err := con("pointOnCircle", []uint64{pt.EntityID}); err == nil {
		t.Error("pointOnCircle with a single ref should error")
	}
}
