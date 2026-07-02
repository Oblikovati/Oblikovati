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

// TestSSISeederBoundIsConservative checks the prune never discards a cell that brackets a crossing: the
// field-variation bound must be at least the actual corner-to-corner field change. (A direct guard on
// the Lipschitz bound that the whole approach rests on.)
func TestSSISeederBoundIsConservative(t *testing.T) {
	big, _ := NewSphere(math.P3(0, 0, 0), 1)
	other, _ := NewSphere(math.P3(1.2, 0, 0), 1)
	g := resolveGrid(big, SurfaceGrid{})
	c := newSSISeedField(big, other, g)

	res := 1 << ssiSeedMaxDepth
	a, b := c.at(0, 0), c.at(res, res) // opposite corners of one coarse cell
	su, sv := float64(res)*c.du, float64(res)*c.dv
	bound := ssiSeedSafety * (max(a.tu, b.tu, a.tu, b.tu)*su + max(a.tv, b.tv, a.tv, b.tv)*sv)
	if change := stdmath.Abs(a.f - b.f); bound < change {
		t.Errorf("variation bound %g below the actual field change %g — prune could drop a crossing", bound, change)
	}
}
