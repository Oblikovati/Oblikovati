// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
	"oblikovati/model/sketch"
)

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
	s.feedPick(SketchEntityHandle{Entity: l}) // auto-commits
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
