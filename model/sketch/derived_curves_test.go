// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestEquationCurveSamplesUnitCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// x=cos t, y=sin t over [0, 2π] traces the unit circle.
	e, err := s.EquationCurves().Add("cos(t)", "sin(t)", 0, 2*math.Pi)
	if err != nil {
		t.Fatalf("Add equation curve: %v", err)
	}
	for _, tv := range []float64{0, math.Pi / 2, math.Pi} {
		p := e.At(tv)
		if r := math.Hypot(float64(p.X), float64(p.Y)); math.Abs(r-1) > 1e-9 {
			t.Errorf("|curve(%v)| = %v, want 1", tv, r)
		}
	}
	// (0): (1,0).
	if p := e.At(0); math.Abs(float64(p.X)-1) > 1e-9 || math.Abs(float64(p.Y)) > 1e-9 {
		t.Errorf("curve(0) = %v, want (1,0)", p)
	}
}

func TestEquationCurveRejectsUnknownVariable(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if _, err := s.EquationCurves().Add("cos(u)", "sin(t)", 0, 1); err == nil {
		t.Error("equation curve with unknown variable u should error")
	}
	if _, err := s.EquationCurves().Add("t", "t", 1, 1); err == nil {
		t.Error("empty t-range should error")
	}
}

func TestOffsetSplineSamplesParallel(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// A straight-ish fit spline along +X; offset by 1 should sit ~1 above (left normal +Y).
	parent := s.Splines().AddByPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(2, 0), gmath.P2(4, 0)}, false)
	off := s.OffsetSplines().Add(parent, 1)
	for _, p := range off.Sample() {
		if math.Abs(float64(p.Y)-1) > 1e-9 {
			t.Fatalf("offset point Y = %v, want 1", p.Y)
		}
	}
}

func TestDerivedCurvesRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	if _, err := s.EquationCurves().Add("cos(t)", "sin(t)", 0, 6.28); err != nil {
		t.Fatal(err)
	}
	s.FixedSplines().Add([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)})
	parent := s.Splines().AddByPoints([]gmath.Point2{gmath.P2(5, 5), gmath.P2(7, 5)}, false)
	s.OffsetSplines().Add(parent, 0.5)

	out := roundTrip(t, sc)
	if out.EquationCurves().Count() != 1 || out.FixedSplines().Count() != 1 || out.OffsetSplines().Count() != 1 {
		t.Fatalf("after round trip: eq=%d fixed=%d offset=%d, want 1/1/1",
			out.EquationCurves().Count(), out.FixedSplines().Count(), out.OffsetSplines().Count())
	}
}
