// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

// fakeRailProvider is a scripted railProvider double: it returns exactly the fits/build verdicts
// and certificate the test sets, so resolveBlendWith's tier walking, certificate gating, and
// ordering can be exercised without any real surface fitting (mirrors fakeCornerProvider).
type fakeRailProvider struct {
	kind  CornerBlendKind
	fits  bool
	build bool
	cert  Certificate
}

func (f fakeRailProvider) Name() CornerBlendKind { return f.kind }
func (f fakeRailProvider) Fits(RailLoop) bool    { return f.fits }
func (f fakeRailProvider) Build(RailLoop, Resolution) (CornerBlendPatch, Certificate, bool) {
	return CornerBlendPatch{Kind: f.kind}, f.cert, f.build
}

// TestResolveBlendOrderingEarlierWins pins the tier order: the first tier that both Fits and Builds
// a certificate-Valid patch wins, even when later tiers also fit. A tier that Fits+Builds but whose
// certificate FAILS Valid does not win — the walk continues past it to the next valid tier (mirrors
// resolveCornerBlend's anti-crease guarantee).
func TestResolveBlendOrderingEarlierWins(t *testing.T) {
	t.Parallel()
	scale := blendScale()
	weld := scale.Weld()
	tiers := []railProvider{
		fakeRailProvider{kind: "declines", fits: false},
		fakeRailProvider{kind: "kindA", fits: true, build: true, cert: validCert(weld)},
		fakeRailProvider{kind: "kindB", fits: true, build: true, cert: validCert(weld)},
	}
	patch, ok := resolveBlendWith(RailLoop{}, scale, tiers)
	if !ok || patch.Kind != "kindA" {
		t.Fatalf("expected earlier fit+valid tier kindA to win, got ok=%v kind=%q", ok, patch.Kind)
	}

	invalidThenValid := []railProvider{
		fakeRailProvider{kind: "kindA", fits: true, build: true, cert: Certificate{}}, // fits+builds, invalid cert
		fakeRailProvider{kind: "kindB", fits: true, build: true, cert: validCert(weld)},
	}
	patch, ok = resolveBlendWith(RailLoop{}, scale, invalidThenValid)
	if !ok || patch.Kind != "kindB" {
		t.Fatalf("expected the walk to skip the invalid-certificate tier and take kindB, got ok=%v kind=%q", ok, patch.Kind)
	}
}

// TestResolveBlendHonestReject pins the honest-reject floor (ADR-3): with no tier, a tier that
// doesn't fit, a tier that fits but fails to build, or a tier that fits+builds an invalid
// certificate, resolveBlendWith returns ok=false.
func TestResolveBlendHonestReject(t *testing.T) {
	t.Parallel()
	scale := blendScale()
	cases := map[string][]railProvider{
		"nil tiers":        nil,
		"none fit":         {fakeRailProvider{fits: false}},
		"fit but no build": {fakeRailProvider{fits: true, build: false}},
		"fit+build but invalid cert": {
			fakeRailProvider{fits: true, build: true, cert: Certificate{}},
		},
	}
	for name, tiers := range cases {
		if _, ok := resolveBlendWith(RailLoop{}, scale, tiers); ok {
			t.Errorf("%s: expected decline, got a patch", name)
		}
	}
}

// TestResolveBlendTiersOrder pins the ADR-0051 foundation tier order: analytic-known-part first
// (exact sphere), then the M6' canal tier (Canal-payload-marked loops only), then the U4-4b
// exact-station canal loft (Stations-payload-marked dual-host CORE panels only), then the SETBACK-CLOSE
// run-out envelope (Runout-payload-marked bands only), then the general fills (4-sided coons4, 3-sided
// tri3). See TestCanalProviderFits (corner_provider_canal_test.go) and the U4 core tests for the
// payload-specific coverage. Supersedes the plate tier this slot held before ADR-C3.
func TestResolveBlendTiersOrder(t *testing.T) {
	t.Parallel()
	tiers := blendTiers()
	if len(tiers) != 6 {
		t.Fatalf("expected 6 tiers, got %d", len(tiers))
	}
	want := []CornerBlendKind{BlendKindSphere, BlendKindCanal, BlendKindCanalStation,
		BlendKindRunoutCanal, BlendKindCoons4, BlendKindTri3}
	for i, k := range want {
		if tiers[i].Name() != k {
			t.Errorf("tier %d: expected %q, got %q", i, k, tiers[i].Name())
		}
	}
}

// TestResolveBlendSphereWins pins that a 3-sided loop that COULD reach tri3 is claimed by the exact
// analytic sphere first, since the sphere tier is tried before tri3 (ADR-2 promotion-by-ordering).
func TestResolveBlendSphereWins(t *testing.T) {
	t.Parallel()
	patch, ok := resolveBlend(sphereTriLoop(t, 4), blendScale())
	if !ok || patch.Kind != BlendKindSphere {
		t.Fatalf("expected the analytic sphere to win, got ok=%v kind=%q", ok, patch.Kind)
	}
}

// TestResolveBlendCoons4 pins that a Valence-4 loop (the sphere provider declines: it needs
// concentric same-radius arcs on every side) falls through to the general coons4 fill.
func TestResolveBlendCoons4(t *testing.T) {
	t.Parallel()
	patch, ok := resolveBlend(quarterCylLoop(t, 8), blendScale())
	if !ok || patch.Kind != BlendKindCoons4 {
		t.Fatalf("expected coons4 to win, got ok=%v kind=%q", ok, patch.Kind)
	}
}

// TestResolveBlendRealHonestReject pins the honest-reject floor against the REAL tiers: a
// Valence-2 loop fits none of them (sphere needs >=3 arcs, coons4 needs Valence 4, tri3 needs
// Valence 3), so resolveBlend declines rather than fabricating a patch.
func TestResolveBlendRealHonestReject(t *testing.T) {
	t.Parallel()
	loop := RailLoop{Sides: []Side{{}, {}}}
	if _, ok := resolveBlend(loop, blendScale()); ok {
		t.Fatal("expected a Valence-2 loop to be honest-rejected by every real tier")
	}
}
