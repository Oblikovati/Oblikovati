// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func mustPlane(t *testing.T, ox, oy, oz, nx, ny, nz float64) Plane {
	t.Helper()
	pl, err := NewPlane(math.P3(math.Scalar(ox), math.Scalar(oy), math.Scalar(oz)), math.V3(math.Scalar(nx), math.Scalar(ny), math.Scalar(nz)))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return pl
}

// A plane perpendicular to a cylinder axis cuts it in an exact circle of the cylinder's
// radius.
func TestAnalyticPlaneCylinderIsCircle(t *testing.T) {
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 0, 0, 1), cyl)
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩cylinder = %v ok=%v, want one curve", curves, ok)
	}
	c, isCircle := curves[0].(Circle)
	if !isCircle || !near(c.Radius, 2) || !near(float64(c.Center.Z), 0) {
		t.Errorf("circle = %+v, want radius 2 at z=0", c)
	}
}

// Cutting a plane at z=3 through a sphere of radius 5 gives a circle of radius 4.
func TestAnalyticPlaneSphereSmallCircle(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 3, 0, 0, 1), sp)
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩sphere = %v ok=%v, want one curve", curves, ok)
	}
	c := curves[0].(Circle)
	if !near(c.Radius, 4) || !near(float64(c.Center.Z), 3) {
		t.Errorf("circle = %+v, want radius 4 at z=3", c)
	}
}

// A plane clear of the sphere is handled with no curve (not a fallback).
func TestAnalyticPlaneSphereMiss(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 10, 0, 0, 1), sp)
	if !ok || len(curves) != 0 {
		t.Errorf("plane clear of sphere = %v ok=%v, want handled with no curve", curves, ok)
	}
}

// Two planes meet in a line; argument order does not matter.
func TestAnalyticPlanePlaneIsLine(t *testing.T) {
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 0, 0, 1), mustPlane(t, 0, 0, 0, 1, 0, 0))
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩plane = %v ok=%v, want one line", curves, ok)
	}
	if _, isLine := curves[0].(Line); !isLine {
		t.Errorf("plane∩plane curve = %T, want a Line", curves[0])
	}
}

// Parallel planes are handled (no intersection curve).
func TestAnalyticParallelPlanes(t *testing.T) {
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 0, 0, 1), mustPlane(t, 0, 0, 5, 0, 0, 1))
	if !ok || len(curves) != 0 {
		t.Errorf("parallel planes = %v ok=%v, want handled with no curve", curves, ok)
	}
}

// An oblique plane∩cylinder is an exact ellipse: minor = radius, major = radius/|cosA|
// (here the 45° tilt gives cosA = 1/√2, so major = 2√2).
func TestAnalyticObliquePlaneCylinderIsEllipse(t *testing.T) {
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 1, 0, 1), cyl)
	if !ok || len(curves) != 1 {
		t.Fatalf("oblique plane∩cylinder = %v ok=%v, want one ellipse", curves, ok)
	}
	e, isEllipse := curves[0].(EllipseFull)
	if !isEllipse || !near(e.MinorRadius, 2) || !near(e.MajorRadius, 2*stdmath.Sqrt2) {
		t.Errorf("ellipse = %+v, want minor 2 major %g", e, 2*stdmath.Sqrt2)
	}
}

// A plane parallel to the cylinder axis (a line pair) is left to the numeric tracer.
func TestAnalyticPlaneParallelToCylinderDefers(t *testing.T) {
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if _, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 1, 0, 0), cyl); ok {
		t.Error("a plane parallel to the cylinder axis should defer (handled=false)")
	}
}

// A plane perpendicular to a cone's axis cuts a circle of radius distance×tan(halfAngle).
func TestAnalyticPlaneConeIsCircle(t *testing.T) {
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/4) // tan 45° = 1
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 3, 0, 0, 1), cone)
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩cone = %v ok=%v, want one circle", curves, ok)
	}
	c := curves[0].(Circle)
	if !near(c.Radius, 3) || !near(float64(c.Center.Z), 3) {
		t.Errorf("cone circle = %+v, want radius 3 at z=3", c)
	}
}

// Argument order is symmetric (cylinder first).
func TestAnalyticOrderIndependent(t *testing.T) {
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(cyl, mustPlane(t, 0, 0, 0, 0, 0, 1))
	if !ok || len(curves) != 1 {
		t.Errorf("cylinder∩plane (swapped) = %v ok=%v, want one circle", curves, ok)
	}
}
