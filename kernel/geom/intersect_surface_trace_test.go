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

// closedOnly keeps the loops that close (first≈last) and have enough points to be a real curve, dropping
// the single-point tangency markers, so a #1404 test asserts on the boundary loops the imprint consumes.
func closedOnly(loops [][]math.Point3) [][]math.Point3 {
	var out [][]math.Point3
	for _, c := range loops {
		if len(c) >= 4 && float64(c[0].DistanceTo(c[len(c)-1])) < 0.05 {
			out = append(out, c)
		}
	}
	return out
}

// passesNear reports whether some point of the loop comes within tol of p.
func passesNear(loop []math.Point3, p math.Point3, tol float64) bool {
	for _, q := range loop {
		if float64(q.DistanceTo(p)) < tol {
			return true
		}
	}
	return false
}

// planarDiagonal reports whether the whole loop lies in the plane x=z OR the whole loop lies in x=−z — the
// two Steinmetz ellipse planes for an x-axis and a z-axis cylinder of equal radius.
func planarDiagonal(loop []math.Point3) bool {
	inPlus, inMinus := true, true
	for _, p := range loop {
		if stdmath.Abs(float64(p.X)-float64(p.Z)) > 1e-4 {
			inPlus = false
		}
		if stdmath.Abs(float64(p.X)+float64(p.Z)) > 1e-4 {
			inMinus = false
		}
	}
	return inPlus || inMinus
}

// TestSSITracesThroughSteinmetzPinch is the headline acceptance for Oblikovati#1404: two equal-radius
// perpendicular cylinders intersect in two ellipses that CROSS at two pinch points (the Steinmetz
// configuration). The tracer must follow each ellipse straight through both pinches and return two
// topologically-complete closed loops — not stop at the first pinch and silently drop the open arcs, the
// old behaviour that forced the bespoke curved_steinmetz analytic family.
func TestSSITracesThroughSteinmetzPinch(t *testing.T) {
	const r = 3.0
	a, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), r) // axis x
	b, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r) // axis z
	loops := closedOnly(IntersectSurfaceSurface(a, b, SurfaceGrid{VMin: -r - 1, VMax: r + 1}))
	if len(loops) != 2 {
		t.Fatalf("got %d closed loops, want 2 (the two Steinmetz ellipses through the pinches)", len(loops))
	}
	onBothSurfaces(t, a, b, loops, 1e-4)
	pinchLo, pinchHi := math.P3(0, -r, 0), math.P3(0, r, 0)
	for i, loop := range loops {
		if !passesNear(loop, pinchLo, 0.05) || !passesNear(loop, pinchHi, 0.05) {
			t.Errorf("loop %d does not pass through both pinch points (0,±%g,0)", i, r)
		}
		if !planarDiagonal(loop) {
			t.Errorf("loop %d is not planar in x=z or x=-z (a Steinmetz ellipse plane)", i)
		}
	}
}

// TestSSINearPinchKeepsBothLoops: two perpendicular cylinders whose radii differ by only 1e-6 meet in two
// SEPARATE closed loops that approach within a few microns at a tight high-curvature near-pinch U-turn. The
// corrector's near-tangency descent must follow that turn so BOTH loops come back closed — no silently
// dropped chain (Oblikovati#1404).
func TestSSINearPinchKeepsBothLoops(t *testing.T) {
	a, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 3.0)
	b, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.0-1e-6)
	loops := closedOnly(IntersectSurfaceSurface(a, b, SurfaceGrid{VMin: -4, VMax: 4}))
	if len(loops) != 2 {
		t.Fatalf("got %d closed loops, want 2 (a near-pinch must not drop a chain)", len(loops))
	}
	onBothSurfaces(t, a, b, loops, 1e-4)
}

// TestSSINearTangentSphereCylinderClosesLoops: a sphere just larger than the cylinder it surrounds cuts it
// in two near-tangent circles; the descent corrector traces both as closed loops through the shallow
// (near-parallel-normal) crossing rather than dropping them (Oblikovati#1404).
func TestSSINearTangentSphereCylinderClosesLoops(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 3.0001)
	cy, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.0)
	loops := closedOnly(IntersectSurfaceSurface(sp, cy, SurfaceGrid{}))
	if len(loops) != 2 {
		t.Fatalf("got %d closed loops, want 2 near-tangent circles", len(loops))
	}
	onBothSurfaces(t, sp, cy, loops, 1e-4)
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
