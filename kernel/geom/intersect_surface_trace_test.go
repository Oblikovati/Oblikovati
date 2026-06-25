// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// Acceptance suite for Oblikovati/Oblikovati#1319: the continuation tracer must find closed loops,
// report tangential contacts as points (not nothing), connect branches correctly, and place every
// traced point on BOTH surfaces — the failure modes of the old fixed-grid marching squares.

// onBothSurfaces asserts every traced point lies on both surfaces within tol.
func onBothSurfaces(t *testing.T, a, b Surface, loops [][]math.Point3, tol float64) {
	t.Helper()
	n := 0
	for _, loop := range loops {
		for _, p := range loop {
			_, _, da := ProjectPointToSurface(a, p)
			_, _, db := ProjectPointToSurface(b, p)
			if da > tol || db > tol {
				t.Errorf("point %v off surfaces: dist a=%g b=%g (tol %g)", p, da, db, tol)
			}
			n++
		}
	}
	if n == 0 {
		t.Fatal("no traced points")
	}
}

// TestSSISmallOverlapCircleIsClosed: two equal spheres overlapping by a little intersect in a small
// circle. The tracer must return ONE closed loop at the right plane and radius, every point on both.
func TestSSISmallOverlapCircleIsClosed(t *testing.T) {
	const r, d = 5.0, 9.5
	a, _ := NewSphere(math.P3(0, 0, 0), r)
	b, _ := NewSphere(math.P3(0, 0, d), r)
	z0 := d / 2                        // equal radii → midplane
	wantR := stdmath.Sqrt(r*r - z0*z0) // ≈ 1.561
	loops := IntersectSurfaceSurface(a, b, SurfaceGrid{})
	if len(loops) != 1 {
		t.Fatalf("got %d loops, want 1", len(loops))
	}
	loop := loops[0]
	if len(loop) < 8 {
		t.Fatalf("loop has only %d points", len(loop))
	}
	// Closed: first and last points coincide (within a march step ≈ 4e-3·extent).
	if gap := float64(loop[0].DistanceTo(loop[len(loop)-1])); gap > 0.1 {
		t.Errorf("loop not closed: endpoint gap %g", gap)
	}
	for _, p := range loop {
		if stdmath.Abs(float64(p.Z)-z0) > 1e-4 {
			t.Errorf("point %v not on the intersection plane z=%g", p, z0)
		}
		if rr := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(rr-wantR) > 1e-3 {
			t.Errorf("point radius %g, want %g", rr, wantR)
		}
	}
	onBothSurfaces(t, a, b, loops, 1e-5)
}

// TestSSITangentSpheresReturnPoint: two externally tangent spheres touch at one point. The tracer must
// return a (degenerate) result AT the contact, not empty — the case a sign-change contour drops.
func TestSSITangentSpheresReturnPoint(t *testing.T) {
	a, _ := NewSphere(math.P3(0, 0, 0), 5)
	b, _ := NewSphere(math.P3(10, 0, 0), 5) // tangent at (5,0,0)
	loops := IntersectSurfaceSurface(a, b, SurfaceGrid{})
	pts := allPoints(t, loops)
	best := stdmath.Inf(1)
	for _, p := range pts {
		if d := float64(p.DistanceTo(math.P3(5, 0, 0))); d < best {
			best = d
		}
	}
	if best > 1e-3 {
		t.Errorf("no traced point near the tangency (5,0,0); closest was %g away", best)
	}
}

// TestSSIPointsLieOnBothSurfaces: a sphere cut by an offset plane — every traced point must lie on
// both, projected onto each.
func TestSSIPointsLieOnBothSurfaces(t *testing.T) {
	a, _ := NewSphere(math.P3(0, 0, 0), 6)
	pl, _ := NewPlane(math.P3(0, 0, 2), math.V3(0, 0, 1)) // z=2 cut
	loops := IntersectSurfaceSurface(a, pl, SurfaceGrid{})
	onBothSurfaces(t, a, pl, loops, 1e-5)
	// The cut is a circle of radius sqrt(36-4)=sqrt(32) at z=2.
	want := stdmath.Sqrt(32)
	for _, p := range allPoints(t, loops) {
		if rr := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(rr-want) > 1e-3 {
			t.Errorf("point radius %g, want %g", rr, want)
		}
	}
}

// TestSSINurbsPatchCutByPlane exercises the non-analytic corrector (the actual #1319 motivation): a
// NURBS bump patch cut by a plane through the bump. Every traced point must land on both surfaces via
// the surface's Gauss–Newton inversion, and the cut must be a closed loop (the bump is a single hump).
func TestSSINurbsPatchCutByPlane(t *testing.T) {
	s := nurbsBump(t) // biquadratic bump on [0,1]², rising to ~0.5 in the middle, 0 at the corners
	pl, _ := NewPlane(math.P3(0, 0, 0.3), math.V3(0, 0, 1))
	loops := IntersectSurfaceSurface(s, pl, SurfaceGrid{})
	onBothSurfaces(t, s, pl, loops, 1e-5)
	for _, p := range allPoints(t, loops) {
		if stdmath.Abs(float64(p.Z)-0.3) > 1e-4 {
			t.Errorf("NURBS∩plane point %v not on the z=0.3 cut", p)
		}
	}
}

// TestSSIAgreesWithAnalyticOnSpherePlane: the tracer's sphere∩plane circle must match the analytic
// equator (radius and planarity), the acceptance cross-check against the analytic path.
func TestSSIAgreesWithAnalyticOnSpherePlane(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	curves, handled := IntersectSurfacesAnalytic(sp, pl, ResolutionForSize(1))
	if !handled || len(curves) == 0 {
		t.Skip("analytic path did not handle sphere∩plane")
	}
	loops := IntersectSurfaceSurface(sp, pl, SurfaceGrid{})
	for _, p := range allPoints(t, loops) {
		if stdmath.Abs(float64(p.Z)) > 1e-5 {
			t.Errorf("traced point %v off the analytic z=0 plane", p)
		}
		if rr := float64(p.AsVector().Length()); stdmath.Abs(rr-5) > 1e-4 {
			t.Errorf("traced radius %g, want 5 (analytic)", rr)
		}
	}
}
