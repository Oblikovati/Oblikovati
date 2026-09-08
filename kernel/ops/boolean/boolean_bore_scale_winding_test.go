// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestBoreIntegratesAsRemovedAtEveryScale is the corpus row for the third half of
// Oblikovati/Oblikovati#3512: a face's loop handedness must be read in ARC LENGTH, not in raw (u, v).
//
// materialSideVotes decides which side of a boundary edge holds the face's material by stepping a
// quarter of that edge's own length off it. It took the step as a quarter TURN in (u, v), which is
// "left" only where the chart is metrically isotropic. A cylinder's u is an angle and its v a length, so
// on a 30 µm bore a rim segment's du/4 ≈ 0.05 lands far outside a chart 6e-5 tall: every rim station
// abstained and the seam stations carried the vote the wrong way. The bore wall then integrated as
// ADDED material, and the boolean's Requicha bracket refused the cut outright (2.4565e-12 against a
// V(A) of 2.4e-12). It only ever showed at µm scale because the geometric probe was overriding the
// handedness face by face, which is exactly what a per-face override hides — see
// TestThinFinWallKeepsTheSenseItsLoopsCarry, which is why it no longer does.
//
// The volume is analytic: a 2s × 2s × 0.6s slab less a π(0.3s)²·0.6s bore.
func TestBoreIntegratesAsRemovedAtEveryScale(t *testing.T) {
	for _, s := range []float64{1e-4, 1e-3, 1, 1e3} {
		drilled := drilledSlabAtScale(t, s)
		terms, ok := query.AnalyticBodyTerms(drilled)
		if !ok {
			t.Fatalf("scale %g: the drilled slab has no analytic volume", s)
		}
		want := (2 * 2 * 0.6 * s * s * s) - stdmath.Pi*(0.3*s)*(0.3*s)*(0.6*s)
		if rel := stdmath.Abs(terms.Vol-want) / want; rel > 1e-9 { // tol:calibrated — exact analytic faces
			t.Errorf("scale %g: volume = %g, want %g (rel %g) — a bore read as ADDED is the winding defect",
				s, terms.Vol, want, rel)
		}
	}
}

// drilledSlabAtScale cuts a through-bore in a slab whose extent is s, through the general pipeline.
func drilledSlabAtScale(t *testing.T, s float64) *topo.Body {
	t.Helper()
	slab, err := brep.SolidBlock(math.P3(math.Scalar(-s), math.Scalar(-s), 0),
		math.P3(math.Scalar(s), math.Scalar(s), math.Scalar(0.6*s)), "slab")
	if err != nil {
		t.Fatalf("slab at %g: %v", s, err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, math.Scalar(-0.1*s)), math.V3(0, 0, 1), 0.3*s, 0.8*s)
	if err != nil {
		t.Fatalf("rod at %g: %v", s, err)
	}
	drilled, err := Boolean(Cut, slab, rod)
	if err != nil {
		t.Fatalf("cut at %g: %v", s, err)
	}
	return drilled
}
