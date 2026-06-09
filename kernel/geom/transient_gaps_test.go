// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// --- CircleByThreePoints / PlaneByThreePoints ------------------------------

func TestCircleByThreePoints(t *testing.T) {
	// Three points on the unit circle in the XY plane at angles 0, 90, 180.
	a, b, c := math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(-1, 0, 0)
	circ, err := CircleByThreePoints(a, b, c)
	if err != nil {
		t.Fatalf("CircleByThreePoints: %v", err)
	}
	if !nearPoint(circ.Center, math.P3(0, 0, 0)) {
		t.Errorf("center = %v, want origin", circ.Center)
	}
	if stdmath.Abs(circ.Radius-1) > 1e-9 {
		t.Errorf("radius = %v, want 1", circ.Radius)
	}
	for _, p := range []math.Point3{a, b, c} {
		if stdmath.Abs(circ.Center.DistanceTo(p)-circ.Radius) > 1e-9 {
			t.Errorf("point %v not on the circle", p)
		}
	}
}

func TestCircleByThreePointsCollinearFails(t *testing.T) {
	_, err := CircleByThreePoints(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0))
	if err == nil {
		t.Error("collinear points should not define a circle")
	}
}

func TestPlaneByThreePoints(t *testing.T) {
	a, b, c := math.P3(0, 0, 1), math.P3(2, 0, 1), math.P3(0, 3, 1)
	pl, err := PlaneByThreePoints(a, b, c)
	if err != nil {
		t.Fatalf("PlaneByThreePoints: %v", err)
	}
	// All three points lie on the plane: (p−Origin)·Normal == 0.
	n := pl.Normal()
	for _, p := range []math.Point3{a, b, c} {
		if d := pl.Origin.VectorTo(p).Dot(n); stdmath.Abs(d) > 1e-9 {
			t.Errorf("point %v off-plane by %v", p, d)
		}
	}
	// The plane is z=1, so the normal is ±Z.
	if stdmath.Abs(stdmath.Abs(n.Z)-1) > 1e-9 {
		t.Errorf("normal = %v, want ±Z", n)
	}
}

func TestPlaneByThreePointsCollinearFails(t *testing.T) {
	_, err := PlaneByThreePoints(math.P3(0, 0, 0), math.P3(1, 1, 1), math.P3(2, 2, 2))
	if err == nil {
		t.Error("collinear points should not define a plane")
	}
}

// --- EllipticalCylinder ----------------------------------------------------

func TestEllipticalCylinderPointAndParam(t *testing.T) {
	c, err := NewEllipticalCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 1)
	if err != nil {
		t.Fatalf("NewEllipticalCylinder: %v", err)
	}
	if !nearPoint(c.PointAt(0, 0), math.P3(3, 0, 0)) {
		t.Errorf("PointAt(0,0) = %v, want major-axis tip (3,0,0)", c.PointAt(0, 0))
	}
	if !nearPoint(c.PointAt(stdmath.Pi/2, 2), math.P3(0, 1, 2)) {
		t.Errorf("PointAt(π/2,2) = %v, want (0,1,2)", c.PointAt(stdmath.Pi/2, 2))
	}
	// ParamAt inverts PointAt on the surface.
	for _, uv := range [][2]float64{{0.5, 1.3}, {2.0, -0.7}, {4.1, 2.2}} {
		gotU, gotV := c.ParamAt(c.PointAt(uv[0], uv[1]))
		if stdmath.Abs(gotU-uv[0]) > 1e-9 || stdmath.Abs(gotV-uv[1]) > 1e-9 {
			t.Errorf("ParamAt∘PointAt(%v) = (%v,%v)", uv, gotU, gotV)
		}
	}
}

func TestEllipticalCylinderNormalOutwardUnit(t *testing.T) {
	c, _ := NewEllipticalCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 1)
	n := c.NormalAt(0, 0)
	if stdmath.Abs(n.Length()-1) > 1e-9 {
		t.Errorf("normal length = %v, want 1", n.Length())
	}
	if !nearVec(n, math.V3(1, 0, 0)) { // outward at the major-axis tip
		t.Errorf("normal at (0,0) = %v, want +X", n)
	}
}

// --- EllipticalCone --------------------------------------------------------

func TestEllipticalConePointAndParam(t *testing.T) {
	c, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), stdmath.Pi/4, stdmath.Pi/6)
	if err != nil {
		t.Fatalf("NewEllipticalCone: %v", err)
	}
	// At v=2,u=0 the major semi-axis is 2·tan(π/4)=2, so the point is (2,0,2).
	if !nearPoint(c.PointAt(0, 2), math.P3(2, 0, 2)) {
		t.Errorf("PointAt(0,2) = %v, want (2,0,2)", c.PointAt(0, 2))
	}
	for _, uv := range [][2]float64{{0.5, 1.3}, {2.0, 0.7}, {4.1, 2.2}} {
		gotU, gotV := c.ParamAt(c.PointAt(uv[0], uv[1]))
		if stdmath.Abs(gotU-uv[0]) > 1e-9 || stdmath.Abs(gotV-uv[1]) > 1e-9 {
			t.Errorf("ParamAt∘PointAt(%v) = (%v,%v)", uv, gotU, gotV)
		}
	}
}

func TestEllipticalConeRejectsBadAngle(t *testing.T) {
	if _, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0, stdmath.Pi/6); err == nil {
		t.Error("a zero half angle should be rejected")
	}
	if _, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), stdmath.Pi/4, stdmath.Pi); err == nil {
		t.Error("a half angle >= π/2 should be rejected")
	}
}

