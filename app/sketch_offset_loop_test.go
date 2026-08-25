// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// squareLoop adds a closed 4-line square [0,side]^2 to sk and returns its lines.
func squareLoop(sk *sketch.Sketch, side float64) []*sketch.Line {
	c := []math.Point2{math.P2(0, 0), math.P2(side, 0), math.P2(side, side), math.P2(0, side)}
	var ls []*sketch.Line
	for i := 0; i < 4; i++ {
		ls = append(ls, sk.Lines().AddByTwoPoints(c[i], c[(i+1)%4]))
	}
	return ls
}

// TestOffsetToolLoopSelectOffsetsWholeLoop is the Inventor behaviour from the reference video: with
// Loop Select on (default), picking one curve offsets the entire connected loop, inward for a
// positive distance.
func TestOffsetToolLoopSelectOffsetsWholeLoop(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	lines := squareLoop(sk, 4)
	before := sk.Lines().Count()

	tool := NewSketchOffsetTool(0.5)
	if !tool.LoopSelect() {
		t.Fatal("Loop Select must be on by default (Inventor)")
	}
	s.StartTool(tool)
	tool.Pick(s, SketchEntityHandle{Entity: lines[0]})
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := sk.Lines().Count() - before; got != 4 {
		t.Fatalf("Loop Select offset created %d lines, want 4 (the whole loop)", got)
	}
	// the new lines form the inner square [0.5,3.5]^2 — check one is the left edge inset to x=0.5.
	innerLeft := false
	for i := before; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		if absOffset(float64(l.A.Position().X)-0.5) < 1e-6 && absOffset(float64(l.B.Position().X)-0.5) < 1e-6 {
			innerLeft = true
		}
	}
	if !innerLeft {
		t.Error("no inner offset edge at x=0.5 — the loop did not offset inward by 0.5")
	}
}

// TestOffsetToolSingleWhenLoopSelectOff: with Loop Select off, only the picked curve offsets.
func TestOffsetToolSingleWhenLoopSelectOff(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, _ := s.CreateSketch(sketch.XYPlane())
	lines := squareLoop(sk, 4)
	before := sk.Lines().Count()

	tool := NewSketchOffsetTool(0.5)
	tool.SetLoopSelect(false)
	s.StartTool(tool)
	tool.Pick(s, SketchEntityHandle{Entity: lines[0]})
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := sk.Lines().Count() - before; got != 1 {
		t.Fatalf("single-curve offset created %d lines, want 1", got)
	}
}

// TestOffsetToolPlacementSetsSideAndDistance is Inventor's cursor-driven placement: after selecting a
// curve, the click that places the offset sets both its side and its distance from the picked
// geometry. A placement one unit inside a square loop offsets the whole loop inward by exactly one.
func TestOffsetToolPlacementSetsSideAndDistance(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, _ := s.CreateSketch(sketch.XYPlane())
	lines := squareLoop(sk, 4) // CCW [0,4]^2; lines[0] is the bottom edge (0,0)->(4,0)
	before := sk.Lines().Count()

	tool := NewSketchOffsetTool(0.5) // default distance ignored once a placement is set
	s.StartTool(tool)
	tool.Pick(s, SketchEntityHandle{Entity: lines[0]})
	tool.placement, tool.placed = math.P2(2, 1), true // 1 unit inside, above the bottom edge
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := sk.Lines().Count() - before; got != 4 {
		t.Fatalf("placement offset created %d lines, want 4 (whole loop)", got)
	}
	innerBottom := false
	for i := before; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		if absOffset(float64(l.A.Position().Y)-1) < 1e-6 && absOffset(float64(l.B.Position().Y)-1) < 1e-6 {
			innerBottom = true
		}
	}
	if !innerBottom {
		t.Error("no inner edge at y=1 — the placement did not offset inward by its 1-unit distance")
	}
}

// TestPlacementSignedOffset checks the side/distance the cursor placement encodes: the magnitude is
// the perpendicular distance to the curve, the sign the side (left of A->B is positive).
func TestPlacementSignedOffset(t *testing.T) {
	line := []math.Point2{math.P2(0, 0), math.P2(4, 0)} // natural direction +X, left normal +Y
	if d := placementSignedOffset(line, math.P2(2, 3)); absOffset(d-3) > 1e-9 {
		t.Errorf("left placement signed offset = %v, want +3", d)
	}
	if d := placementSignedOffset(line, math.P2(2, -2)); absOffset(d+2) > 1e-9 {
		t.Errorf("right placement signed offset = %v, want -2", d)
	}
}

func absOffset(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
