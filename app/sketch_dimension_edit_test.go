// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/math"
	"oblikovati/model/sketch"
)

// placeDistanceDim drives the Dimension tool over two points and returns the sketch.
func placeDistanceDim(t *testing.T, s *Session, sk *sketch.Sketch, a, b math.Point2) {
	t.Helper()
	s.StartTool(newDimensionTool())
	s.feedPick(SketchEntityHandle{Entity: sk.Points().Add(a)})
	s.feedPick(SketchEntityHandle{Entity: sk.Points().Add(b)})
}

func TestPlacingDimensionMakesItPending(t *testing.T) {
	s, sk := sketchSession(t)
	placeDistanceDim(t, s, sk, math.P2(0, 0), math.P2(3, 0))
	if s.PendingDimension() == nil {
		t.Fatal("placing a dimension should leave it pending for value entry")
	}
	if got := s.PendingDimensionExpression(); got == "" {
		t.Errorf("pending expression = %q, want the measured value pre-filled", got)
	}
}

func TestCommitPendingDimensionDrivesGeometry(t *testing.T) {
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(3, 0)) // 3 cm apart
	s.StartTool(newDimensionTool())
	s.feedPick(SketchEntityHandle{Entity: a})
	s.feedPick(SketchEntityHandle{Entity: b})
	if err := s.CommitPendingDimension("50 mm"); err != nil {
		t.Fatalf("CommitPendingDimension: %v", err)
	}
	if s.PendingDimension() != nil {
		t.Error("committing should clear the pending dimension")
	}
	// 50 mm = 5 cm: the solver should drive the points to 5 db-units (cm) apart.
	if d := a.Position().DistanceTo(b.Position()); d < 4.99 || d > 5.01 {
		t.Errorf("after committing 50 mm the points are %v cm apart, want ~5", d)
	}
}

func TestCommitInvalidExpressionKeepsPending(t *testing.T) {
	s, sk := sketchSession(t)
	placeDistanceDim(t, s, sk, math.P2(0, 0), math.P2(3, 0))
	if err := s.CommitPendingDimension("not a number ++"); err == nil {
		t.Fatal("an invalid expression should error")
	}
	if s.PendingDimension() == nil {
		t.Error("an invalid expression should keep the dimension pending so the box stays open")
	}
}

func TestCancelPendingDimensionKeepsMeasuredValue(t *testing.T) {
	s, sk := sketchSession(t)
	placeDistanceDim(t, s, sk, math.P2(0, 0), math.P2(3, 0))
	s.CancelPendingDimension()
	if s.PendingDimension() != nil {
		t.Error("cancel should dismiss the edit box")
	}
	if sk.DimensionConstraints().Count() != 1 {
		t.Error("cancel should keep the placed dimension at its measured value")
	}
}

func TestFinishSketchClearsPendingDimension(t *testing.T) {
	s, sk := sketchSession(t)
	placeDistanceDim(t, s, sk, math.P2(0, 0), math.P2(3, 0))
	if err := s.FinishSketch(); err != nil {
		t.Fatalf("FinishSketch: %v", err)
	}
	if s.PendingDimension() != nil {
		t.Error("leaving the sketch should clear any pending dimension edit")
	}
}

func TestBeginEditDimensionReopens(t *testing.T) {
	s, sk := sketchSession(t)
	placeDistanceDim(t, s, sk, math.P2(0, 0), math.P2(3, 0))
	d := sk.DimensionConstraints().Item(0)
	s.CancelPendingDimension()
	s.BeginEditDimension(d)
	if s.PendingDimension() != d {
		t.Error("BeginEditDimension should re-open the given dimension for editing")
	}
}
