// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// xyPlaneAt is the world XY plane raised to z, with U=+X, V=+Y (so its normal is +Z).
func xyPlaneAt(z float64) Plane {
	pl, _ := NewPlaneFromAxes(math.P3(0, 0, z), math.V3(1, 0, 0), math.V3(0, 1, 0))
	return pl
}

// TestProjectLineSegmentToPlane: a 3D segment projects to the 2D segment of its projected endpoints.
func TestProjectLineSegmentToPlane(t *testing.T) {
	t.Parallel()
	seg := NewLineSegment(math.P3(1, 2, 5), math.P3(4, 6, 9))
	c2, ok := ProjectCurveToPlane(xyPlaneAt(0), seg)
	if !ok {
		t.Fatal("line segment declined")
	}
	l, isLine := c2.(LineSegment2d)
	if !isLine {
		t.Fatalf("want LineSegment2d, got %T", c2)
	}
	if l.StartPoint.DistanceTo(math.P2(1, 2)) > 1e-12 || l.EndPoint.DistanceTo(math.P2(4, 6)) > 1e-12 {
		t.Fatalf("projected segment = %v..%v, want (1,2)..(4,6)", l.StartPoint, l.EndPoint)
	}
}

// TestProjectParallelCircleToPlane: a circle whose plane is parallel to the sketch plane (any z
// offset) projects to a circle of the SAME radius, centre projected — the piston-rim case.
func TestProjectParallelCircleToPlane(t *testing.T) {
	t.Parallel()
	circ, _ := NewCircle(math.P3(3, -2, 8), math.V3(0, 0, 1), 4.3)
	c2, ok := ProjectCurveToPlane(xyPlaneAt(0), circ)
	if !ok {
		t.Fatal("parallel circle declined")
	}
	cc, isCircle := c2.(Circle2d)
	if !isCircle {
		t.Fatalf("want Circle2d, got %T", c2)
	}
	if cc.Center.DistanceTo(math.P2(3, -2)) > 1e-12 || stdmath.Abs(cc.Radius-4.3) > 1e-12 {
		t.Fatalf("projected circle = centre %v r %g, want (3,-2) r 4.3", cc.Center, cc.Radius)
	}
}

// TestProjectAntiParallelCircleToPlane: a circle whose normal is -Z (anti-parallel) still projects
// to a circle of the same radius (|cos| = 1).
func TestProjectAntiParallelCircleToPlane(t *testing.T) {
	t.Parallel()
	circ, _ := NewCircle(math.P3(0, 0, 2), math.V3(0, 0, -1), 1.5)
	c2, ok := ProjectCurveToPlane(xyPlaneAt(0), circ)
	if !ok {
		t.Fatal("anti-parallel circle declined")
	}
	if cc, isCircle := c2.(Circle2d); !isCircle || stdmath.Abs(cc.Radius-1.5) > 1e-12 {
		t.Fatalf("want Circle2d r 1.5, got %T %+v", c2, c2)
	}
}

// TestProjectObliqueCircleToEllipse: a circle tilted 45° about Y projects to an ellipse whose
// major axis (the rotation axis, Y) keeps radius r and whose minor axis (the tilt, X) is r·cos45°.
func TestProjectObliqueCircleToEllipse(t *testing.T) {
	t.Parallel()
	circ, _ := NewCircle(math.P3(1, 1, 0), math.V3(1, 0, 1), 2) // normal 45° from +Z, in the X–Z plane
	c2, ok := ProjectCurveToPlane(xyPlaneAt(0), circ)
	if !ok {
		t.Fatal("oblique circle declined")
	}
	e, isEllipse := c2.(EllipseFull2d)
	if !isEllipse {
		t.Fatalf("want EllipseFull2d, got %T", c2)
	}
	if e.Center.DistanceTo(math.P2(1, 1)) > 1e-9 {
		t.Fatalf("ellipse centre %v, want (1,1)", e.Center)
	}
	if stdmath.Abs(e.MajorRadius-2) > 1e-9 || stdmath.Abs(e.MinorRadius-2*stdmath.Cos(stdmath.Pi/4)) > 1e-9 {
		t.Fatalf("ellipse radii = %g/%g, want 2 / %g", e.MajorRadius, e.MinorRadius, 2*stdmath.Cos(stdmath.Pi/4))
	}
	if stdmath.Abs(float64(e.MajorAxis.X())) > 1e-9 { // major axis is ±Y (the rotation axis)
		t.Fatalf("major axis %v, want ±Y (the tilt shrinks X, not Y)", e.MajorAxis)
	}
}

