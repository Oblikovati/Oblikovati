// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// PBI-172 (#469) over the wire: combine cut/intersect of two overlapping extruded boxes
// produces one validated solid. Join + split round-trips are covered in
// split_combine_m09_test.go; this exercises the remaining boolean operations end to end.

// twoOverlappingBoxes builds box A (x∈[0,2]) and box B (x∈[1,3]), both 2×2×2, leaving two
// solid bodies whose overlap is 1×2×2.
func twoOverlappingBoxes(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[2,2]]}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"2 cm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"rectangle","points":[[1,0],[3,2]]}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":1,"profileIndex":0,"distance":"2 cm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	return r, s
}

func TestCombineCutOverWire(t *testing.T) {
	t.Parallel()
	r, s := twoOverlappingBoxes(t)
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"combine","args":{"targetIndex":0,"toolIndex":1,"operation":"cut"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("combine cut bodies = %d, want 1", res.Bodies)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("cut body invalid: %+v", v.Problems)
	}
}

func TestCombineIntersectOverWire(t *testing.T) {
	t.Parallel()
	r, s := twoOverlappingBoxes(t)
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"combine","args":{"targetIndex":0,"toolIndex":1,"operation":"intersect"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("combine intersect bodies = %d, want 1", res.Bodies)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("intersect body invalid: %+v", v.Problems)
	}
}
