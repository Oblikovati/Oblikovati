// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// solveAndLineDir solves the sketch and returns the first line's direction components.
func solveAndLineDir(t *testing.T, r *Router, s *app.Session) (float64, float64) {
	t.Helper()
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &wire.SolveSketchResult{})
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	for _, e := range ents.Entities {
		if e.Kind == "line" && len(e.Points) == 2 {
			return e.Points[1][0] - e.Points[0][0], e.Points[1][1] - e.Points[0][1]
		}
	}
	t.Fatal("no line enumerated")
	return 0, 0
}

func nearlyZero(v float64) bool { return v < 1e-6 && v > -1e-6 }

// TestEllipseParallelOverWire: parallel with an ellipse operand aligns the line to the ellipse's
// major axis and enumerates as "parallel" (#1879).
func TestEllipseParallelOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.setInferenceOptions", `{"inferEnabled":false}`, &wire.InferenceOptionsView{})
	ell := addEnt(t, r, s, `{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"axis":[1,0],"majorRadius":"3 cm","minorRadius":"1 cm"}`)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,1]]}`)

	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "parallel", Entities: []uint64{l.EntityID, ell.EntityID},
	}), &res)
	if res.Kind != "parallel" {
		t.Fatalf("result kind = %q, want parallel", res.Kind)
	}
	dx, dy := solveAndLineDir(t, r, s)
	if !nearlyZero(dy) || nearlyZero(dx) {
		t.Errorf("line direction = (%v,%v), want horizontal (parallel to the major axis)", dx, dy)
	}
}

// TestEllipseParallelMinorAxisOverWire: UseEllipseTwoMajorAxis=false selects the minor axis, so
// the line becomes vertical (#1879).
func TestEllipseParallelMinorAxisOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.setInferenceOptions", `{"inferEnabled":false}`, &wire.InferenceOptionsView{})
	ell := addEnt(t, r, s, `{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"axis":[1,0],"majorRadius":"3 cm","minorRadius":"1 cm"}`)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[1,4]]}`)

	no := false
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "parallel", Entities: []uint64{l.EntityID, ell.EntityID},
		UseEllipseTwoMajorAxis: &no,
	}), &res)
	dx, dy := solveAndLineDir(t, r, s)
	if !nearlyZero(dx) || nearlyZero(dy) {
		t.Errorf("line direction = (%v,%v), want vertical (parallel to the minor axis)", dx, dy)
	}
}

// TestEllipsePerpendicularOverWire: perpendicular with an ellipse major-axis operand makes the
// line vertical, enumerating as "perpendicular" (#1879).
func TestEllipsePerpendicularOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.setInferenceOptions", `{"inferEnabled":false}`, &wire.InferenceOptionsView{})
	ell := addEnt(t, r, s, `{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"axis":[1,0],"majorRadius":"3 cm","minorRadius":"1 cm"}`)
	l := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[1,4]]}`)
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "perpendicular", Entities: []uint64{l.EntityID, ell.EntityID},
	}), &res)
	if res.Kind != "perpendicular" {
		t.Fatalf("result kind = %q, want perpendicular", res.Kind)
	}
	dx, dy := solveAndLineDir(t, r, s)
	if !nearlyZero(dx) || nearlyZero(dy) {
		t.Errorf("line direction = (%v,%v), want vertical", dx, dy)
	}
}

// TestTwoLineParallelStillWorks: with no ellipse operand, parallel is the plain two-line
// relation, unchanged by the ellipse-axis path (#1879).
func TestTwoLineParallelStillWorks(t *testing.T) {
	r, s := seededSession(t)
	l1 := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,0],[2,0]]}`)
	l2 := addEnt(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[0,1],[3,2]]}`)
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "parallel", Entities: []uint64{l1.EntityID, l2.EntityID},
	}), &res)
	if res.Kind != "parallel" {
		t.Errorf("two-line parallel kind = %q, want parallel", res.Kind)
	}
}
