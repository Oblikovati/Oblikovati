// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestScalarHelpers(t *testing.T) {
	if clamp01(-1) != 0 || clamp01(2) != 1 || clamp01(0.5) != 0.5 {
		t.Error("clamp01")
	}
	if clampUnit(2) != 1 || clampUnit(-2) != -1 || clampUnit(0.25) != 0.25 {
		t.Error("clampUnit")
	}
	if clampTo(-1, 0, 1) != 0 || clampTo(2, 0, 1) != 1 || clampTo(0.5, 0, 1) != 0.5 {
		t.Error("clampTo")
	}
}

func TestAngleWrapHelpers(t *testing.T) {
	if got := wrapPositive(0); stdmath.Abs(got-twoPi) > 1e-12 {
		t.Errorf("wrapPositive(0) = %v, want 2π", got)
	}
	if got := wrapPositive(1); stdmath.Abs(got-1) > 1e-12 {
		t.Errorf("wrapPositive(1) = %v, want 1", got)
	}
	if got := wrap2pi(-1); got < 0 || got >= twoPi {
		t.Errorf("wrap2pi(-1) = %v, want [0,2π)", got)
	}
	if got := wrap2pi(1); stdmath.Abs(got-1) > 1e-12 {
		t.Errorf("wrap2pi(1) = %v, want 1", got)
	}
}

func TestUnitOrZeroAndPerpendicular(t *testing.T) {
	if v := unitOrZero(math.V3(0, 0, 0)); v.Length() != 0 {
		t.Errorf("unitOrZero(0) = %v, want zero", v)
	}
	if v := unitOrZero(math.V3(0, 0, 2)); stdmath.Abs(v.Length()-1) > 1e-12 {
		t.Errorf("unitOrZero(non-zero) not unit: %v", v)
	}
	// perpendicularUnit must be ⟂ its input for both the X-dominant and general seeds.
	for _, u := range []math.UnitVector3{mustUnit(t, 1, 0, 0), mustUnit(t, 0, 0, 1)} {
		p := perpendicularUnit(u)
		if stdmath.Abs(p.AsVector().Dot(u.AsVector())) > 1e-9 {
			t.Errorf("perpendicularUnit(%v) not perpendicular", u)
		}
	}
}

func TestConstructorsRejectZeroDirection(t *testing.T) {
	zero := math.V3(0, 0, 0)
	o := math.P3(0, 0, 0)
	if _, err := NewCircle(o, zero, 1); err == nil {
		t.Error("NewCircle with zero normal should error")
	}
	if _, err := NewCylinder(o, zero, 1); err == nil {
		t.Error("NewCylinder with zero axis should error")
	}
	if _, err := NewCone(o, zero, stdmath.Pi/4); err == nil {
		t.Error("NewCone with zero axis should error")
	}
	if _, err := NewTorus(o, zero, 2, 1); err == nil {
		t.Error("NewTorus with zero axis should error")
	}
	if _, err := NewEllipseFull(o, zero, math.V3(1, 0, 0), 2, 1); err == nil {
		t.Error("NewEllipseFull with zero normal should error")
	}
	if _, err := NewArc3d(o, zero, math.V3(1, 0, 0), 1, 0, 1); err == nil {
		t.Error("NewArc3d with zero normal should error")
	}
	if _, err := NewLine2d(math.P2(0, 0), math.V2(0, 0)); err == nil {
		t.Error("NewLine2d with zero direction should error")
	}
	if _, err := NewPlaneFromAxes(o, zero, math.V3(0, 1, 0)); err == nil {
		t.Error("NewPlaneFromAxes with zero uAxis should error")
	}
}

func TestConstructorRangeErrors(t *testing.T) {
	axis := math.V3(0, 0, 1)
	o := math.P3(0, 0, 0)
	if _, err := NewCone(o, axis, -1); err == nil {
		t.Error("NewCone with a non-positive half angle should error")
	}
	if _, err := NewCone(o, axis, stdmath.Pi); err == nil {
		t.Error("NewCone with a half angle ≥ π/2 should error")
	}
	if _, err := NewTorus(o, axis, -1, 1); err == nil {
		t.Error("NewTorus with a non-positive major radius should error")
	}
	// uAxis parallel to vAxis ⇒ the orthogonalized v collapses ⇒ error.
	if _, err := NewPlaneFromAxes(o, math.V3(1, 0, 0), math.V3(2, 0, 0)); err == nil {
		t.Error("NewPlaneFromAxes with parallel axes should error")
	}
}

func TestMoreConstructorErrors(t *testing.T) {
	o := math.P3(0, 0, 0)
	if _, err := NewEllipticalArc(o, math.V3(0, 0, 0), math.V3(1, 0, 0), 2, 1, 0, 1); err == nil {
		t.Error("NewEllipticalArc with zero normal should error")
	}
	if _, err := NewEllipticalArc2d(math.P2(0, 0), math.V2(0, 0), 2, 1, 0, 1); err == nil {
		t.Error("NewEllipticalArc2d with zero major axis should error")
	}
	if _, err := NewPolyline2d([]math.Point2{math.P2(0, 0)}); err == nil {
		t.Error("NewPolyline2d with one vertex should error")
	}
	if _, err := Arc2dByThreePoints(math.P2(0, 0), math.P2(1, 0), math.P2(2, 0)); err == nil {
		t.Error("Arc2dByThreePoints on collinear points should error")
	}
}

