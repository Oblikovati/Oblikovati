// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestElevateDegreePreservesCurve(t *testing.T) {
	c := sampleCubicCurve(t)
	got, err := c.ElevateDegree(1)
	if err != nil {
		t.Fatalf("ElevateDegree: %v", err)
	}
	if got.Degree != c.Degree+1 {
		t.Errorf("degree = %d, want %d", got.Degree, c.Degree+1)
	}
	if len(got.Knots) != len(got.Ctrl)+got.Degree+1 {
		t.Errorf("knot/ctrl mismatch: %d knots, %d ctrl, degree %d", len(got.Knots), len(got.Ctrl), got.Degree)
	}
	curvesAgree(t, c, got, 1e-12)
}

func TestElevateDegreeByTwo(t *testing.T) {
	c := sampleCubicCurve(t)
	got, err := c.ElevateDegree(2)
	if err != nil {
		t.Fatalf("ElevateDegree x2: %v", err)
	}
	if got.Degree != c.Degree+2 {
		t.Errorf("degree = %d, want %d", got.Degree, c.Degree+2)
	}
	curvesAgree(t, c, got, 1e-12)
}

func TestElevateDegreeBezier(t *testing.T) {
	// A single-span (Bézier) quadratic: elevating to cubic must keep it on the curve.
	c := quarterCircleNURBS(t)
	got, err := c.ElevateDegree(1)
	if err != nil {
		t.Fatalf("ElevateDegree: %v", err)
	}
	if got.Degree != 3 || len(got.Ctrl) != 4 {
		t.Errorf("elevated Bézier: degree %d, %d ctrl; want 3 and 4", got.Degree, len(got.Ctrl))
	}
	for i := 0; i <= 20; i++ {
		u := float64(i) / 20
		if !c.PointAt(u).IsEqualTo(got.PointAt(u), 1e-12) {
			t.Fatalf("elevated quarter circle diverges at u=%g", u)
		}
	}
}

func TestElevateDegreeRejectsZero(t *testing.T) {
	c := sampleCubicCurve(t)
	if _, err := c.ElevateDegree(0); err == nil {
		t.Error("elevating by 0 should error")
	}
}

func TestElevateDegree2d(t *testing.T) {
	c, err := NewBSplineCurve2dUniformWeights(
		2,
		[]math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(3, 0), math.P2(4, 1)},
		[]float64{0, 0, 0, 0.5, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("2d curve: %v", err)
	}
	got, err := c.ElevateDegree(1)
	if err != nil {
		t.Fatalf("ElevateDegree: %v", err)
	}
	lo, hi := c.Domain()
	for i := 0; i <= 20; i++ {
		u := lo + (hi-lo)*float64(i)/20
		if !c.PointAt(u).IsEqualTo(got.PointAt(u), 1e-12) {
			t.Fatalf("2d curve diverges at u=%g after elevation", u)
		}
	}
}

func TestElevateDegreeSurface(t *testing.T) {
	s := sampleQuadraticSurface(t)
	gu, err := s.ElevateDegreeU(1)
	if err != nil {
		t.Fatalf("ElevateDegreeU: %v", err)
	}
	gv, err := gu.ElevateDegreeV(1)
	if err != nil {
		t.Fatalf("ElevateDegreeV: %v", err)
	}
	if gv.UDegree != 3 || gv.VDegree != 3 {
		t.Errorf("degrees = %dx%d, want 3x3", gv.UDegree, gv.VDegree)
	}
	for i := 0; i <= 10; i++ {
		for j := 0; j <= 10; j++ {
			u, v := float64(i)/10, float64(j)/10
			if !s.PointAt(u, v).IsEqualTo(gv.PointAt(u, v), 1e-12) {
				t.Fatalf("surface diverges at (%g,%g) after elevation: %v vs %v", u, v, s.PointAt(u, v), gv.PointAt(u, v))
			}
		}
	}
}

func TestBinomial(t *testing.T) {
	cases := []struct {
		n, k int
		want float64
	}{{5, 0, 1}, {5, 5, 1}, {5, 2, 10}, {6, 3, 20}, {4, 5, 0}, {4, -1, 0}}
	for _, c := range cases {
		if got := binomial(c.n, c.k); got != c.want {
			t.Errorf("binomial(%d,%d) = %g, want %g", c.n, c.k, got, c.want)
		}
	}
}
