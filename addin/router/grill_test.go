// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// The #863 acceptance over the wire: the grill op cuts a vent through a wall, leaving the
// boundary profile's rib structure bridging it — one validated solid.
func TestGrillOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	// Wall: a 10×10×1 panel (sketch 0).
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[10,10]]}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"1 cm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})

	// Grill sketch (1): a 6×6 boundary bridged by two ribs (drawn as inner rectangles ⇒ holes).
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"rectangle","points":[[2,2],[8,8]]}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"rectangle","points":[[3.75,2.5],[4.25,7.5]]}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"rectangle","points":[[5.75,2.5],[6.25,7.5]]}`, &struct{}{})

	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"grill","args":{"sketchIndex":1,"boundaries":[0]}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("grill body count = %d, want 1", res.Bodies)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("grill body invalid: %+v", v.Problems)
	}
}

// TestGrillOverWireRejectsEmptyBoundaries: a grill with no boundaries must error.
func TestGrillOverWireRejectsEmptyBoundaries(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[10,10]]}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"1 cm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"grill","args":{"sketchIndex":0,"boundaries":[]}}`)); err == nil {
		t.Fatal("a grill with no boundaries must error")
	}
}
