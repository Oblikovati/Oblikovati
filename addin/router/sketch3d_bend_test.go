// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketch3DAddBendViaAPI drives the addEntity bend path end to end: two chained
// lines, a 2.5 mm bend, and the auto-added bend constraint reported by enumeration.
func TestSketch3DAddBendViaAPI(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var l1, l2 wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &l1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[1,0,0],[1,1,0]]}`, &l2)

	var bend wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity",
		fmt.Sprintf(`{"sketchIndex":0,"kind":"bend","lines":[%d,%d],"radius":"2.5 mm"}`, l1.EntityID, l2.EntityID), &bend)
	if bend.Kind != "bend" || len(bend.PointIDs) != 3 {
		t.Fatalf("bend result = %+v, want kind bend with center/start/end point ids", bend)
	}

	// The maintaining constraint enumerates as kind "bend" over [arc, l1, l2].
	var cons wire.ListConstraints3DResult
	call(t, r, s, "sketch3d.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 1 || cons.Constraints[0].Kind != "bend" {
		t.Fatalf("constraints = %+v, want one bend", cons.Constraints)
	}
	want := []uint64{bend.EntityID, l1.EntityID, l2.EntityID}
	for i, id := range cons.Constraints[0].Entities {
		if id != want[i] {
			t.Errorf("bend constraint entity[%d] = %d, want %d", i, id, want[i])
		}
	}

	// The arc trims into the chain: line/arc/line still solve as one connected rail.
	var paths wire.ListPaths3DResult
	call(t, r, s, "sketch3d.paths", `{"sketchIndex":0}`, &paths)
	if len(paths.Paths) != 1 {
		t.Errorf("paths after bend = %d, want 1 connected chain", len(paths.Paths))
	}
}

// TestSketch3DAddBendErrors covers the addEntity bend validation paths.
func TestSketch3DAddBendErrors(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var l1, far wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &l1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[5,5,5],[6,5,5]]}`, &far)

	cases := []string{
		fmt.Sprintf(`{"sketchIndex":0,"kind":"bend","lines":[%d],"radius":"1 mm"}`, l1.EntityID),                   // one line
		fmt.Sprintf(`{"sketchIndex":0,"kind":"bend","lines":[%d,%d],"radius":"1 mm"}`, l1.EntityID, far.EntityID),  // no shared corner
		fmt.Sprintf(`{"sketchIndex":0,"kind":"bend","lines":[%d,%d],"radius":"bogus"}`, l1.EntityID, far.EntityID), // bad radius
		fmt.Sprintf(`{"sketchIndex":0,"kind":"bend","lines":[%d,999],"radius":"1 mm"}`, l1.EntityID),               // unknown id
	}
	for _, b := range cases {
		if _, err := r.Handle(s, "sketch3d.addEntity", []byte(b)); err == nil {
			t.Errorf("expected an error for %s", b)
		}
	}
}
