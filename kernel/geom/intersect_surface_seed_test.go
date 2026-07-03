// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// onSurfacePair fails if any point of loops is not on BOTH surfaces within tol — the invariant every
// SSI seed/curve must satisfy, regardless of how thin the feature is.
func onSurfacePair(t *testing.T, loops [][]math.Point3, a, b Surface, tol float64) {
	t.Helper()
	pts := allPoints(t, loops)
	for _, p := range pts {
		_, _, da := ProjectPointToSurface(a, p)
		_, _, db := ProjectPointToSurface(b, p)
		if da > tol || db > tol {
			t.Errorf("point %v off surfaces (da=%g, db=%g, tol=%g)", p, da, db, tol)
		}
	}
}

// TestSSIFindsThinSubGridLoop is acceptance criterion 1 of #1400: an intersection loop far thinner than
// the retired fixed grid's spacing is found. A small sphere centred ON a unit sphere's surface meets it
// in a circle of 3D radius ~0.015 — a parameter loop ~0.03 across, well below the old du≈2π/160≈0.039.
// The centre is placed off any coarse grid line so the result cannot depend on grid alignment.
func TestSSIFindsThinSubGridLoop(t *testing.T) {
	big, _ := NewSphere(math.P3(0, 0, 0), 1)
	centre := big.PointAt(0.31, 0.21) // a point on the unit sphere, off the coarse seed lattice
	small, _ := NewSphere(centre, 0.015)

	loops := IntersectSurfaceSurface(big, small, SurfaceGrid{})
	if len(loops) == 0 {
		t.Fatal("thin sub-grid intersection loop was dropped — the #1400 regression")
	}
	onSurfacePair(t, loops, big, small, 1e-5)
}

// TestSSIDetectsTangentialContact is acceptance criterion 1's degenerate case: a plane grazing a sphere
// touches at a single point where the signed-distance field reaches zero WITHOUT changing sign (the
// sphere lies entirely on one side), so the old sign-change seeding could not see it. The adaptive
// seeder's sub-cell minimum still seeds it and the tracer reports the contact.
func TestSSIDetectsTangentialContact(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 1)
	pl, _ := NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1)) // tangent at the north pole (0,0,1)

	contact := math.P3(0, 0, 1)
	found := false
	for _, p := range allPoints(t, IntersectSurfaceSurface(sp, pl, SurfaceGrid{})) {
		if p.DistanceTo(contact) < 1e-3 {
			found = true
		}
	}
	if !found {
		t.Errorf("tangential contact at %v not detected", contact)
	}
}

// TestSSISeederPrunesEmptySpace is acceptance criterion 2 of #1400: where the intersection feature is
// small relative to the domain, the adaptive seeder evaluates the (projection-bearing) field far fewer
// times than the retired 161×161 = 25921-node grid, because cells nowhere near the other surface are
// pruned by the Lipschitz bound. Target: at least a 10× reduction.
func TestSSISeederPrunesEmptySpace(t *testing.T) {
	const fixedGridNodes = 161 * 161
	big, _ := NewSphere(math.P3(0, 0, 0), 1)
	small, _ := NewSphere(big.PointAt(0.31, 0.21), 0.015)
	g := resolveGrid(big, SurfaceGrid{})

	c := newSSISeedField(big, small, g)
	seeds := c.seeds(ssiStep(big, g))
	if len(seeds) == 0 {
		t.Fatal("seeder found no seeds for a real intersection")
	}
	if c.evals*10 > fixedGridNodes {
		t.Errorf("seeder used %d field evals; want < %d (10x below the fixed grid)", c.evals, fixedGridNodes/10)
	}
	t.Logf("adaptive seeder: %d field evals (%.0fx fewer than the %d-node grid)",
		c.evals, float64(fixedGridNodes)/float64(c.evals), fixedGridNodes)
}

