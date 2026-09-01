// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSingleLineHorizontalOverWire: kind=horizontal with a single line ref makes the line
// horizontal and both the result and enumeration report "horizontal", relating the line (#1871).
func TestSingleLineHorizontalOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,1]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "horizontal", Entities: []uint64{l.EntityID},
	}), &res)
	if res.Kind != "horizontal" {
		t.Fatalf("single-line result kind = %q, want horizontal", res.Kind)
	}
	for _, c := range enumerated(t, r, s) {
		if c.Kind == "horizontal" {
			if len(c.Entities) != 1 || c.Entities[0] != l.EntityID {
				t.Fatalf("horizontal relates %v, want just the line %d", c.Entities, l.EntityID)
			}
			return
		}
	}
	t.Fatal("no horizontal constraint enumerated after single-line horizontal")
}

// TestTwoPointHorizontalIsAlign: kind=horizontal with two point refs creates the align form,
// reported as "horizontalAlign" — distinct from the single-line horizontal (#1871).
func TestTwoPointHorizontalIsAlign(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	a := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[0,0]]}`)
	b := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[4,1]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "horizontal", Entities: []uint64{a.EntityID, b.EntityID},
	}), &res)
	if res.Kind != "horizontalAlign" {
		t.Fatalf("two-point horizontal result kind = %q, want horizontalAlign", res.Kind)
	}
	found := false
	for _, c := range enumerated(t, r, s) {
		if c.Kind == "horizontalAlign" {
			found = true
		}
		if c.Kind == "horizontal" {
			t.Errorf("two-point horizontal enumerated as %q, want horizontalAlign", c.Kind)
		}
	}
	if !found {
		t.Fatal("no horizontalAlign constraint enumerated after two-point horizontal")
	}
}

// TestExplicitAlignKinds: the explicit horizontalAlign/verticalAlign kinds create the two-point
// align forms (#1871).
func TestExplicitAlignKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"horizontalAlign", "verticalAlign"} {
		t.Run(kind, func(t *testing.T) {
			r, s := seededSession(t)
			a := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[0,0]]}`)
			b := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[4,1]]}`)
			var res wire.AddConstraintResult
			call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
				SketchIndex: 0, Kind: kind, Entities: []uint64{a.EntityID, b.EntityID},
			}), &res)
			if res.Kind != kind {
				t.Errorf("result kind = %q, want %q", res.Kind, kind)
			}
		})
	}
}

// TestSingleLineVerticalOverWire: the vertical single-line twin (#1871).
func TestSingleLineVerticalOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[1,4]]}`)
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "vertical", Entities: []uint64{l.EntityID},
	}), &res)
	if res.Kind != "vertical" {
		t.Errorf("single-line vertical result kind = %q, want vertical", res.Kind)
	}
}

// TestHorizontalBadRefCountErrors: neither 0 nor 3 refs is a valid horizontal form (#1871).
func TestHorizontalBadRefCountErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	a := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[0,0]]}`)
	b := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[4,1]]}`)
	c := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[8,2]]}`)
	if err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "horizontal", Entities: []uint64{a.EntityID, b.EntityID, c.EntityID},
	})); err == nil {
		t.Error("horizontal with 3 refs should error")
	}
}
