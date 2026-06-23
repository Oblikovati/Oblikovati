// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestElevateThenReduceRoundTrip(t *testing.T) {
	c := sampleCubicCurve(t)
	elevated, err := c.ElevateDegree(1)
	if err != nil {
		t.Fatalf("ElevateDegree: %v", err)
	}
	got, ok, err := elevated.ReduceDegree(1e-9)
	if err != nil {
		t.Fatalf("ReduceDegree: %v", err)
	}
	if !ok {
		t.Fatal("an elevated curve must reduce back to its original degree")
	}
	if got.Degree != c.Degree {
		t.Errorf("degree = %d, want %d", got.Degree, c.Degree)
	}
	if len(got.Ctrl) != len(c.Ctrl) {
		t.Errorf("control count = %d, want %d (back to the original)", len(got.Ctrl), len(c.Ctrl))
	}
	curvesAgree(t, c, got, 1e-7)
}

func TestReduceDegreeBezierRoundTrip(t *testing.T) {
	c := quarterCircleNURBS(t) // degree 2 Bézier
	elevated, err := c.ElevateDegree(1)
	if err != nil {
		t.Fatalf("ElevateDegree: %v", err)
	}
	got, ok, err := elevated.ReduceDegree(1e-9)
	if err != nil {
		t.Fatalf("ReduceDegree: %v", err)
	}
	if !ok || got.Degree != 2 || len(got.Ctrl) != 3 {
		t.Fatalf("reduced Bézier: ok=%v degree=%d ctrl=%d; want true/2/3", ok, got.Degree, len(got.Ctrl))
	}
	for i := 0; i <= 20; i++ {
		u := float64(i) / 20
		if !c.PointAt(u).IsEqualTo(got.PointAt(u), 1e-7) {
			t.Fatalf("reduced quarter circle diverges at u=%g", u)
		}
	}
}

func TestReduceDegreeRejectsIrreducible(t *testing.T) {
	// A genuine cubic (its degree-3 term is essential) cannot drop to a quadratic at a
	// tight tolerance.
	c := sampleCubicCurve(t)
	if _, ok, err := c.ReduceDegree(1e-9); err != nil {
		t.Fatalf("ReduceDegree: %v", err)
	} else if ok {
		t.Error("a genuine cubic must not reduce to a quadratic within a tight tolerance")
	}
}

func TestReduceDegreeRejectsDegreeOne(t *testing.T) {
	c, err := NewBSplineCurveUniformWeights(
		1,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)},
		[]float64{0, 0, 1, 1},
	)
	if err != nil {
		t.Fatalf("linear curve: %v", err)
	}
	if _, _, err := c.ReduceDegree(1e-6); err == nil {
		t.Error("reducing a degree-1 curve should error")
	}
}

func TestReduceDegree2dRoundTrip(t *testing.T) {
	c, err := NewBSplineCurve2dUniformWeights(
		2,
		[]math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(3, 0), math.P2(4, 1)},
		[]float64{0, 0, 0, 0.5, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("2d curve: %v", err)
	}
	elevated, err := c.ElevateDegree(1)
	if err != nil {
		t.Fatalf("ElevateDegree: %v", err)
	}
	got, ok, err := elevated.ReduceDegree(1e-9)
	if err != nil || !ok {
		t.Fatalf("ReduceDegree: ok=%v err=%v", ok, err)
	}
	if got.Degree != 2 {
		t.Errorf("degree = %d, want 2", got.Degree)
	}
	lo, hi := c.Domain()
	for i := 0; i <= 20; i++ {
		u := lo + (hi-lo)*float64(i)/20
		if !c.PointAt(u).IsEqualTo(got.PointAt(u), 1e-7) {
			t.Fatalf("2d curve diverges at u=%g after round-trip", u)
		}
	}
}

func TestElevateThenReduceSurfaceRoundTrip(t *testing.T) {
	s := sampleQuadraticSurface(t)
	elevated, err := s.ElevateDegreeU(1)
	if err != nil {
		t.Fatalf("ElevateDegreeU: %v", err)
	}
	gotU, ok, err := elevated.ReduceDegreeU(1e-9)
	if err != nil || !ok {
		t.Fatalf("ReduceDegreeU: ok=%v err=%v", ok, err)
	}
	if gotU.UDegree != s.UDegree {
		t.Errorf("U degree = %d, want %d", gotU.UDegree, s.UDegree)
	}
	elevatedV, err := s.ElevateDegreeV(1)
	if err != nil {
		t.Fatalf("ElevateDegreeV: %v", err)
	}
	gotV, okV, err := elevatedV.ReduceDegreeV(1e-9)
	if err != nil || !okV {
		t.Fatalf("ReduceDegreeV: ok=%v err=%v", okV, err)
	}
	if gotV.VDegree != s.VDegree {
		t.Errorf("V degree = %d, want %d", gotV.VDegree, s.VDegree)
	}
	for i := 0; i <= 10; i++ {
		for j := 0; j <= 10; j++ {
			u, v := float64(i)/10, float64(j)/10
			if !s.PointAt(u, v).IsEqualTo(gotV.PointAt(u, v), 1e-7) {
				t.Fatalf("surface diverges at (%g,%g) after V round-trip", u, v)
			}
		}
	}
}