// TestSSIRippleFindsEveryCrossing is the #1608 prune-soundness guard: a 24-span rippled
// B-spline sheet (one bump per span) crossed by the z=0 plane meets it in one iso-u curve per
// sign change of the sheet's height spline. The quadtree prune — now certified by the
// hodograph derivative bound instead of a sampled 2.0 guess — must not discard any cell
// containing a crossing, so the tracer must find EVERY crossing curve, not a grid-dependent
// subset.
func TestSSIRippleFindsEveryCrossing(t *testing.T) {
	ripple := rippledSheet(t, 24, 1, 1.0, false)
	plane, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))

	want := rippleZeroCrossings(ripple, 4096)
	loops := IntersectSurfaceSurface(ripple, plane, SurfaceGrid{})
	if len(loops) != want {
		t.Fatalf("traced %d intersection curves, want the analytic count %d — a crossing was silently dropped", len(loops), want)
	}
	// Every traced point must lie on BOTH surfaces: on the plane (z=0) AND — via the knot-span-aware
	// point inversion (Piece 1) — back on the high-span ripple within the model-relative tolerance.
	res := surfaceNetResolution(ripple)
	for _, p := range allPoints(t, loops) {
		if stdmath.Abs(float64(p.Z)) > 1e-4 { // tol:numeric — on-plane check vs the trace tolerance, not a weld
			t.Errorf("traced point %v is off the z=0 plane", p)
		}
		if _, _, d := ProjectPointToSurface(ripple, p); d > res.Sew() {
			t.Errorf("traced point %v inverts %g off the ripple (> %g) — seeding recovered the wrong foot", p, d, res.Sew())
		}
	}
}

// TestRippleSSIUsesCertifiedHodographBound is the #1608 no-guess guard: on the high-span sheet the
// prune must run on the rigorous hodograph bound (never fall back to the sampled 2.0 factor), and that
// bound must be a true upper bound on the surface's tangent magnitude — otherwise a crossing could be
// pruned. It samples |S_u|,|S_v| densely and asserts the hodograph bound dominates every sample.
func TestRippleSSIUsesCertifiedHodographBound(t *testing.T) {
	ripple := rippledSheet(t, 24, 3, 1.0, true)
	plane, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	g := resolveGrid(ripple, SurfaceGrid{})
	f := newSSISeedField(ripple, plane, g)
	f.seeds(ssiStep(ripple, g))
	if f.declined {
		t.Fatal("SSI prune fell back to the sampled 2.0 guess on a B-spline surface — hodograph bound not used")
	}
	su, sv, ok := ripple.tangentBoundOverBox(0, 1, 0, 1)
	if !ok {
		t.Fatal("non-rational B-spline reported no hodograph bound")
	}
	for i := 0; i <= 200; i++ {
		du, dv := ripple.DerivativesAt(float64(i)/200, float64(i%40)/40)
		if float64(du.Length()) > su+1e-9 || float64(dv.Length()) > sv+1e-9 {
			t.Fatalf("hodograph bound (%.3f,%.3f) below actual tangent (%.3f,%.3f) — not an upper bound", su, sv, du.Length(), dv.Length())
		}
	}
}

// rippleZeroCrossings counts the sign changes of the sheet's height along u at mid-v by
// dense sampling — the independent 1D oracle for the number of plane-crossing curves (the
// ripple's z is constant in v by construction).
func rippleZeroCrossings(s BSplineSurface, samples int) int {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	vMid := (vLo + vHi) / 2
	crossings := 0
	prev := float64(s.PointAt(uLo, vMid).Z)
	for i := 1; i <= samples; i++ {
		z := float64(s.PointAt(uLo+(uHi-uLo)*float64(i)/float64(samples), vMid).Z)
		if (prev > 0) != (z > 0) {
			crossings++
		}
		prev = z
	}
	return crossings
}

// TestSSISeederBoundIsConservative checks the prune never discards a cell that brackets a crossing: the
// certified tangent bound (cellTangentBound → the sphere's closed-form hodograph bound, #1608) times
// the cell size must be at least the actual corner-to-corner field change. A direct guard on the
// Lipschitz bound that the whole approach rests on — now on the rigorous bound, not the retired guess.
func TestSSISeederBoundIsConservative(t *testing.T) {
	big, _ := NewSphere(math.P3(0, 0, 0), 1)
	other, _ := NewSphere(math.P3(1.2, 0, 0), 1)
	g := resolveGrid(big, SurfaceGrid{})
	c := newSSISeedField(big, other, g)

	res := 1 << ssiSeedMaxDepth
	a, b := c.at(0, 0), c.at(res, res) // opposite corners of one coarse cell
	su, sv := float64(res)*c.du, float64(res)*c.dv
	tu, tv := c.cellTangentBound(0, res, 0, res, a, c.at(res, 0), c.at(0, res), b)
	if bound := tu*su + tv*sv; bound < stdmath.Abs(a.f-b.f) {
		t.Errorf("variation bound %g below the actual field change %g — prune could drop a crossing", bound, stdmath.Abs(a.f-b.f))
	}
}