// --- BSplineCurve2d --------------------------------------------------------

func TestBSplineCurve2dQuadratic(t *testing.T) {
	// A clamped degree-2 curve over three control points is the quadratic Bézier.
	ctrl := []math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(2, 0)}
	knots := []float64{0, 0, 0, 1, 1, 1}
	c, err := NewBSplineCurve2dUniformWeights(2, ctrl, knots)
	if err != nil {
		t.Fatalf("NewBSplineCurve2d: %v", err)
	}
	if !nearPoint2(c.PointAt(0), ctrl[0]) || !nearPoint2(c.PointAt(1), ctrl[2]) {
		t.Errorf("endpoints = %v,%v, want %v,%v", c.PointAt(0), c.PointAt(1), ctrl[0], ctrl[2])
	}
	if !nearPoint2(c.PointAt(0.5), math.P2(1, 1)) { // 0.25P0+0.5P1+0.25P2
		t.Errorf("midpoint = %v, want (1,1)", c.PointAt(0.5))
	}
	if tan := c.TangentAt(0); !nearVec2(tan, math.V2(2, 4)) { // 2(P1−P0)
		t.Errorf("TangentAt(0) = %v, want (2,4)", tan)
	}
}

// --- PolylineFromCurve -----------------------------------------------------

func TestPolylineFromCurve3OnCircle(t *testing.T) {
	circ, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	pl, err := PolylineFromCurve3(circ, 8)
	if err != nil {
		t.Fatalf("PolylineFromCurve3: %v", err)
	}
	if len(pl.Vertices) != 9 {
		t.Fatalf("got %d vertices, want 9 (8 segments + 1)", len(pl.Vertices))
	}
	for _, v := range pl.Vertices {
		if stdmath.Abs(v.DistanceTo(math.P3(0, 0, 0))-2) > 1e-9 {
			t.Errorf("vertex %v not on the circle of radius 2", v)
		}
	}
}

func TestPolylineFromCurveRejectsUnbounded(t *testing.T) {
	line, _ := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0)) // infinite domain
	if _, err := PolylineFromCurve3(line, 4); err == nil {
		t.Error("an unbounded curve has no finite polyline")
	}
	circ, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	if _, err := PolylineFromCurve3(circ, 0); err == nil {
		t.Error("zero segments should be rejected")
	}
}

// --- FittedBSplineCurve ----------------------------------------------------

func TestFittedBSplineCurve3dInterpolates(t *testing.T) {
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 0), math.P3(2, 0, 1), math.P3(3, 1, 0)}
	c, err := NewFittedBSplineCurve(pts)
	if err != nil {
		t.Fatalf("NewFittedBSplineCurve: %v", err)
	}
	for k, u := range chordParamsOracle3(pts) {
		if got := c.PointAt(u); !nearPoint(got, pts[k]) {
			t.Errorf("curve at ū=%v = %v, want input point %v", u, got, pts[k])
		}
	}
}

func TestFittedBSplineCurve2dInterpolates(t *testing.T) {
	pts := []math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(3, 1)}
	c, err := NewFittedBSplineCurve2d(pts)
	if err != nil {
		t.Fatalf("NewFittedBSplineCurve2d: %v", err)
	}
	for k, u := range chordParamsOracle2(pts) {
		if got := c.PointAt(u); !nearPoint2(got, pts[k]) {
			t.Errorf("curve at ū=%v = %v, want input point %v", u, got, pts[k])
		}
	}
}

func TestFittedBSplineTwoPointsIsLine(t *testing.T) {
	c, err := NewFittedBSplineCurve([]math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0)})
	if err != nil {
		t.Fatalf("NewFittedBSplineCurve: %v", err)
	}
	if c.Degree != 1 {
		t.Errorf("degree = %d, want 1 for two points", c.Degree)
	}
	if !nearPoint(c.PointAt(0.5), math.P3(1, 0, 0)) {
		t.Errorf("midpoint = %v, want (1,0,0)", c.PointAt(0.5))
	}
}

func TestFittedBSplineRejectsCoincident(t *testing.T) {
	_, err := NewFittedBSplineCurve([]math.Point3{math.P3(1, 1, 1), math.P3(1, 1, 1)})
	if err == nil {
		t.Error("coincident points have no chord length and should be rejected")
	}
}

// nearVec / nearVec2 compare vectors within 1e-9.
func nearVec(a, b math.Vector3) bool  { return a.Sub(b).Length() < 1e-9 }
func nearVec2(a, b math.Vector2) bool { return a.Sub(b).Length() < 1e-9 }

// chordParamsOracle3 / chordParamsOracle2 reproduce the constructor's chord-length
// parameterization independently, giving the parameters at which a true interpolating
// spline must reproduce its input points.
func chordParamsOracle3(pts []math.Point3) []float64 {
	d := make([]float64, len(pts))
	for k := 1; k < len(pts); k++ {
		d[k] = d[k-1] + float64(pts[k-1].DistanceTo(pts[k]))
	}
	return normalizeCumulative(d)
}

func chordParamsOracle2(pts []math.Point2) []float64 {
	d := make([]float64, len(pts))
	for k := 1; k < len(pts); k++ {
		d[k] = d[k-1] + float64(pts[k-1].DistanceTo(pts[k]))
	}
	return normalizeCumulative(d)
}

func normalizeCumulative(d []float64) []float64 {
	total := d[len(d)-1]
	u := make([]float64, len(d))
	for k := range u {
		u[k] = d[k] / total
	}
	u[len(u)-1] = 1
	return u
}
