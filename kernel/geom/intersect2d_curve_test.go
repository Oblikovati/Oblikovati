// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// ellipse42 is the axis-aligned test ellipse centered at the origin with
// radii 4 (x) and 2 (y).
func ellipse42(t *testing.T) EllipseFull2d {
	t.Helper()
	e, err := NewEllipseFull2d(math.P2(0, 0), math.V2(1, 0), 4, 2)
	if err != nil {
		t.Fatalf("NewEllipseFull2d: %v", err)
	}
	return e
}

// TestLineCurve2dIntersectionEllipse: a horizontal line through the center of
// a 4×2 ellipse must cross at exactly (±4, 0).
func TestLineCurve2dIntersectionEllipse(t *testing.T) {
	l, err := NewLine2d(math.P2(-10, 0), math.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	hits := LineCurve2dIntersection(l, ellipse42(t))
	if len(hits) != 2 {
		t.Fatalf("crossings = %d (%v), want 2", len(hits), hits)
	}
	for _, p := range hits {
		if stdmath.Abs(stdmath.Abs(float64(p.X))-4) > 1e-9 || stdmath.Abs(float64(p.Y)) > 1e-9 {
			t.Errorf("crossing %v, want (±4, 0) within 1e-9", p)
		}
	}
}

// TestSegmentCurve2dIntersectionBoundsToSegment: the same support line cut to
// a segment that ends inside the ellipse must keep only the crossing it spans.
func TestSegmentCurve2dIntersectionBoundsToSegment(t *testing.T) {
	seg := NewLineSegment2d(math.P2(-10, 0), math.P2(0, 0))
	hits := SegmentCurve2dIntersection(seg, ellipse42(t))
	if len(hits) != 1 {
		t.Fatalf("crossings = %d (%v), want only the x=-4 one", len(hits), hits)
	}
	if stdmath.Abs(float64(hits[0].X)+4) > 1e-9 {
		t.Errorf("crossing = %v, want (-4, 0)", hits[0])
	}
}

// TestCircleCurve2dIntersectionEllipse: a circle of radius 2 about the origin
// must touch the 4×2 ellipse exactly at (0, ±2) — the minor vertices, where
// |p| − 2 crosses zero.
func TestCircleCurve2dIntersectionEllipse(t *testing.T) {
	hits := CircleCurve2dIntersection(NewCircle2d(math.P2(0, 0), 3), ellipse42(t))
	if len(hits) != 4 {
		t.Fatalf("crossings = %d (%v), want 4 (one per quadrant)", len(hits), hits)
	}
	for _, p := range hits {
		onCircle := stdmath.Abs(float64(p.DistanceTo(math.P2(0, 0)))-3) < 1e-9
		x, y := float64(p.X), float64(p.Y)
		onEllipse := stdmath.Abs(x*x/16+y*y/4-1) < 1e-9
		if !onCircle || !onEllipse {
			t.Errorf("crossing %v not on both curves (circle %v, ellipse %v)", p, onCircle, onEllipse)
		}
	}
}

// TestLineCurve2dTangentialContact is the #1608 even-multiplicity regression: the quadratic
// Bézier with y(t) = (3t−2)² is tangent to the x-axis at t = 2/3 — the field touches zero
// WITHOUT changing sign, so sample-and-bracket seeding can never see it (and t = 2/3 is not
// a dyadic sample, so no sample lands on the zero by luck). The certified root isolation
// must report exactly the one contact, at (11/9, 0).
func TestLineCurve2dTangentialContact(t *testing.T) {
	curve, err := NewBSplineCurve2dUniformWeights(2,
		[]math.Point2{math.P2(-1, 4), math.P2(1, -2), math.P2(2, 1)},
		[]float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineCurve2dUniformWeights: %v", err)
	}
	axis, err := NewLine2d(math.P2(0, 0), math.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	hits := LineCurve2dIntersection(axis, curve)
	if len(hits) != 1 {
		t.Fatalf("tangential contacts = %d (%v), want exactly 1 at (11/9, 0)", len(hits), hits)
	}
	want := math.P2(11.0/9.0, 0)
	if hits[0].DistanceTo(want) > 1e-5 {
		t.Errorf("contact = %v, want %v within 1e-5", hits[0], want)
	}
}

// TestLineCurve2dIntersectionBSpline: a fitted spline through a known zigzag
// crosses the x-axis once per sign change of its fit points, on the curve.
func TestLineCurve2dIntersectionBSpline(t *testing.T) {
	curve, _, err := NewFittedBSplineCurve2dParam([]math.Point2{
		math.P2(0, -1), math.P2(1, 1), math.P2(2, -1), math.P2(3, 1),
	}, FitCentripetal)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	axis, err := NewLine2d(math.P2(0, 0), math.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	hits := LineCurve2dIntersection(axis, curve)
	if len(hits) != 3 {
		t.Fatalf("crossings = %d (%v), want 3", len(hits), hits)
	}
	for _, p := range hits {
		if stdmath.Abs(float64(p.Y)) > 1e-9 {
			t.Errorf("crossing %v not on the axis within 1e-9", p)
		}
	}
}
