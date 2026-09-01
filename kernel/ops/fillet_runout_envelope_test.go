// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The S1-only seam pin that used to live here (s1SeamHalfSpan = 3.38093422681526, read off the DRAWEXE
// control net) is SUPERSEDED by TestRunoutBandStationsMatchOCCT in fillet_runout_oracle_test.go, which
// pins all four spine stations of ALL SIX two-boss cases off the oracle patch surfaces' own v-bounds —
// the same number for S1, plus the five siblings that had no station protection at all.

// TestSeamCornerLiesOnInnerFootprint proves the seam condition is what it claims: the surf-rst tangency
// contact at the seam sits ON the inner boss's footprint conic (φ=0), which is why the flank and the
// central band share that corner and why bSeam still doubles as the inner footprint endpoint.
func TestSeamCornerLiesOnInnerFootprint(t *testing.T) {
	t.Parallel()
	tl := s1Tiling(t)
	for name, p := range map[string]math.Point3{"bSeamLo": tl.bSeamLo, "bSeamHi": tl.bSeamHi} {
		phi, ok := footprintMembership(tl.inner, p)
		if !ok {
			t.Fatalf("%s: footprintMembership declined", name)
		}
		if stdmath.Abs(phi) > 1e-6 {
			t.Errorf("%s: φ = %g, want 0 (the seam corner must lie on the inner footprint)", name, phi)
		}
	}
}

// TestRunoutBandsAreTheRollingBallEnvelope is the fidelity gate the coons4 certificate could not
// provide: every station of every band has BOTH feet exactly at the ball radius from its centre — the
// algebraic statement that the lofted surface is the envelope, checked on the real S1 tiling.
func TestRunoutBandsAreTheRollingBallEnvelope(t *testing.T) {
	t.Parallel()
	tl := s1Tiling(t)
	for name, band := range map[string]*runoutBand{"left": tl.left, "central": tl.mid, "right": tl.right} {
		if band == nil {
			t.Fatalf("%s band unresolved", name)
		}
		for i, st := range band.stations {
			for side, foot := range map[string]math.Point3{"A": st.footA, "B": st.footB} {
				if d := stdmath.Abs(float64(st.centre.DistanceTo(foot)) - tl.cyl.Radius); d > 1e-9 {
					t.Errorf("%s station %d foot %s: |c−f| off radius by %g", name, i, side, d)
				}
			}
		}
	}
}

// The S1-only per-face area pins (s1FlankPatchArea 26.5949 / s1CentralPatchArea 34.1915) and
// integrateSurfaceArea now live in fillet_runout_oracle_test.go, where the same DRAWEXE `sprops` receipt
// is taken for ALL SIX two-boss cases alongside the interior-surface point pins that make the per-face
// claim load-bearing (an area-only assertion would not have caught S7 — see that file's header).

// s1Tiling resolves the real S1 fixture's setback tiling — the shared arrangement of these tests.
func s1Tiling(t *testing.T) setbackTiling {
	t.Helper()
	ef, res := runoutFixtureCrossingBoss(t)
	b, ok := detectSetbackBands(ef, res)
	if !ok {
		t.Fatalf("detectSetbackBands declined the S1 fixture")
	}
	tl, ok := resolveSetbackTiling(b, ef, res)
	if !ok {
		t.Fatalf("resolveSetbackTiling declined the S1 fixture")
	}
	return tl
}

// TestSurfRstCentreDeclinesNearParallelHosts is the pitfall-5 guard: when the two hosts are
// near-parallel the branch sign σ = sign(n_A·(n_B×e)) is undefined, and the solver must DECLINE rather
// than pick one — a clamped or guessed branch would place the ball on the wrong side entirely.
func TestSurfRstCentreDeclinesNearParallelHosts(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, -12, -3), math.V3(1, 0, 0), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	env := newRunoutEnvelope(cyl)
	front, err := geom.NewPlane(math.P3(0, -15, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	// A "restrict" host PARALLEL to the tangency host: n_A·(n_B×e) collapses to 0.
	parallel, err := geom.NewPlane(math.P3(0, -9, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	if _, ok := env.surfRstCentre(front, parallel, 0, math.P3(0, -13, 0), 1e-6); ok {
		t.Errorf("surfRstCentre accepted a near-parallel host pair; want an honest decline")
	}
}

// TestSurfRstCentreDeclinesUnreachableRestriction is the pitfall-1 guard: a restriction point farther
// than r from the offset line L_B admits NO radius-r ball, and the discriminant must not be clamped to
// zero (which would fabricate a tangency and silently ship a wrong surface).
func TestSurfRstCentreDeclinesUnreachableRestriction(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, -12, -3), math.V3(1, 0, 0), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	env := newRunoutEnvelope(cyl)
	front, _ := geom.NewPlane(math.P3(0, -15, 0), math.V3(0, 1, 0))
	top, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	// q is 30 from the offset line, ten times the ball radius.
	if _, ok := env.surfRstCentre(front, top, 0, math.P3(0, -42, 0), 1e-6); ok {
		t.Errorf("surfRstCentre accepted an unreachable restriction point; want an honest decline")
	}
}

// TestRstRstCentreDeclinesTooFarApart is the pitfall-2 guard: two restriction points more than 2r apart
// have no common radius-r ball.
func TestRstRstCentreDeclinesTooFarApart(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, -12, -3), math.V3(1, 0, 0), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	env := newRunoutEnvelope(cyl)
	if _, ok := env.rstRstCentre(0, math.P3(0, 0, 0), math.P3(0, 20, 0), 1e-6); ok {
		t.Errorf("rstRstCentre accepted feet 20 apart at r=3; want an honest decline")
	}
}
