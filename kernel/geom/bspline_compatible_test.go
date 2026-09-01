// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestMakeCompatibleEqualizesDegreeAndKnots(t *testing.T) {
	t.Parallel()
	// a: quadratic with an interior knot at 0.5; b: cubic with an interior knot at 0.25,
	// over a different parameter range. After MakeCompatible both must share degree and knots.
	a, err := NewBSplineCurveUniformWeights(
		2,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 0), math.P3(2, 0, 0), math.P3(3, 1, 0)},
		[]float64{0, 0, 0, 0.5, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("curve a: %v", err)
	}
	b, err := NewBSplineCurveUniformWeights(
		3,
		[]math.Point3{math.P3(0, 2, 0), math.P3(1, 3, 0), math.P3(2, 2, 0), math.P3(3, 3, 0), math.P3(4, 2, 0)},
		[]float64{2, 2, 2, 2, 3, 4, 4, 4, 4},
	)
	if err != nil {
		t.Fatalf("curve b: %v", err)
	}

	ca, cb, err := MakeCompatible(a, b)
	if err != nil {
		t.Fatalf("MakeCompatible: %v", err)
	}
	if ca.Degree != cb.Degree || ca.Degree != 3 {
		t.Errorf("degrees = %d,%d, want both 3", ca.Degree, cb.Degree)
	}
	if len(ca.Knots) != len(cb.Knots) {
		t.Fatalf("knot counts differ: %d vs %d", len(ca.Knots), len(cb.Knots))
	}
	for i := range ca.Knots {
		if ca.Knots[i] != cb.Knots[i] {
			t.Fatalf("knot %d differs: %g vs %g", i, ca.Knots[i], cb.Knots[i])
		}
	}
	if len(ca.Ctrl) != len(cb.Ctrl) {
		t.Errorf("control counts differ: %d vs %d", len(ca.Ctrl), len(cb.Ctrl))
	}
}

func TestMakeCompatiblePreservesGeometry(t *testing.T) {
	t.Parallel()
	a, err := NewBSplineCurveUniformWeights(
		2,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(3, 0, 0)},
		[]float64{0, 0, 0, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("curve a: %v", err)
	}
	b, err := NewBSplineCurveUniformWeights(
		3,
		[]math.Point3{math.P3(0, 1, 0), math.P3(1, 2, 0), math.P3(2, 0, 0), math.P3(3, 1, 0)},
		[]float64{0, 0, 0, 0, 1, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("curve b: %v", err)
	}
	ca, cb, err := MakeCompatible(a, b)
	if err != nil {
		t.Fatalf("MakeCompatible: %v", err)
	}
	// a and b share the domain [0,1] already, so the compatible curves must trace the
	// same points as the originals (reparametrization is the identity here).
	for i := 0; i <= 20; i++ {
		u := float64(i) / 20
		if !a.PointAt(u).IsEqualTo(ca.PointAt(u), 1e-9) {
			t.Fatalf("curve a diverges at u=%g after MakeCompatible", u)
		}
		if !b.PointAt(u).IsEqualTo(cb.PointAt(u), 1e-9) {
			t.Fatalf("curve b diverges at u=%g after MakeCompatible", u)
		}
	}
}

func TestMakeCompatibleElevatesEitherSide(t *testing.T) {
	t.Parallel()
	// a is the higher-degree curve here, exercising the b<a branch of matchDegree.
	a, err := NewBSplineCurveUniformWeights(
		3,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 0), math.P3(2, 0, 0), math.P3(3, 1, 0)},
		[]float64{0, 0, 0, 0, 1, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("curve a: %v", err)
	}
	b, err := NewBSplineCurveUniformWeights(
		2,
		[]math.Point3{math.P3(0, 2, 0), math.P3(1, 3, 0), math.P3(2, 2, 0)},
		[]float64{0, 0, 0, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("curve b: %v", err)
	}
	ca, cb, err := MakeCompatible(a, b)
	if err != nil {
		t.Fatalf("MakeCompatible: %v", err)
	}
	if ca.Degree != 3 || cb.Degree != 3 {
		t.Errorf("degrees = %d,%d, want both 3", ca.Degree, cb.Degree)
	}
}

func TestExpandKnots(t *testing.T) {
	t.Parallel()
	got := expandKnots([]float64{0.25, 0.5}, []int{1, 2})
	want := []float64{0.25, 0.5, 0.5}
	if len(got) != len(want) {
		t.Fatalf("expandKnots = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expandKnots = %v, want %v", got, want)
		}
	}
}
