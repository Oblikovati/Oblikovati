// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// s1SeamHalfSpan is OCCT's OWN seam station for the S1 fixture, read straight off the DRAWEXE control
// net: the flank patch result_8's poles run x ∈ [−6.92820323027551, −3.38093422681526] and the central
// result_3's x ∈ [−3.38093422681526, +3.38093422681526]. The fixture's spine origin puts the band
// midplane at 10, so the seam stations are 10 ∓ this. The tiling used to place them where the inner
// footprint crosses the PLAIN fillet contact line, ±4.4721 — 32% out. See
// .superpowers/sdd/runout-envelope-report.md §"Seam station".
const s1SeamHalfSpan = 3.38093422681526

// TestSeamStationMatchesOCCT pins the φ(s)=0 seam condition against DRAWEXE. It is the sharpest single
// number in the run-out derivation: the seam is where the SURF-RST contact locus on the inner host
// reaches the inner boss's footprint, not where that footprint crosses the plain fillet's contact line.
func TestSeamStationMatchesOCCT(t *testing.T) {
	tl := s1Tiling(t)
	mid := 0.5 * (tl.cutLo + tl.cutHi)
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"seamLo", tl.seamLo, mid - s1SeamHalfSpan},
		{"seamHi", tl.seamHi, mid + s1SeamHalfSpan},
	} {
		if d := stdmath.Abs(c.got - c.want); d > 1e-9 {
			t.Errorf("%s = %.12f, OCCT %.12f (off by %g)", c.name, c.got, c.want, d)
		}
	}
}

// TestSeamCornerLiesOnInnerFootprint proves the seam condition is what it claims: the surf-rst tangency
// contact at the seam sits ON the inner boss's footprint conic (φ=0), which is why the flank and the
// central band share that corner and why bSeam still doubles as the inner footprint endpoint.
func TestSeamCornerLiesOnInnerFootprint(t *testing.T) {
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

// s1FlankPatchArea / s1CentralPatchArea are DRAWEXE `sprops result_<i> 1.e-9` on the S1 oracle blend:
// result_8 and result_11 (the flanks) read 26.5949 and result_3 (the central) 34.1915. They are the
// per-face standard n4_cornerweld_layer_test.go set for a new corner green — reconcile the oracle PER
// FACE, never on whole-body area alone.
const (
	s1FlankPatchArea   = 26.5949
	s1CentralPatchArea = 34.1915
)

// TestRunoutPatchSurfaceMatchesOCCTPerFace reconciles the lofted SURFACE (integrated, not meshed — the
// mesh carries its own boundary-chording bias) against OCCT's own per-face areas. It is what makes the
// nine formerly-false greens honest: the Coons fill these bands used to get read 49.49 / 19.92 / 19.92
// against the same 34.19 / 26.59 / 26.59, i.e. +44.7% / −25.1%.
func TestRunoutPatchSurfaceMatchesOCCTPerFace(t *testing.T) {
	tl := s1Tiling(t)
	for _, c := range []struct {
		name string
		loop func() (RailLoop, bool)
		want float64
	}{
		{"left flank", tl.leftFlank, s1FlankPatchArea},
		{"central", tl.centralBand, s1CentralPatchArea},
		{"right flank", tl.rightFlank, s1FlankPatchArea},
	} {
		lp, ok := c.loop()
		if !ok {
			t.Fatalf("%s: loop did not build", c.name)
		}
		patch, ok := resolveBlend(lp, s1Resolution(t))
		if !ok {
			t.Fatalf("%s: resolveBlend declined", c.name)
		}
		if patch.Kind != BlendKindRunoutCanal {
			t.Fatalf("%s: kind %q, want %q (a run-out band must never fall back to a Coons fill)",
				c.name, patch.Kind, BlendKindRunoutCanal)
		}
		got := integrateSurfaceArea(patch.Surface.(geom.BSplineSurface))
		if rel := stdmath.Abs(got-c.want) / c.want; rel > 2e-3 {
			t.Errorf("%s: surface area %.4f vs OCCT %.4f (rel %.4f%% > 0.2%%)", c.name, got, c.want, rel*100)
		}
	}
}

// integrateSurfaceArea is a midpoint quadrature of |S_u × S_v| over the patch domain — a like-for-like
// comparison with DRAWEXE's `sprops`, which is itself a surface quadrature, and independent of the
// tessellator's boundary chording (the trap n4_cornerweld_layer_test.go documents).
func integrateSurfaceArea(s geom.BSplineSurface) float64 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	const n = 64
	du, dv := (u1-u0)/n, (v1-v0)/n
	sum := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			su, sv := s.DerivativesAt(u0+(float64(i)+0.5)*du, v0+(float64(j)+0.5)*dv)
			sum += float64(su.Cross(sv).Length()) * du * dv
		}
	}
	return sum
}

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

// s1Resolution is the fixture's model-relative resolution (ADR-0042), needed by resolveBlend.
func s1Resolution(t *testing.T) Resolution {
	t.Helper()
	_, res := runoutFixtureCrossingBoss(t)
	return res
}

// TestSurfRstCentreDeclinesNearParallelHosts is the pitfall-5 guard: when the two hosts are
// near-parallel the branch sign σ = sign(n_A·(n_B×e)) is undefined, and the solver must DECLINE rather
// than pick one — a clamped or guessed branch would place the ball on the wrong side entirely.
func TestSurfRstCentreDeclinesNearParallelHosts(t *testing.T) {
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
	cyl, err := geom.NewCylinder(math.P3(0, -12, -3), math.V3(1, 0, 0), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	env := newRunoutEnvelope(cyl)
	if _, ok := env.rstRstCentre(0, math.P3(0, 0, 0), math.P3(0, 20, 0), 1e-6); ok {
		t.Errorf("rstRstCentre accepted feet 20 apart at r=3; want an honest decline")
	}
}
