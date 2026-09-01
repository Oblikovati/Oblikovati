// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The curved half-space pipeline must give the SAME RELATIVE result at every model scale
// (Oblikovati/Oblikovati#1399): the analytic imprint that decides where the cutting plane meets a
// face now derives its clearance tolerance from the body's own extent (geom.ResolutionForBox of
// the RangeBox) instead of a cm-anchored epsilon. A sphere cut through its centre yields a
// hemisphere cap of volume (2/3)πR³, so the dimensionless coefficient vol/R³ is invariant across
// a unit, a km and a Mm copy — the issue's "boolean and mesh a unit / 1e6× copy, assert identical
// volume within relative tolerance" case, run end-to-end through brep.HalfSpaceCut.
func TestHalfSpaceCutSphereScaleSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4s): `make test-corpus`")
	}
	t.Parallel()
	const hemisphereCoef = 2.0 / 3.0 * stdmath.Pi // (2/3)π
	const relTol = 0.02                           // tessellated cap vs analytic, scale-independent
	for _, radius := range []float64{1, 1e3, 1e6} {
		sphere, err := brep.SolidSphere(math.P3(0, 0, 0), radius, "s")
		if err != nil {
			t.Fatalf("SolidSphere(r=%g): %v", radius, err)
		}
		plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
		if err != nil {
			t.Fatalf("NewPlane: %v", err)
		}
		cap, err := brep.HalfSpaceCut(sphere, plane)
		if err != nil {
			t.Fatalf("HalfSpaceCut(r=%g): %v", radius, err)
		}
		coef := vol(cap) / (radius * radius * radius)
		if relErr := stdmath.Abs(coef-hemisphereCoef) / hemisphereCoef; relErr > relTol {
			t.Errorf("radius %g: hemisphere cap coefficient %.5f, want %.5f (rel err %.4f > %.4f) — "+
				"half-space cut is not scale-faithful", radius, coef, hemisphereCoef, relErr, relTol)
		}
	}
}