func TestClosestPointOnDegenerateSegment(t *testing.T) {
	s := LineSegment{StartPoint: math.P3(2, 2, 2), EndPoint: math.P3(2, 2, 2)}
	if got := ClosestPointOnSegment(s, math.P3(5, 5, 5)); got != s.StartPoint {
		t.Errorf("closest on a zero-length segment = %v, want the point itself", got)
	}
}

func TestIntersectParamHelpers(t *testing.T) {
	if paramTol(0) != 1e-9 || paramTol(0.1) != 0.1 {
		t.Error("paramTol")
	}
	if clampUnitParam(-0.5) != 0 || clampUnitParam(1.5) != 1 || clampUnitParam(0.5) != 0.5 {
		t.Error("clampUnitParam")
	}
}

func TestArcSweepWindings(t *testing.T) {
	c := math.P2(0, 0)
	// CCW: start (1,0) → on (0,1) → end (-1,0) gives a positive (≈π) sweep.
	if s := arcSweep(math.P2(1, 0), math.P2(0, 1), math.P2(-1, 0), c); s <= 0 {
		t.Errorf("CCW arcSweep = %v, want positive", s)
	}
	// CW: start (1,0) → on (0,-1) → end (-1,0) gives a negative sweep.
	if s := arcSweep(math.P2(1, 0), math.P2(0, -1), math.P2(-1, 0), c); s >= 0 {
		t.Errorf("CW arcSweep = %v, want negative", s)
	}
}

func TestValidateBSplineErrors(t *testing.T) {
	// degree < 1
	if validateBSpline(0, 4, 4, 6) == nil {
		t.Error("degree 0 should be invalid")
	}
	// ctrlCount < degree+1
	if validateBSpline(3, 2, 2, 6) == nil {
		t.Error("too few control points should be invalid")
	}
	// weightCount != ctrlCount
	if validateBSpline(2, 4, 3, 7) == nil {
		t.Error("mismatched weight count should be invalid")
	}
	// A valid spline passes (degree 2, 4 ctrl, 4 weights, knots = ctrl+degree+1 = 7).
	if err := validateBSpline(2, 4, 4, 7); err != nil {
		t.Errorf("a valid B-spline rejected: %v", err)
	}
}

func TestPlanarRefFallsBackWhenParallel(t *testing.T) {
	// A major axis parallel to the normal projects to zero ⇒ planarRef falls back to a
	// perpendicular reference, so the ellipse still constructs.
	e, err := NewEllipseFull(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 2), 2, 1)
	if err != nil {
		t.Fatalf("NewEllipseFull with parallel major axis: %v", err)
	}
	if stdmath.Abs(e.MajorAxis.AsVector().Dot(e.Normal.AsVector())) > 1e-9 {
		t.Error("fallback major axis is not perpendicular to the normal")
	}
}

func TestBSplineSurfaceNetValidation(t *testing.T) {
	w22 := [][]float64{{1, 1}, {1, 1}}
	k := []float64{0, 0, 1, 1}
	// Empty net.
	if _, err := NewBSplineSurface(1, 1, nil, nil, k, k); err == nil {
		t.Error("empty control net should error")
	}
	// Non-rectangular net.
	ragged := [][]math.Point3{{math.P3(0, 0, 0), math.P3(1, 0, 0)}, {math.P3(0, 1, 0)}}
	if _, err := NewBSplineSurface(1, 1, ragged, w22, k, k); err == nil {
		t.Error("ragged control net should error")
	}
	// Weight net not matching the control net.
	ctrl := [][]math.Point3{{math.P3(0, 0, 0), math.P3(1, 0, 0)}, {math.P3(0, 1, 0), math.P3(1, 1, 0)}}
	if _, err := NewBSplineSurface(1, 1, ctrl, [][]float64{{1, 1}}, k, k); err == nil {
		t.Error("mismatched weight net should error")
	}
	// A non-positive weight is rejected.
	if _, err := NewBSplineSurface(1, 1, ctrl, [][]float64{{1, 1}, {1, -1}}, k, k); err == nil {
		t.Error("a non-positive weight should error")
	}
	// A valid bilinear surface constructs and evaluates at its corners + interior, which
	// exercises findSpan's boundary and binary-search branches.
	s, err := NewBSplineSurface(1, 1, ctrl, w22, k, k)
	if err != nil {
		t.Fatalf("a valid bilinear B-spline surface rejected: %v", err)
	}
	for _, uv := range [][2]float64{{0, 0}, {1, 1}, {0.5, 0.5}} {
		if p := s.PointAt(uv[0], uv[1]); stdmath.IsNaN(float64(p.X)) {
			t.Errorf("PointAt%v returned NaN", uv)
		}
	}
}

func TestBSplineCurveMultiSpanEvaluation(t *testing.T) {
	// A degree-1 curve with 4 control points and interior knots (multiple spans) so
	// findSpan's binary search descends on both sides.
	ctrl := []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 0), math.P3(2, 0, 0), math.P3(3, 1, 0)}
	knots := []float64{0, 0, 1, 2, 3, 3}
	c, err := NewBSplineCurveUniformWeights(1, ctrl, knots)
	if err != nil {
		t.Fatalf("NewBSplineCurveUniformWeights: %v", err)
	}
	for _, u := range []float64{0, 0.5, 1.5, 2.5, 3} {
		if p := c.PointAt(u); stdmath.IsNaN(float64(p.X)) {
			t.Errorf("PointAt(%v) returned NaN", u)
		}
	}
}

func mustUnit(t *testing.T, x, y, z float64) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(x, y, z)
	if err != nil {
		t.Fatalf("NewUnitVector3(%g,%g,%g): %v", x, y, z, err)
	}
	return u
}
