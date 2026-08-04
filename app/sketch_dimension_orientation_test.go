// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2025: every distance dimension was aligned to its geometry, so a diagonal line could only
// carry a dimension along itself. The model has measured |Δx| and |Δy| with their own solver
// residuals since #1869; nothing ever asked for them. These drive the real click path.

// diagonal is a 3-4-5 line: aligned 5, ΔX 3, ΔY 4 — three distinguishable values.
var diagStart, diagEnd = math.P2(0, 0), math.P2(3, 4)

// placeDimension picks the line then places its label at `at`, returning the dimension.
func placeDimension(t *testing.T, s *Session, sk *sketch.Sketch, l *sketch.Line, at math.Point2) *sketch.DimensionConstraint {
	t.Helper()
	s.StartTool(NewDimensionTool())
	clickSketch(t, s, midOf(l))
	clickSketch(t, s, at)
	if sk.DimensionConstraints().Count() == 0 {
		t.Fatal("no dimension was created")
	}
	return sk.DimensionConstraints().Item(sk.DimensionConstraints().Count() - 1)
}

// TestDraggingUpwardGivesAHorizontalDimension: pulling the label straight up off a diagonal
// asks for its ΔX — the dimension line is horizontal, so the drag is perpendicular to it.
func TestDraggingUpwardGivesAHorizontalDimension(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(diagStart, diagEnd)
	d := placeDimension(t, s, sk, l, math.P2(1.5, 9)) // straight up from the midpoint

	if got := d.Orientation(); got != sketch.HorizontalDistance {
		t.Fatalf("orientation = %v, want HorizontalDistance", got)
	}
	if got := d.Measured(); stdmath.Abs(got-3) > 1e-6 {
		t.Errorf("measured %v, want 3 (the X separation), not the aligned 5", got)
	}
}

// TestDraggingSidewaysGivesAVerticalDimension: pulling the label sideways asks for ΔY.
func TestDraggingSidewaysGivesAVerticalDimension(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(diagStart, diagEnd)
	d := placeDimension(t, s, sk, l, math.P2(9, 2))

	if got := d.Orientation(); got != sketch.VerticalDistance {
		t.Fatalf("orientation = %v, want VerticalDistance", got)
	}
	if got := d.Measured(); stdmath.Abs(got-4) > 1e-6 {
		t.Errorf("measured %v, want 4 (the Y separation)", got)
	}
}

// TestDraggingPerpendicularKeepsTheAlignedDimension: the historical behaviour must still be
// reachable, and it is what a perpendicular drag means.
func TestDraggingPerpendicularKeepsTheAlignedDimension(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(diagStart, diagEnd)
	// Perpendicular to (3,4) is (-4,3)/5; step off the midpoint (1.5,2) along it.
	at := math.P2(1.5-4*0.6, 2+3*0.6)
	d := placeDimension(t, s, sk, l, at)

	if got := d.Orientation(); got != sketch.AlignedDistance {
		t.Fatalf("orientation = %v, want AlignedDistance", got)
	}
	if got := d.Measured(); stdmath.Abs(got-5) > 1e-6 {
		t.Errorf("measured %v, want 5 (the 3-4-5 length)", got)
	}
}

// TestHorizontalDimensionLeavesTheYSeparationFree is the point of the whole feature: these are
// different CONSTRAINTS, not different renderings. Driving a horizontal dimension must move the
// points in X and leave their Y separation alone.
func TestHorizontalDimensionLeavesTheYSeparationFree(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(diagStart, diagEnd)
	placeDimension(t, s, sk, l, math.P2(1.5, 9)) // horizontal

	if err := s.CommitPendingDimension("60 mm"); err != nil { // 6 cm in X
		t.Fatalf("CommitPendingDimension: %v", err)
	}
	pa, pb := l.A.Position(), l.B.Position()
	if got := stdmath.Abs(pb.X - pa.X); stdmath.Abs(got-6) > 1e-3 {
		t.Errorf("X separation %v, want 6 — the horizontal dimension did not drive ΔX", got)
	}
	if got := stdmath.Abs(pb.Y - pa.Y); stdmath.Abs(got-4) > 1e-3 {
		t.Errorf("Y separation %v, want the original 4 left free by a horizontal dimension", got)
	}
}

// TestHorizontalDimensionLineIsDrawnHorizontal: the glyph must show the value it measures. A
// horizontal dimension drawn along the diagonal would claim a length of 5 while constraining 3.
func TestHorizontalDimensionLineIsDrawnHorizontal(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(diagStart, diagEnd)
	placeDimension(t, s, sk, l, math.P2(1.5, 9))

	views := s.SketchDimensions()
	if len(views) != 1 {
		t.Fatalf("got %d views, want 1", len(views))
	}
	dimLine := views[0].Segments[len(views[0].Segments)-1] // the dimension line is drawn last
	if dy := stdmath.Abs(dimLine[1].Y - dimLine[0].Y); dy > 1e-6 {
		t.Errorf("the dimension line rises %v; a horizontal dimension must be drawn horizontal", dy)
	}
	if span := stdmath.Abs(dimLine[1].X - dimLine[0].X); stdmath.Abs(span-3) > 1e-6 {
		t.Errorf("the dimension line spans %v, want 3 — it must show the ΔX it measures", span)
	}
}

// TestOrientationRuleIsStableForAxisAlignedLines: on a horizontal line the aligned and
// horizontal candidates coincide, and the tie must resolve to aligned rather than flapping.
func TestOrientationRuleIsStableForAxisAlignedLines(t *testing.T) {
	a, b := math.P2(0, 0), math.P2(4, 0)
	if got := orientationForPlacement(a, b, math.P2(2, 3)); got != sketch.AlignedDistance {
		t.Errorf("placing above a horizontal line gave %v, want AlignedDistance", got)
	}
}

// TestPlacementOnTheSegmentFallsBackToAligned: a label dropped on the midpoint has no drag
// direction to read, and must not produce an arbitrary orientation.
func TestPlacementOnTheSegmentFallsBackToAligned(t *testing.T) {
	a, b := diagStart, diagEnd
	if got := orientationForPlacement(a, b, a.Midpoint(b)); got != sketch.AlignedDistance {
		t.Errorf("placing on the midpoint gave %v, want AlignedDistance", got)
	}
}
