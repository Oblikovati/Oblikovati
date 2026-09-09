// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func isectBox(lo, hi float64) math.Box {
	return math.NewBox(math.P3(lo, lo, lo), math.P3(hi, hi, hi))
}

// curvesOnBothSurfaces asserts every sample of every returned curve lies on both surfaces within tol.
func curvesOnBothSurfaces(t *testing.T, curves []Curve3, a, b Surface, tol float64) {
	t.Helper()
	for _, c := range curves {
		lo, hi := c.Domain()
		for i := 0; i <= 16; i++ {
			p := c.PointAt(lo + (hi-lo)*float64(i)/16)
			if da := stdmath.Abs(SignedDistanceToSurface(a, p)); da > tol {
				t.Errorf("curve point %v off surface a by %g (tol %g)", p, da, tol)
			}
			if db := stdmath.Abs(SignedDistanceToSurface(b, p)); db > tol {
				t.Errorf("curve point %v off surface b by %g (tol %g)", p, db, tol)
			}
		}
	}
}

// TestSurfaceIntersectClosedFormPlaneSphere: the closed-form path returns the equator as an analytic
// Circle (not a polyline), on both surfaces.
func TestSurfaceIntersectClosedFormPlaneSphere(t *testing.T) {
	t.Parallel()
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	box := isectBox(-6, 6)
	curves, handled := SurfaceIntersect(pl, sp, box, ResolutionForBox(box))
	if !handled || len(curves) != 1 {
		t.Fatalf("plane∩sphere: handled=%v curves=%d, want 1", handled, len(curves))
	}
	if _, ok := curves[0].(Circle); !ok {
		t.Errorf("closed-form plane∩sphere must be an analytic Circle, got %T", curves[0])
	}
	curvesOnBothSurfaces(t, curves, pl, sp, 1e-9)
}

// TestSurfaceIntersectMarchesCrossingCylinders: no closed form exists for cylinder∩cylinder, so the
// general marcher runs, windowed to the box from the base cylinder, and returns the two Steinmetz
// saddle curves — every point on both cylinders.
func TestSurfaceIntersectMarchesCrossingCylinders(t *testing.T) {
	t.Parallel()
	const r = 3.0
	cx, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), r)
	cz, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	box := isectBox(-4, 4)
	curves, handled := SurfaceIntersect(cx, cz, box, ResolutionForBox(box))
	if !handled || len(curves) != 2 {
		t.Fatalf("cylinder∩cylinder: handled=%v curves=%d, want the 2 marched saddle curves", handled, len(curves))
	}
	curvesOnBothSurfaces(t, curves, cx, cz, 1e-3) // marched (polyline) tolerance
}

// TestSurfaceIntersectNonCrossingIsHandledEmpty: a plane clear of the sphere is a definite non-crossing —
// handled=true with no curves (the closed form knows they do not meet), so the caller composes cleanly.
func TestSurfaceIntersectNonCrossingIsHandledEmpty(t *testing.T) {
	t.Parallel()
	sp, _ := NewSphere(math.P3(0, 0, 0), 1)
	pl, _ := NewPlane(math.P3(0, 0, 5), math.V3(0, 0, 1)) // 5 units above a unit sphere
	box := isectBox(-6, 6)
	curves, handled := SurfaceIntersect(pl, sp, box, ResolutionForBox(box))
	if !handled || len(curves) != 0 {
		t.Fatalf("plane clear of sphere: handled=%v curves=%d, want handled+empty", handled, len(curves))
	}
}

// TestTheMarchCarriesTheClosedFormsRefusal: a pair the closed form gives up for CONDITIONING is
// answered by the tracer, and the demotion used to vanish there — SurfaceIntersect discarded the
// reason, so a caller could not tell an exact section from a marched stand-in for one
// (Oblikovati/Oblikovati#3525). handled stays true; the reason rides along.
func TestTheMarchCarriesTheClosedFormsRefusal(t *testing.T) {
	t.Parallel()
	ring, err := NewTorus(math.P3(0, 0, 0), math.V3(1, 0, 0), 6, 1.5)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	rod, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	box := math.NewBox(math.P3(-9, -9, -9), math.P3(9, 9, 9))
	if _, why, _ := IntersectSurfacesAnalyticDeclining(ring, rod, ResolutionForBox(box)); !why.IsConditioning() {
		t.Fatalf("the fixture no longer demotes for conditioning: why=%v", why)
	}
	curves, why, handled := SurfaceIntersectDeclining(ring, rod, box, ResolutionForBox(box))
	if !handled || len(curves) == 0 {
		t.Fatalf("the marcher answered handled=%v with %d curves; the fixture needs a marched answer", handled, len(curves))
	}
	if !why.IsConditioning() {
		t.Errorf("the marched answer reports why=%v; the closed form's conditioning demotion was dropped", why)
	}
}

// TestAnExactSectionReportsNoDemotion is the other half: a pair the closed form solves reports
// DeclineNone, so a caller cannot mistake every section for a degraded one.
func TestAnExactSectionReportsNoDemotion(t *testing.T) {
	t.Parallel()
	pl, err := NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	sp, err := NewSphere(math.P3(0, 0, 0), 2)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	box := math.NewBox(math.P3(-3, -3, -3), math.P3(3, 3, 3))
	curves, why, handled := SurfaceIntersectDeclining(pl, sp, box, ResolutionForBox(box))
	if !handled || len(curves) != 1 {
		t.Fatalf("plane∩sphere: handled=%v with %d curves, want the one exact circle", handled, len(curves))
	}
	if why != DeclineNone {
		t.Errorf("an exact section reports why=%v, want DeclineNone", why)
	}
}
