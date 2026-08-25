// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketchFilletToolInSketchClickRoutes is the #1799 regression: it drives the REAL
// in-sketch pick route (Session.Click → sketchEntityPointer), not feedPick. Before the fix
// the fillet tool did not satisfy SketchEntityTool, so the type assertion in
// sketchEntityPointer failed and the clicks were silently dropped — no arc, no error.
func TestSketchFilletToolInSketchClickRoutes(t *testing.T) {
	s, sk := sketchSession(t)
	corner := sk.Points().Add(math.P2(0, 0)) // origin ↔ pixel (100,100)
	farA, _ := screenToSketch(s, 170, 100)   // a point out along +X
	farB, _ := screenToSketch(s, 100, 170)   // a point out along +Y
	sk.Lines().Add(corner, sk.Points().Add(farA))
	sk.Lines().Add(corner, sk.Points().Add(farB))

	tool := NewSketchFilletTool(1)
	s.StartTool(tool)
	s.Click(135, 100) // lands on the horizontal line (not on an endpoint)
	s.Click(100, 135) // lands on the vertical line → second pick auto-commits

	if s.ActiveTool() != nil {
		t.Fatal("fillet tool still active — the in-sketch clicks did not reach it (#1799)")
	}
	if sk.Arcs().Count() != 1 {
		t.Fatalf("arcs after in-sketch fillet = %d, want 1 (clicks were dropped)", sk.Arcs().Count())
	}
}

// TestSketchModifyToolsAcceptedKinds covers each tool's Accepts hover-candidate filter
// (#1799): fillet/chamfer accept only lines, offset accepts line/circle/arc, mirror accepts
// any geometry but not nil.
func TestSketchModifyToolsAcceptedKinds(t *testing.T) {
	_, sk := sketchSession(t)
	line := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	circle := sk.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	arc := sk.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	point := sk.Points().Add(math.P2(2, 2))

	fillet := NewSketchFilletTool(1)
	chamfer := NewSketchChamferTool(1)
	offset := NewSketchOffsetTool(1)
	mirror := NewSketchMirrorTool()

	cases := []struct {
		name   string
		accept func(sketch.Entity) bool
		yes    []sketch.Entity
		no     []sketch.Entity
	}{
		{"fillet", fillet.Accepts, []sketch.Entity{line}, []sketch.Entity{circle, arc, point, nil}},
		{"chamfer", chamfer.Accepts, []sketch.Entity{line}, []sketch.Entity{circle, arc, point, nil}},
		{"offset", offset.Accepts, []sketch.Entity{line, circle, arc}, []sketch.Entity{point, nil}},
		{"mirror", mirror.Accepts, []sketch.Entity{line, circle, arc, point}, []sketch.Entity{nil}},
	}
	for _, c := range cases {
		for _, e := range c.yes {
			if !c.accept(e) {
				t.Errorf("%s.Accepts(%T) = false, want true", c.name, e)
			}
		}
		for _, e := range c.no {
			if c.accept(e) {
				t.Errorf("%s.Accepts(%T) = true, want false", c.name, e)
			}
		}
	}
}

// TestPickCollectorIgnoresRepeatPick covers the dedup guard: clicking the same line twice
// leaves the tool waiting for a distinct second line, so it does not auto-commit.
func TestPickCollectorIgnoresRepeatPick(t *testing.T) {
	s, sk := sketchSession(t)
	corner := sk.Points().Add(math.P2(0, 0))
	farA, _ := screenToSketch(s, 170, 100)
	sk.Lines().Add(corner, sk.Points().Add(farA))

	tool := NewSketchFilletTool(1)
	s.StartTool(tool)
	s.Click(135, 100)
	s.Click(135, 100) // same line again — must be ignored, no second distinct pick

	if s.ActiveTool() == nil {
		t.Fatal("fillet committed on a duplicate pick — dedup guard failed")
	}
	if got := len(tool.Picked()); got != 1 {
		t.Errorf("picks after duplicate click = %d, want 1", got)
	}
}

func TestSketchFilletToolRoundsCorner(t *testing.T) {
	s, sk := sketchSession(t)
	corner := sk.Points().Add(math.P2(0, 0))
	l1 := sk.Lines().Add(corner, sk.Points().Add(math.P2(4, 0)))
	l2 := sk.Lines().Add(corner, sk.Points().Add(math.P2(0, 4)))

	tool := NewSketchFilletTool(1)
	s.StartTool(tool)
	s.feedPick(SketchEntityHandle{Entity: l1})
	s.feedPick(SketchEntityHandle{Entity: l2}) // second pick auto-commits
	if s.ActiveTool() != nil {
		t.Fatal("fillet tool should deactivate after two picks")
	}
	// A tangent arc of radius 1 now exists, centered at (1,1).
	if sk.Arcs().Count() != 1 {
		t.Fatalf("arcs after fillet = %d, want 1", sk.Arcs().Count())
	}
	if c := sk.Arcs().Item(0).Center.Position(); stdmath.Abs(float64(c.X)-1) > 1e-9 || stdmath.Abs(float64(c.Y)-1) > 1e-9 {
		t.Errorf("fillet arc center = %v, want (1,1)", c)
	}
}

func TestSketchOffsetToolOffsetsCurve(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	tool := NewSketchOffsetTool(2)
	s.StartTool(tool)
	tool.Pick(s, SketchEntityHandle{Entity: l}) // select the curve
	if err := s.OK(); err != nil {              // OK finishes with the default distance (no placement click)
		t.Fatalf("OK: %v", err)
	}
	if sk.Lines().Count() != 2 {
		t.Fatalf("lines after offset = %d, want 2", sk.Lines().Count())
	}
	// The offset line sits at y=2 (left normal of +X).
	off := sk.Lines().Item(1)
	if stdmath.Abs(float64(off.A.Y)-2) > 1e-9 {
		t.Errorf("offset line at y=%v, want 2", off.A.Y)
	}
}

func TestSketchMirrorToolReflects(t *testing.T) {
	s, sk := sketchSession(t)
	dot := sk.Points().Add(math.P2(3, 2))
	axis := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 1)) // Y axis
	tool := NewSketchMirrorTool()
	s.StartTool(tool)
	s.feedPick(SketchEntityHandle{Entity: dot})
	s.feedPick(SketchEntityHandle{Entity: axis}) // auto-commits
	// A mirrored point at (-3,2) now exists.
	if !pointExistsAt(sk, math.P2(-3, 2)) {
		t.Fatal("no mirrored point at (-3,2)")
	}
}

func TestSketchModifyCommandsRegistered(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{"Sketch.Offset", "Sketch.Mirror", "Sketch.Fillet"} {
		if _, ok := s.Commands().ByID(id); !ok {
			t.Errorf("command %q not registered", id)
		}
	}
}

// pointExistsAt reports whether any constrainable point sits at q.
func pointExistsAt(sk *sketch.Sketch, q math.Point2) bool {
	for _, p := range sk.AllPoints() {
		if stdmath.Abs(float64(p.X-q.X)) < 1e-9 && stdmath.Abs(float64(p.Y-q.Y)) < 1e-9 {
			return true
		}
	}
	return false
}
