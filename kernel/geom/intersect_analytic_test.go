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
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 0, 0, 1), cyl, ResolutionForSize(1))
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
	t.Parallel()
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 3, 0, 0, 1), sp, ResolutionForSize(1))
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
	t.Parallel()
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 10, 0, 0, 1), sp, ResolutionForSize(1))
	if !ok || len(curves) != 0 {
		t.Errorf("plane clear of sphere = %v ok=%v, want handled with no curve", curves, ok)
	}
}

// Two planes meet in a line; argument order does not matter.
func TestAnalyticPlanePlaneIsLine(t *testing.T) {
	t.Parallel()
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 0, 0, 1), mustPlane(t, 0, 0, 0, 1, 0, 0), ResolutionForSize(1))
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩plane = %v ok=%v, want one line", curves, ok)
	}
	if _, isLine := curves[0].(Line); !isLine {
		t.Errorf("plane∩plane curve = %T, want a Line", curves[0])
	}
}

// PlanePlaneLine returns the exact line the planar B-rep imprint uses: the z=0 and x=0 planes meet along the
// y-axis (a point on it and unit direction ±Y), and two parallel planes report ok=false.
func TestPlanePlaneLineExactAndParallel(t *testing.T) {
	t.Parallel()
	p0, dir, ok := PlanePlaneLine(mustPlane(t, 0, 0, 0, 0, 0, 1), mustPlane(t, 0, 0, 0, 1, 0, 0))
	if !ok {
		t.Fatal("z=0 ∩ x=0 must be a line, got ok=false")
	}
	if stdmath.Abs(float64(p0.X)) > 1e-12 || stdmath.Abs(float64(p0.Z)) > 1e-12 {
		t.Errorf("point on the intersection line %v must lie on the y-axis (x=z=0)", p0)
	}
	if l := float64(dir.Length()); stdmath.Abs(l-1) > 1e-12 {
		t.Errorf("direction length %g, want unit", l)
	}
	if stdmath.Abs(float64(dir.X)) > 1e-12 || stdmath.Abs(float64(dir.Z)) > 1e-12 {
		t.Errorf("direction %v must be ±Y (the shared y-axis)", dir)
	}
	if _, _, ok := PlanePlaneLine(mustPlane(t, 0, 0, 0, 0, 0, 1), mustPlane(t, 0, 0, 5, 0, 0, 1)); ok {
		t.Error("parallel planes must report ok=false (no isolated intersection line)")
	}
}

// Parallel planes are handled (no intersection curve).
func TestAnalyticParallelPlanes(t *testing.T) {
	t.Parallel()
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 0, 0, 1), mustPlane(t, 0, 0, 5, 0, 0, 1), ResolutionForSize(1))
	if !ok || len(curves) != 0 {
		t.Errorf("parallel planes = %v ok=%v, want handled with no curve", curves, ok)
	}
}

// An oblique plane∩cylinder is an exact ellipse: minor = radius, major = radius/|cosA|
// (here the 45° tilt gives cosA = 1/√2, so major = 2√2).
func TestAnalyticObliquePlaneCylinderIsEllipse(t *testing.T) {
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 1, 0, 1), cyl, ResolutionForSize(1))
	if !ok || len(curves) != 1 {
		t.Fatalf("oblique plane∩cylinder = %v ok=%v, want one ellipse", curves, ok)
	}
	e, isEllipse := curves[0].(EllipseFull)
	if !isEllipse || !near(e.MinorRadius, 2) || !near(e.MajorRadius, 2*stdmath.Sqrt2) {
		t.Errorf("ellipse = %+v, want minor 2 major %g", e, 2*stdmath.Sqrt2)
	}
}

// A plane parallel to the cylinder axis, cutting inside the radius, grazes the cylinder along a
// pair of axis-parallel lines (the section edges of a box-wall subtraction).
func TestAnalyticPlaneParallelToCylinderIsLinePair(t *testing.T) {
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 1, 0, 0), cyl, ResolutionForSize(1)) // x = 0
	if !ok || len(curves) != 2 {
		t.Fatalf("plane∥axis through center = %v ok=%v, want two lines", curves, ok)
	}
	for _, c := range curves {
		ln, isLine := c.(Line)
		if !isLine || !near(stdmath.Abs(float64(ln.Origin.Y)), 2) || !near(float64(ln.Origin.X), 0) {
			t.Errorf("line %+v, want axis-parallel through y=±2 on x=0", c)
		}
		if !near(stdmath.Abs(float64(ln.Dir.AsVector().Z)), 1) {
			t.Errorf("line direction %v, want along the cylinder axis", ln.Dir)
		}
	}
}

// A plane parallel to the axis but clear of (or tangent to) the cylinder grazes no enclosed
// region: handled, with no curves, so the half-space cut keeps the cylinder whole.
func TestAnalyticPlaneParallelClearsCylinder(t *testing.T) {
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 3, 0, 0, 1, 0, 0), cyl, ResolutionForSize(1)) // x = 3, radius 2
	if !ok || len(curves) != 0 {
		t.Errorf("plane clear of the cylinder = %v ok=%v, want handled with no curves", curves, ok)
	}
}

// A plane perpendicular to a cone's axis cuts a circle of radius distance×tan(halfAngle).
func TestAnalyticPlaneConeIsCircle(t *testing.T) {
	t.Parallel()
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/4) // tan 45° = 1
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 3, 0, 0, 1), cone, ResolutionForSize(1))
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
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	curves, ok := IntersectSurfacesAnalytic(cyl, mustPlane(t, 0, 0, 0, 0, 0, 1), ResolutionForSize(1))
	if !ok || len(curves) != 1 {
		t.Errorf("cylinder∩plane (swapped) = %v ok=%v, want one circle", curves, ok)
	}
}

// A plane perpendicular to a torus axis cuts the tube in two concentric circles (the outer R+√(r²−d²)
// and inner R−√(r²−d²)); an oblique plane that cuts the tube is a spiric quartic, deferred (handled=false).
func TestAnalyticPlaneTorusPerpendicularIsTwoCircles(t *testing.T) {
	t.Parallel()
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	curves, handled := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 1, 0, 0, 1), tor, ResolutionForSize(1)) // perpendicular at z=1
	if !handled || len(curves) != 2 {
		t.Fatalf("perpendicular torus cut: handled=%v, %d curves; want 2 circles", handled, len(curves))
	}
	want := []float64{5 + stdmath.Sqrt(4-1), 5 - stdmath.Sqrt(4-1)} // R ± √(r²−d²), d=1
	for i, c := range curves {
		circ, ok := c.(Circle)
		if !ok {
			t.Fatalf("curve %d is %T, want Circle", i, c)
		}
		if stdmath.Abs(circ.Radius-want[i]) > 1e-9 {
			t.Errorf("circle %d radius %.6f, want %.6f", i, circ.Radius, want[i])
		}
	}
}

func TestAnalyticPlaneTorusObliqueDefers(t *testing.T) {
	t.Parallel()
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if _, handled := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 0, 1, 0, 1), tor, ResolutionForSize(1)); handled {
		t.Error("an oblique torus cut (a spiric quartic) must defer, got handled=true")
	}
}

func TestAnalyticPlaneTorusClearsIsEmpty(t *testing.T) {
	t.Parallel()
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	curves, handled := IntersectSurfacesAnalytic(mustPlane(t, 20, 0, 0, 1, 0, 0), tor, ResolutionForSize(1)) // axis-parallel, far clear
	if !handled || len(curves) != 0 {
		t.Errorf("a plane clearing the torus must be handled with no curves, got handled=%v n=%d", handled, len(curves))
	}
}
