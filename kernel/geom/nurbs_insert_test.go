// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// sampleCubicCurve is a non-rational cubic B-spline with one interior knot, a
// general-position control polygon the refinement tests subdivide.
func sampleCubicCurve(t *testing.T) BSplineCurve {
	t.Helper()
	c, err := NewBSplineCurveUniformWeights(
		3,
		[]math.Point3{
			math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(3, 2, 1),
			math.P3(4, 0, 1), math.P3(6, 1, 0),
		},
		[]float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("sample cubic curve: %v", err)
	}
	return c
}

// sampleQuadraticSurface is a 2×2-interior NURBS surface (degree 2×2) with varied
// weights, for surface-direction refinement round-trips.
func sampleQuadraticSurface(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 1), math.P3(0, 2, 0)},
		{math.P3(1, 0, 1), math.P3(1, 1, 2), math.P3(1, 2, 1)},
		{math.P3(2, 0, 0), math.P3(2, 1, 1), math.P3(2, 2, 0)},
	}
	weights := [][]float64{{1, 2, 1}, {2, 4, 2}, {1, 2, 1}}
	s, err := NewBSplineSurface(2, 2, ctrl, weights,
		[]float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("sample quadratic surface: %v", err)
	}
	return s
}

// curvesAgree samples both curves over the (shared) domain and fails if any sample
// deviates beyond tol — the geometric-identity check refinement must satisfy.
func curvesAgree(t *testing.T, a, b BSplineCurve, tol float64) {
	t.Helper()
	lo, hi := a.Domain()
	for i := 0; i <= 20; i++ {
		u := lo + (hi-lo)*float64(i)/20
		if !a.PointAt(u).IsEqualTo(b.PointAt(u), tol) {
			t.Fatalf("curves diverge at u=%g: %v vs %v", u, a.PointAt(u), b.PointAt(u))
		}
	}
}

func TestInsertKnotPreservesCurve(t *testing.T) {
	c := sampleCubicCurve(t)
	got, err := c.InsertKnot(0.25, 1)
	if err != nil {
		t.Fatalf("InsertKnot: %v", err)
	}
	if len(got.Ctrl) != len(c.Ctrl)+1 {
		t.Errorf("control count = %d, want %d", len(got.Ctrl), len(c.Ctrl)+1)
	}
	if knotMultiplicity(got.Knots, 0.25) != 1 {
		t.Errorf("inserted knot multiplicity = %d, want 1", knotMultiplicity(got.Knots, 0.25))
	}
	curvesAgree(t, c, got, 1e-12)
}

func TestInsertKnotRepeatedRaisesMultiplicity(t *testing.T) {
	c := sampleCubicCurve(t)
	got, err := c.InsertKnot(0.25, 3) // degree 3, fresh knot → up to 3 allowed
	if err != nil {
		t.Fatalf("InsertKnot x3: %v", err)
	}
	if m := knotMultiplicity(got.Knots, 0.25); m != 3 {
		t.Errorf("multiplicity = %d, want 3", m)
	}
	curvesAgree(t, c, got, 1e-12)
}

func TestInsertKnotRejectsOverflow(t *testing.T) {
	c := sampleCubicCurve(t)
	if _, err := c.InsertKnot(0.5, 3); err == nil {
		t.Error("inserting an interior knot (already multiplicity 1) 3 times at degree 3 should error")
	}
	if _, err := c.InsertKnot(0.0, 1); err == nil {
		t.Error("inserting the domain-boundary knot should error (outside the open domain)")
	}
	if _, err := c.InsertKnot(0.5, 0); err == nil {
		t.Error("a zero insertion count should error")
	}
}

func TestRefineKnotsPreservesCurve(t *testing.T) {
	c := sampleCubicCurve(t)
	got, err := c.RefineKnots([]float64{0.25, 0.25, 0.75})
	if err != nil {
		t.Fatalf("RefineKnots: %v", err)
	}
	if len(got.Ctrl) != len(c.Ctrl)+3 {
		t.Errorf("control count = %d, want %d", len(got.Ctrl), len(c.Ctrl)+3)
	}
	curvesAgree(t, c, got, 1e-12)
}

func TestInsertKnot2dPreservesCurve(t *testing.T) {
	c, err := NewBSplineCurve2dUniformWeights(
		2,
		[]math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(3, 0), math.P2(4, 1)},
		[]float64{0, 0, 0, 0.5, 1, 1, 1},
	)
	if err != nil {
		t.Fatalf("2d curve: %v", err)
	}
	got, err := c.InsertKnot(0.25, 1)
	if err != nil {
		t.Fatalf("InsertKnot: %v", err)
	}
	lo, hi := c.Domain()
	for i := 0; i <= 20; i++ {
		u := lo + (hi-lo)*float64(i)/20
		if !c.PointAt(u).IsEqualTo(got.PointAt(u), 1e-12) {
			t.Fatalf("2d curves diverge at u=%g: %v vs %v", u, c.PointAt(u), got.PointAt(u))
		}
	}
}

func TestInsertKnotSurfacePreservesGeometry(t *testing.T) {
	s := sampleQuadraticSurface(t)
	gu, err := s.InsertKnotU(0.5, 1)
	if err != nil {
		t.Fatalf("InsertKnotU: %v", err)
	}
	gv, err := gu.InsertKnotV(0.5, 1)
	if err != nil {
		t.Fatalf("InsertKnotV: %v", err)
	}
	if len(gv.Ctrl) != len(s.Ctrl)+1 || len(gv.Ctrl[0]) != len(s.Ctrl[0])+1 {
		t.Errorf("net dims = %dx%d, want %dx%d", len(gv.Ctrl), len(gv.Ctrl[0]), len(s.Ctrl)+1, len(s.Ctrl[0])+1)
	}
	for i := 0; i <= 10; i++ {
		for j := 0; j <= 10; j++ {
			u, v := float64(i)/10, float64(j)/10
			if !s.PointAt(u, v).IsEqualTo(gv.PointAt(u, v), 1e-12) {
				t.Fatalf("surface diverges at (%g,%g): %v vs %v", u, v, s.PointAt(u, v), gv.PointAt(u, v))
			}
		}
	}
}
