// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A whole circle's box is the disc's box, not the seam point its two ends share.
func TestCurveBoxWholeCircle(t *testing.T) {
	c, err := geom.NewCircle(math.P3(1, 0, 5), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	box, ok := geom.CurveBox(c, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined a circle")
	}
	want := math.NewBox(math.P3(-1, -2, 5), math.P3(3, 2, 5))
	if box.Min.DistanceTo(want.Min) > 1e-9 || box.Max.DistanceTo(want.Max) > 1e-9 {
		t.Fatalf("circle box = %v, want %v", box, want)
	}
}

// A quarter arc reaches only its own quadrant: the box is the ends' box, with no full-circle bulge.
func TestCurveBoxQuarterArc(t *testing.T) {
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	box, ok := geom.CurveBox(a, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined an arc")
	}
	want := math.NewBox(math.P3(0, 0, 0), math.P3(2, 2, 0))
	if box.Min.DistanceTo(want.Min) > 1e-9 || box.Max.DistanceTo(want.Max) > 1e-9 {
		t.Fatalf("quarter-arc box = %v, want %v", box, want)
	}
}

// A half arc bulges past both ends: the box carries the interior stationary point.
func TestCurveBoxHalfArcBulge(t *testing.T) {
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, stdmath.Pi)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	box, ok := geom.CurveBox(a, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined an arc")
	}
	if stdmath.Abs(float64(box.Max.Y)-2) > 1e-9 {
		t.Fatalf("half-arc box top = %v, want y=2 (the arc's crown)", box.Max)
	}
}

// A line segment's box is its two ends.
func TestCurveBoxLineSegment(t *testing.T) {
	s := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 2, 3))
	box, ok := geom.CurveBox(s, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined a segment")
	}
	want := math.NewBox(math.P3(0, 0, 0), math.P3(1, 2, 3))
	if box.Min.DistanceTo(want.Min) > 1e-9 || box.Max.DistanceTo(want.Max) > 1e-9 {
		t.Fatalf("segment box = %v, want %v", box, want)
	}
}
