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
	circ, _ := NewCircle(math.P3(0, 0, 2), math.V3(0, 0, -1), 1.5)
	c2, ok := ProjectCurveToPlane(xyPlaneAt(0), circ)
	if !ok {
		t.Fatal("anti-parallel circle declined")
	}
	if cc, isCircle := c2.(Circle2d); !isCircle || stdmath.Abs(cc.Radius-1.5) > 1e-12 {
		t.Fatalf("want Circle2d r 1.5, got %T %+v", c2, c2)
	}
}

// TestProjectObliqueCircleDeclines: a tilted circle projects to an ellipse (phase 2), so the phase-1
// primitive declines and the caller falls back to sampling.
func TestProjectObliqueCircleDeclines(t *testing.T) {
	circ, _ := NewCircle(math.P3(0, 0, 0), math.V3(1, 0, 1), 2) // 45° tilt
	if _, ok := ProjectCurveToPlane(xyPlaneAt(0), circ); ok {
		t.Fatal("oblique circle must decline (ellipse not yet handled)")
	}
}

// TestProjectParallelArcToPlane: an arc parallel to the plane projects to a 2D arc of the same
// radius through its projected endpoints, with a point at the projected midpoint.
func TestProjectParallelArcToPlane(t *testing.T) {
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
	seg := NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	pts := SampleCurve3(seg, 16)
	if len(pts) != 17 {
		t.Fatalf("SampleCurve3(_,16) = %d points, want 17", len(pts))
	}
	if pts[0].DistanceTo(math.P3(0, 0, 0)) > 1e-12 || pts[16].DistanceTo(math.P3(10, 0, 0)) > 1e-12 {
		t.Fatalf("sampler endpoints = %v..%v, want (0,0,0)..(10,0,0)", pts[0], pts[16])
	}
}
