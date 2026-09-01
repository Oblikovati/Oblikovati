// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// twoBodyPartSession builds a part with two separate box bodies (a 4×4 square at the origin and
// another offset in +X), each extruded as a new body.
func twoBodyPartSession(t *testing.T) (*Router, *app.Session, *compdef.PartComponentDefinition) {
	t.Helper()
	r, s := emptyPartSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	for _, ox := range []float64{0, 10} {
		sk := def.Sketches().Add(sketch.XYPlane())
		c0 := sk.Points().Add(math.P2(ox, 0))
		c1 := sk.Points().Add(math.P2(ox+4, 0))
		c2 := sk.Points().Add(math.P2(ox+4, 4))
		c3 := sk.Points().Add(math.P2(ox, 4))
		sk.Lines().Add(c0, c1)
		sk.Lines().Add(c1, c2)
		sk.Lines().Add(c2, c3)
		sk.Lines().Add(c3, c0)
		feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	}
	def.Recompute()
	return r, s, def
}

// TestBodyDeleteRemovesOneBody drives the #1078 delete acceptance: two bodies, delete the first,
// one remains and the call returns the refreshed list.
func TestBodyDeleteRemovesOneBody(t *testing.T) {
	t.Parallel()
	r, s, def := twoBodyPartSession(t)
	if n := len(def.SurfaceBodies().All()); n != 2 {
		t.Fatalf("setup: %d bodies, want 2", n)
	}

	var list wire.BodyListResult
	call(t, r, s, "body.delete", `{"bodyIndex":0}`, &list)
	if len(list.Bodies) != 1 {
		t.Fatalf("body.delete returned %d bodies, want 1", len(list.Bodies))
	}
	if n := len(def.SurfaceBodies().All()); n != 1 {
		t.Errorf("after delete the part has %d bodies, want 1", n)
	}
}

// TestBodyDeleteIsRecordedInHistory: delete is a history feature, so it appends a recipe step
// (making it undoable / re-computable).
func TestBodyDeleteIsRecordedInHistory(t *testing.T) {
	t.Parallel()
	r, s, def := twoBodyPartSession(t)
	before := def.Features().Count()
	call(t, r, s, "body.delete", `{"bodyIndex":1}`, &wire.BodyListResult{})
	if after := def.Features().Count(); after != before+1 {
		t.Errorf("feature count = %d, want %d (delete appends one history step)", after, before+1)
	}
}

// TestBodyDeleteBadIndexFails: an out-of-range body index is a rejection.
func TestBodyDeleteBadIndexFails(t *testing.T) {
	t.Parallel()
	r, s, _ := twoBodyPartSession(t)
	if _, err := r.Handle(s, "body.delete", []byte(`{"bodyIndex":5}`)); err == nil {
		t.Error("body.delete with an out-of-range index should fail")
	}
}
