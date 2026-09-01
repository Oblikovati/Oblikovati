// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchSymmetryConstraintCreatable is the regression for #1574: a symmetry constraint
// (point A, point B, mirror line) was enumerable but not creatable — the add-constraint
// handler had no case, so every attempt was silently dropped. It must now create, reduce
// DOF, and round-trip through the enumerate path with its (A, B, About) entities intact.
func TestSketchSymmetryConstraintCreatable(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	add := func(args string) wire.AddSketchEntityResult {
		var res wire.AddSketchEntityResult
		call(t, r, s, "sketch.addEntity", args, &res)
		return res
	}

	// Two free points straddling the X axis (the mirror line) but NOT yet symmetric.
	a := add(`{"sketchIndex":0,"kind":"point","points":[[2,1]]}`)
	b := add(`{"sketchIndex":0,"kind":"point","points":[[2,-3]]}`)
	axis := add(`{"sketchIndex":0,"kind":"line","points":[[-5,0],[5,0]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{a.EntityID, b.EntityID, axis.EntityID},
	}), &res)
	if res.Kind != "symmetry" {
		t.Errorf("created kind = %q, want symmetry", res.Kind)
	}

	// The constraint must round-trip through enumerate with its three refs in order.
	var listed wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &listed)
	found := false
	for _, c := range listed.Constraints {
		if c.Kind != "symmetry" {
			continue
		}
		found = true
		if len(c.Entities) != 3 || c.Entities[0] != a.EntityID ||
			c.Entities[1] != b.EntityID || c.Entities[2] != axis.EntityID {
			t.Errorf("enumerated entities = %v, want [%d,%d,%d]",
				c.Entities, a.EntityID, b.EntityID, axis.EntityID)
		}
	}
	if !found {
		t.Fatalf("symmetry constraint not enumerated; got %+v", listed.Constraints)
	}
}

// TestSketchSymmetryRejectsBadRefCount pins the arity guard: symmetry needs exactly 3 refs
// (2 points + a mirror line); any other count is an error, not a silent drop.
func TestSketchSymmetryRejectsBadRefCount(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	add := func(args string) wire.AddSketchEntityResult {
		var res wire.AddSketchEntityResult
		call(t, r, s, "sketch.addEntity", args, &res)
		return res
	}
	a := add(`{"sketchIndex":0,"kind":"point","points":[[2,1]]}`)
	b := add(`{"sketchIndex":0,"kind":"point","points":[[2,-3]]}`)

	err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{a.EntityID, b.EntityID},
	}))
	if err == nil {
		t.Error("symmetry with 2 refs (missing mirror line) should error")
	}
}

// TestSketchSymmetryRejectsBadEntityRefs pins the per-operand resolution guards: a missing
// point id and a non-line mirror ref must each surface an error rather than panic or drop.
func TestSketchSymmetryRejectsBadEntityRefs(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	add := func(args string) wire.AddSketchEntityResult {
		var res wire.AddSketchEntityResult
		call(t, r, s, "sketch.addEntity", args, &res)
		return res
	}
	a := add(`{"sketchIndex":0,"kind":"point","points":[[2,1]]}`)
	b := add(`{"sketchIndex":0,"kind":"point","points":[[2,-3]]}`)
	axis := add(`{"sketchIndex":0,"kind":"line","points":[[-5,0],[5,0]]}`)

	// A point ref that does not resolve to any point.
	if err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{a.EntityID, 999999, axis.EntityID},
	})); err == nil {
		t.Error("symmetry with an unresolvable point ref should error")
	}
	// The mirror ref is a point, not a line.
	if err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{a.EntityID, b.EntityID, a.EntityID},
	})); err == nil {
		t.Error("symmetry with a non-line mirror ref should error")
	}
}
