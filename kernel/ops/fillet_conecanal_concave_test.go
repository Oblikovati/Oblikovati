// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A2 wave: the CONCAVE-BORE cone-canal spine (apexSign=−1, coneCanalSpine.apexSign) — re-landed from a
// reverted session (wave-round4-armlayer.md), derived independently from first principles (ball tangent
// to the cone from the OUTWARD normal instead of the boss's INWARD normal) and cross-checked by
// reproducing the shipped CONVEX ζ(x_f) formula bit-for-bit from that same construction. simple/I4's
// ruling edge is the real-corpus exercise: coneHostMaterialSign reads s<0 there (material outside the
// cone — a bore), which coneCanalArmFillet used to flat-decline as "a follow-on slice" before this fix.

// TestConeCanalSpine_ConcaveBore_ExactFeet mirrors TestConeCanalSpine_ExactFeet with apexSign=−1 on the
// SAME synthetic cone/plane fixtures: the ball centre still sits on the r-offset plane, the hyperbola
// relation still holds (with the −r/sinα apex-shift term, per assertOnHyperbola's apexSign-aware form),
// and both feet still sit at exactly radius r from the centre — proving the concave construction is just
// as exact as the shipped convex one, not merely "declines less often".
func TestConeCanalSpine_ConcaveBore_ExactFeet(t *testing.T) {
	t.Parallel()
	for _, c := range coneCanalCases() {
		t.Run(c.name, func(t *testing.T) {
			co, err := geom.NewCone(c.apex, coneAxisDown(), stdmath.Atan(c.tanAlpha))
			if err != nil {
				t.Fatalf("%s cone: %v", c.name, err)
			}
			nOut, err := math.UnitVector3FromVector(c.nOut)
			if err != nil {
				t.Fatalf("%s nOut: %v", c.name, err)
			}
			spine, reason := newConeCanalSpine(co, nOut, -1, coneArmR, ResolutionForSize(300))
			if reason != coneArmBuilt {
				t.Fatalf("%s newConeCanalSpine(apexSign=-1) declined a valid spine (reason %d)", c.name, reason)
			}
			for i := 0; i <= 12; i++ {
				xf := c.xfLo + (c.xfHi-c.xfLo)*float64(i)/12
				assertStationExact(t, c.name, spine, co, xf)
			}
		})
	}
}

// findConcaveConeRulingEdge is findConeRulingEdge narrowed to the CONCAVE-BORE ruling specifically (a
// body can carry both a convex and a concave ruling edge; I4's actual corpus pick is the concave one,
// which findConeRulingEdge's first-match scan does not guarantee to return).
func findConcaveConeRulingEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	res := ResolutionForBody(body)
	for _, e := range body.Edges() {
		co, pl, coneFace, _, ok := conePlaneEdge(e)
		if !ok || classifyConeArm(co, pl, coneRadiusAt(co, edgeMidpoint(e)), res) != coneClassRuling {
			continue
		}
		if sgn, ok := coneHostMaterialSign(e, co, coneFace); ok && sgn <= 0 {
			return e
		}
	}
	t.Fatal("no concave-bore Cone∧Plane ruling edge found in the imported body")
	return nil
}

// TestConeCanalArm_I4RealImport builds I4's real concave-bore ruling arm end to end through
// coneArmEdge (the same entry point the fillet feature uses) and asserts the loft is exact at every
// adaptively-chosen station and meshes fold-free — the concave-bore sibling of
// TestConeCanalArm_RealImport (C2/C6/D1, convex-external).
func TestConeCanalArm_I4RealImport(t *testing.T) {
	t.Parallel()
	body := importSimpleFixture(t, "I4")
	e := findConcaveConeRulingEdge(t, body)
	ef, handled, err := coneArmEdge(body, e, filletPick{edge: e, r0: coneArmR, r1: coneArmR})
	if !handled || err != nil || ef.armCanalSpine == nil {
		t.Fatalf("I4 ruling edge: want built concave canal arm, got handled=%v err=%v spine=%v", handled, err, ef.armCanalSpine)
	}
	if ef.armCanalSpine.apexSign != -1 {
		t.Fatalf("I4 spine apexSign = %g, want -1 (concave bore)", ef.armCanalSpine.apexSign)
	}
	surf, ok := ef.armSurface.(geom.BSplineSurface)
	if !ok {
		t.Fatalf("I4 canal arm is %T, want geom.BSplineSurface", ef.armSurface)
	}
	res := ResolutionForBody(body)
	assertLoftExactAtStations(t, "I4", *ef.armCanalSpine, e, res)
	assertArmMeshesFoldFree(t, "I4", surf)
}

