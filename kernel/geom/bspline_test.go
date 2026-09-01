// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// quarterCircleNURBS is the textbook rational quadratic that traces a quarter of
// the unit circle from (1,0) to (0,1) — the standard NURBS-circle building block.
func quarterCircleNURBS(t *testing.T) BSplineCurve {
	t.Helper()
	w := stdmath.Sqrt2 / 2
	c, err := NewBSplineCurve(
		2,
		[]math.Point3{math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)},
		[]float64{1, w, 1},
		[]float64{0, 0, 0, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("quarter-circle NURBS: %v", err)
	}
	return c
}

// sampleBSplineSurface is a bilinear (degree 1×1) patch for which P(u,v) =
// (u, v, u+v) — a closed form the tests can check against.
func sampleBSplineSurface(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 1)},
		{math.P3(1, 0, 1), math.P3(1, 1, 2)},
	}
	weights := [][]float64{{1, 1}, {1, 1}}
	s, err := NewBSplineSurface(1, 1, ctrl, weights, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("sample bspline surface: %v", err)
	}
	return s
}

func TestNURBSQuarterCircleStaysOnUnitCircle(t *testing.T) {
	t.Parallel()
	c := quarterCircleNURBS(t)
	if !c.PointAt(0).IsEqualTo(math.P3(1, 0, 0), 1e-12) {
		t.Errorf("start = %v, want {1 0 0}", c.PointAt(0))
	}
	if !c.PointAt(1).IsEqualTo(math.P3(0, 1, 0), 1e-12) {
		t.Errorf("end = %v, want {0 1 0}", c.PointAt(1))
	}
	for i := 0; i <= 10; i++ {
		p := c.PointAt(float64(i) / 10)
		approxScalar(t, p.AsVector().Length(), 1, "radius on unit circle")
	}
}

func TestNURBSDegree1ReproducesSegment(t *testing.T) {
	t.Parallel()
	c, err := NewBSplineCurveUniformWeights(
		1,
		[]math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0)},
		[]float64{0, 0, 1, 1},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.PointAt(0.5).IsEqualTo(math.P3(1, 0, 0), 1e-12) {
		t.Errorf("midpoint = %v, want {1 0 0}", c.PointAt(0.5))
	}
	lo, hi := c.Domain()
	if lo != 0 || hi != 1 {
		t.Errorf("Domain = [%v,%v], want [0,1]", lo, hi)
	}
}

func TestNURBSCurveConstructorErrors(t *testing.T) {
	t.Parallel()
	ctrl := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}
	if _, err := NewBSplineCurve(1, ctrl, []float64{1, 1}, []float64{0, 0, 1}); err == nil {
		t.Error("wrong knot count should error")
	}
	if _, err := NewBSplineCurve(1, ctrl, []float64{1}, []float64{0, 0, 1, 1}); err == nil {
		t.Error("weight/control mismatch should error")
	}
	if _, err := NewBSplineCurve(1, ctrl, []float64{1, -1}, []float64{0, 0, 1, 1}); err == nil {
		t.Error("non-positive weight should error")
	}
}

func TestNURBSSurfaceBilinearClosedForm(t *testing.T) {
	t.Parallel()
	s := sampleBSplineSurface(t)
	for _, uv := range [][2]float64{{0, 0}, {1, 1}, {0.5, 0.5}, {0.25, 0.8}} {
		u, v := uv[0], uv[1]
		want := math.P3(u, v, u+v)
		if got := s.PointAt(u, v); !got.IsEqualTo(want, 1e-12) {
			t.Errorf("PointAt(%v,%v) = %v, want %v", u, v, got, want)
		}
	}
}

func TestNURBSSurfaceConstructorErrors(t *testing.T) {
	t.Parallel()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 0)},
	}
	w := [][]float64{{1, 1}, {1, 1}}
	if _, err := NewBSplineSurface(1, 1, ctrl, w, []float64{0, 0, 1}, []float64{0, 0, 1, 1}); err == nil {
		t.Error("wrong u-knot count should error")
	}
	ragged := [][]math.Point3{{math.P3(0, 0, 0)}, {math.P3(1, 0, 0), math.P3(1, 1, 0)}}
	if _, err := NewBSplineSurface(1, 1, ragged, w, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1}); err == nil {
		t.Error("non-rectangular control net should error")
	}
}
