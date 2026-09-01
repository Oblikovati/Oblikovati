// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// addEnt is a small helper: add one sketch entity and return its result.
func addEnt(t *testing.T, r *Router, s *app.Session, args string) wire.AddSketchEntityResult {
	t.Helper()
	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", args, &res)
	return res
}

// enumerated collects the enumerated constraints of the first sketch.
func enumerated(t *testing.T, r *Router, s *app.Session) []wire.ConstraintInfo {
	t.Helper()
	var listed wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &listed)
	return listed.Constraints
}

// TestSketchLineSymmetryCreatable: kind=symmetry with two line refs + an axis makes the lines
// symmetric and enumerates (coarsely) as "symmetry" with the two lines + axis (#1870).
func TestSketchLineSymmetryCreatable(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	l1 := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[1,0],[2,1]]}`)
	l2 := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[-1,0],[-2,1]]}`)
	axis := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,-1],[0,2]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{l1.EntityID, l2.EntityID, axis.EntityID},
	}), &res)
	if res.Kind != "symmetry" {
		t.Fatalf("created kind = %q, want symmetry", res.Kind)
	}
	for _, c := range enumerated(t, r, s) {
		if c.Kind == "symmetry" && len(c.Entities) == 3 && c.Entities[0] == l1.EntityID &&
			c.Entities[1] == l2.EntityID && c.Entities[2] == axis.EntityID {
			return
		}
	}
	t.Fatalf("line symmetry not enumerated with its two lines + axis")
}

// TestSketchCircularSymmetryCreatable: kind=symmetry with two circle refs + an axis (#1870).
func TestSketchCircularSymmetryCreatable(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	c1 := addEnt(t, r, s, `{"sketchIndex":0,"kind":"circle","points":[[3,1]],"radius":"2"}`)
	c2 := addEnt(t, r, s, `{"sketchIndex":0,"kind":"circle","points":[[-3,1]],"radius":"2"}`)
	axis := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,-1],[0,2]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{c1.EntityID, c2.EntityID, axis.EntityID},
	}), &res)
	if res.Kind != "symmetry" {
		t.Fatalf("created kind = %q, want symmetry", res.Kind)
	}
}

// TestSketchSymmetryMixedOperandsError: two operands of different kinds (a line and a circle)
// are not a matching pair and must error clearly, not silently drop (#1870 AC4).
func TestSketchSymmetryMixedOperandsError(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[1,0],[2,1]]}`)
	c := addEnt(t, r, s, `{"sketchIndex":0,"kind":"circle","points":[[-3,1]],"radius":"2"}`)
	axis := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,-1],[0,2]]}`)
	if err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "symmetry", Entities: []uint64{l.EntityID, c.EntityID, axis.EntityID},
	})); err == nil {
		t.Error("symmetry with mixed line/circle operands should error")
	}
}

// TestSketchArcMidpointCreatable: kind=midpoint with [point, arc] constrains the point to the
// arc midpoint and enumerates (coarsely) as "midpoint" with [point, arc] (#1872).
func TestSketchArcMidpointCreatable(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	arc := addEnt(t, r, s, `{"sketchIndex":0,"kind":"arc","variant":"centerStartEnd","points":[[0,0],[2,0],[0,2]],"ccw":true}`)
	p := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[5,5]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "midpoint", Entities: []uint64{p.EntityID, arc.EntityID},
	}), &res)
	if res.Kind != "midpoint" {
		t.Fatalf("created kind = %q, want midpoint", res.Kind)
	}
	for _, c := range enumerated(t, r, s) {
		if c.Kind == "midpoint" && len(c.Entities) == 2 && c.Entities[0] == p.EntityID &&
			c.Entities[1] == arc.EntityID {
			return
		}
	}
	t.Fatalf("arc midpoint not enumerated with [point, arc]")
}

// TestSketchMidpointLineStillWorks: the line form of midpoint is unchanged (#1872).
func TestSketchMidpointLineStillWorks(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`)
	p := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[2,3]]}`)
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "midpoint", Entities: []uint64{p.EntityID, l.EntityID},
	}), &res)
	if res.Kind != "midpoint" {
		t.Errorf("line midpoint kind = %q, want midpoint", res.Kind)
	}
}

// TestSketchMidpointCircleErrors: a circle has no defined midpoint, so [point, circle] errors
// clearly rather than being accepted (#1872 AC2).
func TestSketchMidpointCircleErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	c := addEnt(t, r, s, `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"2"}`)
	p := addEnt(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[2,3]]}`)
	if err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "midpoint", Entities: []uint64{p.EntityID, c.EntityID},
	})); err == nil {
		t.Error("midpoint with a circle host (no defined midpoint) should error")
	}
}