// TestConeCanalSpine_ZetaSignFlips is a direct, non-round-trip unit check that apexSign actually flips
// the ζ apex-shift term (the ONE place the derivation says it must, per coneCanalSpine.apexSign's
// doc): at x_f=0 (ρ=r), ζ(0) = apexSign·r/sinα + r/tanα. Fixing r, sinα, tanα, the convex (+1) and
// concave (−1) ζ(0) must differ by exactly 2·r/sinα — proving the field is load-bearing on the
// formula, not a dead parameter that happens to cancel out.
func TestConeCanalSpine_ZetaSignFlips(t *testing.T) {
	t.Parallel()
	co, err := geom.NewCone(math.P3(0, 0, 270), coneAxisDown(), stdmath.Atan(1.0/3.0))
	if err != nil {
		t.Fatalf("cone: %v", err)
	}
	nOut, err := math.UnitVector3FromVector(math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("nOut: %v", err)
	}
	res := ResolutionForSize(300)
	convex, reason := newConeCanalSpine(co, nOut, +1, coneArmR, res)
	if reason != coneArmBuilt {
		t.Fatalf("convex spine declined (reason %d)", reason)
	}
	concave, reason := newConeCanalSpine(co, nOut, -1, coneArmR, res)
	if reason != coneArmBuilt {
		t.Fatalf("concave spine declined (reason %d)", reason)
	}
	want := 2 * coneArmR / convex.sinA
	got := convex.zetaAt(0) - concave.zetaAt(0)
	if d := stdmath.Abs(got - want); d > coneCanalExactTol {
		t.Fatalf("zetaAt(0) convex-concave gap = %g, want exactly 2r/sinα = %g (off by %g) — apexSign is not flipping the ζ term", got, want, d)
	}
}

// TestConeCanalArm_DispatchesBothSigns is the dispatch-level mutation witness: coneArmEdge (the real
// entry point) must read coneHostMaterialSign and route a KNOWN-convex real fixture (C2) to
// apexSign=+1 and a KNOWN-concave one (I4) to apexSign=−1 — BOTH directions of the `if sgn <= 0`
// branch in coneCanalArmFillet. A flipped or dropped condition would send one of the two cases through
// the wrong offset formula; this test catches either direction, unlike a same-sign-only check.
func TestConeCanalArm_DispatchesBothSigns(t *testing.T) {
	t.Parallel()
	cases := []struct {
		fixture      string
		findEdge     func(*testing.T, *topo.Body) *topo.Edge
		wantSign     float64
		wantApexSign float64
	}{
		{"C2", findConeRulingEdge, +1, +1},
		{"I4", findConcaveConeRulingEdge, -1, -1},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			body := importSimpleFixture(t, c.fixture)
			e := c.findEdge(t, body)
			co, _, coneFace, _, ok := conePlaneEdge(e)
			if !ok {
				t.Fatalf("%s: not a Cone∧Plane edge", c.fixture)
			}
			sgn, ok := coneHostMaterialSign(e, co, coneFace)
			if !ok || (sgn > 0) != (c.wantSign > 0) {
				t.Fatalf("%s: coneHostMaterialSign = %v (ok=%v), want sign matching %v", c.fixture, sgn, ok, c.wantSign)
			}
			ef, handled, err := coneArmEdge(body, e, filletPick{edge: e, r0: coneArmR, r1: coneArmR})
			if !handled || err != nil || ef.armCanalSpine == nil {
				t.Fatalf("%s: want built canal arm, got handled=%v err=%v spine=%v", c.fixture, handled, err, ef.armCanalSpine)
			}
			if ef.armCanalSpine.apexSign != c.wantApexSign {
				t.Fatalf("%s: dispatched apexSign = %g, want %g", c.fixture, ef.armCanalSpine.apexSign, c.wantApexSign)
			}
		})
	}
}
