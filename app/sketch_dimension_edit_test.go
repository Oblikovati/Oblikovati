// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// placeDistanceDim drives the Dimension tool over two points and returns the sketch.
func placeDistanceDim(t *testing.T, s *Session, sk *sketch.Sketch, a, b math.Point2) {
	t.Helper()
	s.StartTool(NewDimensionTool())
	s.feedPick(SketchEntityHandle{Entity: sk.Points().Add(a)})
	s.feedPick(SketchEntityHandle{Entity: sk.Points().Add(b)})
	// The dimension tool finishes on a placement click or on OK/Enter (#2022); these tests
	// exercise the value-entry that follows, so take the OK path at the default position.
	if err := s.OK(); err != nil {
		t.Fatalf("OK after two point picks: %v", err)
	}
}

func TestPlacingDimensionMakesItPending(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(3, 0)) // 3 cm apart
	s.StartTool(NewDimensionTool())
	s.feedPick(SketchEntityHandle{Entity: a})
	s.feedPick(SketchEntityHandle{Entity: b})
	if err := s.OK(); err != nil {
		t.Fatalf("OK after two point picks: %v", err)
	}
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

// TestCommitBareNumberUsesDocumentUnit is the regression for the "×10 dimension" bug: a bare
// number typed into the value box means the document's display unit (7 → 7 mm), not the raw
// database unit (which would drive 7 cm = 70 mm). The default document is millimetres (#1783).
func TestCommitBareNumberUsesDocumentUnit(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(3, 0)) // 3 cm apart
	s.StartTool(NewDimensionTool())
	s.feedPick(SketchEntityHandle{Entity: a})
	s.feedPick(SketchEntityHandle{Entity: b})
	if err := s.OK(); err != nil {
		t.Fatalf("OK after two point picks: %v", err)
	}
	if err := s.CommitPendingDimension("7"); err != nil { // bare "7" → 7 mm, not 7 cm
		t.Fatalf("CommitPendingDimension(\"7\"): %v", err)
	}
	// 7 mm = 0.7 cm: the solver drives the points to 0.7 db-units apart, NOT 7.
	if d := a.Position().DistanceTo(b.Position()); d < 0.699 || d > 0.701 {
		t.Errorf("bare \"7\" drove the points %.4f cm apart, want ~0.7 (7 mm); ~7 means the ×10 bug", d)
	}
}

func TestCommitInvalidExpressionKeepsPending(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	s, sk := sketchSession(t)
	placeDistanceDim(t, s, sk, math.P2(0, 0), math.P2(3, 0))
	d := sk.DimensionConstraints().Item(0)
	s.CancelPendingDimension()
	s.BeginEditDimension(d)
	if s.PendingDimension() != d {
		t.Error("BeginEditDimension should re-open the given dimension for editing")
	}
}
