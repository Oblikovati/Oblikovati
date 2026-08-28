// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// squareChain adds a closed CCW square [0,side]^2 of four lines and returns the connected path from
// its first edge — the loop an offset Loop Select would follow.
func squareChain(s *Sketch, side float64) (*Path, []*Line) {
	c := []gmath.Point2{gmath.P2(0, 0), gmath.P2(side, 0), gmath.P2(side, side), gmath.P2(0, side)}
	ls := make([]*Line, 4)
	for i := range 4 {
		ls[i] = s.Lines().AddByTwoPoints(c[i], c[(i+1)%4])
	}
	path, _ := s.ConnectedChainFrom(ls[0])
	return path, ls
}

// TestConstrainOffsetLoopBindsAndSolves is Inventor's Constrain Offset: a constrained loop offset
// adds a parallel constraint per line and a coincident join per corner, and the sketch still solves
// cleanly (the constraints are satisfied at creation, so nothing moves).
func TestConstrainOffsetLoopBindsAndSolves(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	path, _ := squareChain(s, 4)
	before := len(s.Constraints())

	offsets, err := s.OffsetConnectedLoop(path, 0.5) // inward
	if err != nil {
		t.Fatalf("OffsetConnectedLoop: %v", err)
	}
	added := s.ConstrainOffsetLoop(path, offsets)
	if added != 8 { // 4 parallel + 4 coincident corners
		t.Fatalf("constrained offset added %d constraints, want 8 (4 parallel + 4 coincident)", added)
	}
	if got := len(s.Constraints()) - before; got != 8 {
		t.Fatalf("sketch grew by %d constraints, want 8", got)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatal("constrained offset does not solve — the added constraints conflict")
	}
	// The inner-left offset edge stays at x=0.5 after solving (the constraints did not move it).
	innerLeft := false
	for _, e := range offsets {
		l, ok := e.(*Line)
		if !ok {
			continue
		}
		if absTol(float64(l.A.Position().X)-0.5) < 1e-6 && absTol(float64(l.B.Position().X)-0.5) < 1e-6 {
			innerLeft = true
		}
	}
	if !innerLeft {
		t.Error("inner-left offset edge is no longer at x=0.5 after constraining/solving")
	}
}

// TestConstrainOffsetSinglePairsLineParallel: a single constrained line offset adds exactly one
// parallel constraint (no corners), and the result solves.
func TestConstrainOffsetSinglePairsLineParallel(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	before := len(s.Constraints())
	off, err := s.OffsetEntity(src, 2)
	if err != nil {
		t.Fatalf("OffsetEntity: %v", err)
	}
	if n := s.ConstrainOffsetSingle(src, off); n != 1 {
		t.Fatalf("single line offset added %d constraints, want 1 (parallel)", n)
	}
	if got := len(s.Constraints()) - before; got != 1 {
		t.Fatalf("sketch grew by %d constraints, want 1", got)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatal("constrained single-line offset does not solve")
	}
}

// TestConstrainOffsetSkipsProjectedSource: a non-analytic projection is a grounded reference spline
// (ADR-0055 phase 3) with no native line/circle binding, so constraining an offset to it pairs
// nothing (the reference geometry is already associative to the model).
func TestConstrainOffsetSkipsProjectedSource(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pc := s.addReferencePolyline([]gmath.Point2{gmath.P2(0, 0), gmath.P2(2, 1), gmath.P2(4, 0)})
	off := s.Lines().AddByTwoPoints(gmath.P2(0, 1), gmath.P2(4, 1))
	if n := s.ConstrainOffsetSingle(pc, off); n != 0 {
		t.Fatalf("projected source added %d constraints, want 0 (skipped)", n)
	}
}

func absTol(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
