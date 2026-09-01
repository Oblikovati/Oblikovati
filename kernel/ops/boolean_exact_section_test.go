// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// The ruled∩quadric intersection is exact — along each straight ruling of a cylinder or cone the other
// quadric reduces to a quadratic in the axial parameter, and its roots ARE the section — so the faces
// it bounds integrate analytically instead of declining on a marched boundary's residual
// (Oblikovati/Oblikovati#3489). These pin the whole chain on the crossing-cylinder drill: exact edges,
// a body that stitches into ONE shell, and areas/volumes matching closed forms independent of the
// engine.

// TestCrossingCylinderCutIsExactToTheEllipticOracle drives the drill through the public boolean and
// holds the result to closed forms. Before the exact section this body's edges were marched polylines
// with a 4.478e-4 achieved tolerance, the analytic integrator declined on the closure residual they
// produced, and the tessellated fallback measured the body ~3% off; a corpus regime gate at 5% cannot
// see that regression, so the bar here is the oracle itself.
func TestCrossingCylinderCutIsExactToTheEllipticOracle(t *testing.T) {
	t.Parallel()
	const bigR, drillR, height = 3.0, 1.5, 12.0
	fat, drill := crossingCylinders(t)
	res, err := ops.Boolean(ops.Cut, fat, drill)
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	if shells := len(res.Shells()); shells != 1 {
		t.Fatalf("the drilled body has %d shells, want 1: the two walls did not weld on the shared section loops", shells)
	}

	gp, ok := ops.AnalyticGeometryProperties(res)
	if !ok {
		t.Fatal("the drilled body declined analytic integration; its section edges should now be exact")
	}
	// The window integrand and its Simpson oracle live with TestDrilledWallAreaSubtractsItsWindows.
	wantWall := 2*math.Pi*bigR*height - 2*crossingCylinderWindowArea(bigR, drillR)
	gotWall, ok := ops.AnalyticFaceArea(widestCylinderFace(res, bigR))
	if !ok {
		t.Fatal("the drilled wall declined analytic area")
	}
	if rel := math.Abs(gotWall-wantWall) / wantWall; rel > 1e-6 {
		t.Errorf("drilled wall area %.9f, want %.9f (rel %.3e): the section boundary is not exact", gotWall, wantWall, rel)
	}
	// V = π·R²·h − (bore of the drill through the solid): the complement of the intersect, so pin it
	// through the volumes' additivity instead — cut + intersect must reassemble the whole cylinder.
	inter, err := ops.Boolean(ops.Intersect, fat, drill)
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	ip, ok := ops.AnalyticGeometryProperties(inter)
	if !ok {
		t.Fatal("the intersect body declined analytic integration")
	}
	whole := math.Pi * bigR * bigR * height
	if rel := math.Abs(gp.Volume+ip.Volume-whole) / whole; rel > 1e-6 {
		t.Errorf("cut %.9f + intersect %.9f = %.9f, want the whole cylinder %.9f (rel %.3e)",
			gp.Volume, ip.Volume, gp.Volume+ip.Volume, whole, rel)
	}
}