// TestProjectObliqueArcToEllipticalArc: an oblique arc projects to an elliptical arc that covers the
// same geometry — its endpoints match the projected 3D endpoints, and every projected arc point lies
// on it.
func TestProjectObliqueArcToEllipticalArc(t *testing.T) {
	t.Parallel()
	arc, _ := NewArc3d(math.P3(0, 0, 0), math.V3(1, 0, 1), math.V3(0, 1, 0), 3, 0.3, stdmath.Pi*0.8)
	pl := xyPlaneAt(0)
	c2, ok := ProjectCurveToPlane(pl, arc)
	if !ok {
		t.Fatal("oblique arc declined")
	}
	ea, isEA := c2.(EllipticalArc2d)
	if !isEA {
		t.Fatalf("want EllipticalArc2d, got %T", c2)
	}
	if ea.PointAt(0).DistanceTo(planeUV(pl, arc.PointAt(0))) > 1e-9 ||
		ea.PointAt(1).DistanceTo(planeUV(pl, arc.PointAt(1))) > 1e-9 {
		t.Fatalf("elliptical-arc endpoints do not match the projected arc endpoints")
	}
	samples := make([]math.Point2, 401)
	for i := range samples {
		samples[i] = ea.PointAt(float64(i) / 400)
	}
	for i := 0; i <= 20; i++ { // every projected 3D arc point lies on the elliptical arc
		p := planeUV(pl, arc.PointAt(float64(i)/20))
		best := stdmath.Inf(1)
		for _, q := range samples {
			if d := float64(p.DistanceTo(q)); d < best {
				best = d
			}
		}
		if best > 1e-3 {
			t.Fatalf("projected arc point %v is %.4g off the elliptical arc", p, best)
		}
	}
}

// TestProjectParallelArcToPlane: an arc parallel to the plane projects to a 2D arc of the same
// radius through its projected endpoints, with a point at the projected midpoint.
func TestProjectParallelArcToPlane(t *testing.T) {
	t.Parallel()
	arc, _ := NewArc3d(math.P3(0, 0, 3), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, stdmath.Pi/2)
	c2, ok := ProjectCurveToPlane(xyPlaneAt(0), arc)
	if !ok {
		t.Fatal("parallel arc declined")
	}
	a2, isArc := c2.(Arc2d)
	if !isArc {
		t.Fatalf("want Arc2d, got %T", c2)
	}
	if a2.Center.DistanceTo(math.P2(0, 0)) > 1e-9 || stdmath.Abs(a2.Radius-2) > 1e-9 {
		t.Fatalf("projected arc centre %v r %g, want (0,0) r 2", a2.Center, a2.Radius)
	}
	// endpoints: start at angle 0 → (2,0); end at π/2 → (0,2).
	if a2.PointAt(0).DistanceTo(math.P2(2, 0)) > 1e-9 || a2.PointAt(1).DistanceTo(math.P2(0, 2)) > 1e-9 {
		t.Fatalf("projected arc endpoints %v..%v, want (2,0)..(0,2)", a2.PointAt(0), a2.PointAt(1))
	}
}

// TestSampleCurve3Count: the shared sampler returns n+1 points spanning the domain endpoints.
func TestSampleCurve3Count(t *testing.T) {
	t.Parallel()
	seg := NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	pts := SampleCurve3(seg, 16)
	if len(pts) != 17 {
		t.Fatalf("SampleCurve3(_,16) = %d points, want 17", len(pts))
	}
	if pts[0].DistanceTo(math.P3(0, 0, 0)) > 1e-12 || pts[16].DistanceTo(math.P3(10, 0, 0)) > 1e-12 {
		t.Fatalf("sampler endpoints = %v..%v, want (0,0,0)..(10,0,0)", pts[0], pts[16])
	}
}
