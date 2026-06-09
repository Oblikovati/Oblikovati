// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

func TestOffsetDimMeasuresPerpendicular(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	p := s.Points().Add(gmath.P2(2, 3))
	d, err := s.DimensionConstraints().AddOffsetDim(p, l, "30 mm")
	if err != nil {
		t.Fatalf("AddOffsetDim: %v", err)
	}
	if got := d.Measured(); math.Abs(got-3) > 1e-9 {
		t.Fatalf("measured offset = %v, want 3", got)
	}
}

func TestThreePointAngleMeasuresRightAngle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	v := s.Points().Add(gmath.P2(0, 0))
	a := s.Points().Add(gmath.P2(1, 0))
	b := s.Points().Add(gmath.P2(0, 1))
	d, err := s.DimensionConstraints().AddThreePointAngle(v, a, b, "90 deg")
	if err != nil {
		t.Fatalf("AddThreePointAngle: %v", err)
	}
	if got := d.Measured(); math.Abs(got-math.Pi/2) > 1e-9 {
		t.Fatalf("measured angle = %v, want π/2", got)
	}
}

func TestEllipseRadiusDimDrives(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	e := s.Ellipses().Add(gmath.P2(0, 0), gmath.V2(1, 0), 2, 1)
	d, err := s.DimensionConstraints().AddEllipseRadius(e, "3 cm")
	if err != nil {
		t.Fatalf("AddEllipseRadius: %v", err)
	}
	s.Solve()
	if got := float64(e.MajorRadius); math.Abs(got-3) > 1e-6 {
		t.Fatalf("after solve, ellipse major radius = %v, want 3", got)
	}
	_ = d
}

func TestAdvancedDimsRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	p := s.Points().Add(gmath.P2(2, 3))
	v := s.Points().Add(gmath.P2(5, 5))
	a := s.Points().Add(gmath.P2(6, 5))
	b := s.Points().Add(gmath.P2(5, 6))
	if _, err := s.DimensionConstraints().AddOffsetDim(p, l, "30 mm"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DimensionConstraints().AddThreePointAngle(v, a, b, "90 deg"); err != nil {
		t.Fatal(err)
	}
	out := roundTrip(t, sc)
	if got := out.DimensionConstraints().Count(); got != 2 {
		t.Fatalf("dimensions after round trip = %d, want 2", got)
	}
}
