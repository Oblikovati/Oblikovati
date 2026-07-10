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

// pinEllipseHorizontal fixes an axis-aligned ellipse's orientation DOF (its axis is already +X, so
// this only removes the rotational freedom) so a following relation moves the LINE, not the ellipse
// (#1879 AC2).
func pinEllipseHorizontal(t *testing.T, r *Router, s *app.Session, ellID uint64) {
	t.Helper()
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "horizontal", Entities: []uint64{ellID},
	}), &wire.AddConstraintResult{})
}

// TestEllipseParallelOverWire: parallel with an ellipse operand aligns the line to the ellipse's
// major axis and enumerates as "parallel" (#1879).
func TestEllipseParallelOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.setInferenceOptions", `{"inferEnabled":false}`, &wire.InferenceOptionsView{})
	ell := addEnt(t, r, s, `{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"axis":[1,0],"majorRadius":"3 cm","minorRadius":"1 cm"}`)
	pinEllipseHorizontal(t, r, s, ell.EntityID) // the ellipse's orientation is a DOF now (#1879 AC2)
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
	pinEllipseHorizontal(t, r, s, ell.EntityID) // pin the ellipse's rotation DOF (#1879 AC2)
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
	pinEllipseHorizontal(t, r, s, ell.EntityID) // pin the ellipse's rotation DOF (#1879 AC2)
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

// ellipseHVSession builds an inference-free sketch with a single 45° ellipse and returns its id.
func ellipseHVSession(t *testing.T) (*Router, *app.Session, uint64) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.setInferenceOptions", `{"inferEnabled":false}`, &wire.InferenceOptionsView{})
	ell := addEnt(t, r, s, `{"sketchIndex":0,"kind":"ellipse","points":[[0,0]],"axis":[1,1],"majorRadius":"3 cm","minorRadius":"1 cm"}`)
	return r, s, ell.EntityID
}

// assertHVRelatesEllipse checks a single constraint of the given kind is enumerated and relates
// exactly the ellipse.
func assertHVRelatesEllipse(t *testing.T, r *Router, s *app.Session, kind string, ellID uint64) {
	t.Helper()
	for _, c := range enumerated(t, r, s) {
		if c.Kind == kind {
			if len(c.Entities) != 1 || c.Entities[0] != ellID {
				t.Fatalf("%s relates %v, want just the ellipse %d", kind, c.Entities, ellID)
			}
			return
		}
	}
	t.Fatalf("no %s constraint enumerated after ellipse %s", kind, kind)
}

// TestEllipseHorizontalOverWire: kind=horizontal with a single ellipse ref rotates the ellipse's
// major axis to horizontal, reports and enumerates as "horizontal", and relates the ellipse
// (#1879 AC2).
func TestEllipseHorizontalOverWire(t *testing.T) {
	r, s, ell := ellipseHVSession(t)
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "horizontal", Entities: []uint64{ell},
	}), &res)
	if res.Kind != "horizontal" {
		t.Fatalf("ellipse horizontal result kind = %q, want horizontal", res.Kind)
	}
	assertHVRelatesEllipse(t, r, s, "horizontal", ell)
	var sr wire.SolveSketchResult
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &sr)
	if !sr.Converged {
		t.Errorf("solve did not converge after ellipse horizontal: %+v", sr)
	}
}

// TestEllipseVerticalOverWire: kind=vertical with a single ellipse ref makes the ellipse's axis
// vertical, enumerating as "vertical" (#1879 AC2).
func TestEllipseVerticalOverWire(t *testing.T) {
	r, s, ell := ellipseHVSession(t)
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "vertical", Entities: []uint64{ell},
	}), &res)
	if res.Kind != "vertical" {
		t.Fatalf("ellipse vertical result kind = %q, want vertical", res.Kind)
	}
	assertHVRelatesEllipse(t, r, s, "vertical", ell)
}

// TestEllipseHorizontalMinorAxisOverWire: UseEllipseMajorAxis=false selects the minor axis for the
// single-operand horizontal form; it still creates and enumerates a horizontal constraint (#1879
// AC2). The major/minor distinction is geometric (not wire-observable), so the geometry is covered
// by the model-level tests.
func TestEllipseHorizontalMinorAxisOverWire(t *testing.T) {
	r, s, ell := ellipseHVSession(t)
	no := false
	var res wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
		SketchIndex: 0, Kind: "horizontal", Entities: []uint64{ell}, UseEllipseMajorAxis: &no,
	}), &res)
	if res.Kind != "horizontal" {
		t.Fatalf("ellipse minor horizontal result kind = %q, want horizontal", res.Kind)
	}
	assertHVRelatesEllipse(t, r, s, "horizontal", ell)
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
