// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	m "oblikovati.org/math"
)

// fakeCornerProvider is a scripted CornerBlendProvider double: it returns exactly the fits/build
// verdicts the test sets, so resolveCornerBlend's tier walking, certificate gating, and ordering can
// be exercised without any real surface fitting.
type fakeCornerProvider struct {
	name  CornerBlendKind
	fits  bool
	build bool
	cert  Certificate
	patch CornerBlendPatch
}

func (f fakeCornerProvider) Name() CornerBlendKind        { return f.name }
func (f fakeCornerProvider) Fits(CornerBlendRequest) bool { return f.fits }
func (f fakeCornerProvider) Build(CornerBlendRequest) (CornerBlendPatch, Certificate, bool) {
	return f.patch, f.cert, f.build
}

// blendScale is a stable model scale whose weld tolerance sizes the certificate thresholds.
func blendScale() Resolution {
	return ResolutionForPoints([]m.Point3{m.P3(0, 0, 0), m.P3(10, 0, 0)})
}

// validCert is a certificate comfortably inside every tolerance at weld w.
func validCert(w float64) Certificate {
	return Certificate{Closed: true, WeldsArms: true, NoFold: true, MaxDev: w / 2, MaxAngleDev: seamAngularTol / 2}
}

func provider(kind CornerBlendKind, cert Certificate) fakeCornerProvider {
	return fakeCornerProvider{name: kind, fits: true, build: true, cert: cert, patch: CornerBlendPatch{Kind: kind}}
}

// TestResolveCornerBlendDeclines pins the honest-reject floor: with no tier, a tier that doesn't fit,
// or a tier that fits but fails to build, resolveCornerBlend returns ok=false (ADR-3).
func TestResolveCornerBlendDeclines(t *testing.T) {
	t.Parallel()
	req := CornerBlendRequest{Setback: blendScale()}
	cases := map[string][]CornerBlendProvider{
		"no tiers":    nil,
		"none fit":    {fakeCornerProvider{fits: false}},
		"build fails": {fakeCornerProvider{fits: true, build: false}},
	}
	for name, tiers := range cases {
		if _, ok := resolveCornerBlend(req, tiers); ok {
			t.Errorf("%s: expected decline, got a patch", name)
		}
	}
}

// TestResolveCornerBlendReturnsValid pins that a fitting provider with a valid certificate wins.
func TestResolveCornerBlendReturnsValid(t *testing.T) {
	t.Parallel()
	scale := blendScale()
	req := CornerBlendRequest{Setback: scale}
	patch, ok := resolveCornerBlend(req, []CornerBlendProvider{provider(BlendKindBSpline, validCert(scale.Weld()))})
	if !ok || patch.Kind != BlendKindBSpline {
		t.Fatalf("expected the bspline patch, got ok=%v kind=%q", ok, patch.Kind)
	}
}

// TestResolveCornerBlendSkipsInvalidCertificate pins that a G1-kinked patch (MaxAngleDev over
// seamAngularTol) is passed over and the next valid tier is taken — the anti-crease guarantee.
func TestResolveCornerBlendSkipsInvalidCertificate(t *testing.T) {
	t.Parallel()
	scale := blendScale()
	req := CornerBlendRequest{Setback: scale}
	kinked := validCert(scale.Weld())
	kinked.MaxAngleDev = seamAngularTol * 10 // tangent kink: shades as a crease
	tiers := []CornerBlendProvider{provider("kinked", kinked), provider(BlendKindBSpline, validCert(scale.Weld()))}
	patch, ok := resolveCornerBlend(req, tiers)
	if !ok || patch.Kind != BlendKindBSpline {
		t.Fatalf("expected the kinked patch skipped and bspline taken, got ok=%v kind=%q", ok, patch.Kind)
	}
}

// TestResolveCornerBlendTierOrderWins pins ADR-2: an analytic provider ranked ahead of the fallback
// wins for its family purely by tier order — the classifier IS the ordering, no switch.
func TestResolveCornerBlendTierOrderWins(t *testing.T) {
	t.Parallel()
	scale := blendScale()
	req := CornerBlendRequest{Setback: scale}
	tiers := []CornerBlendProvider{provider("analytic-torus", validCert(scale.Weld())), provider(BlendKindBSpline, validCert(scale.Weld()))}
	patch, ok := resolveCornerBlend(req, tiers)
	if !ok || patch.Kind != "analytic-torus" {
		t.Fatalf("earlier tier must win, got ok=%v kind=%q", ok, patch.Kind)
	}
}

// TestCertificateValid pins each admissibility gate: the full certificate passes, and violating any
// single field (structural or numeric) fails it.
func TestCertificateValid(t *testing.T) {
	t.Parallel()
	scale := blendScale()
	w := scale.Weld()
	if !validCert(w).Valid(scale) {
		t.Fatal("a fully-within-tolerance certificate must be valid")
	}
	mut := map[string]func(*Certificate){
		"not closed":      func(c *Certificate) { c.Closed = false },
		"arm not welded":  func(c *Certificate) { c.WeldsArms = false },
		"folded":          func(c *Certificate) { c.NoFold = false },
		"G0 over weld":    func(c *Certificate) { c.MaxDev = w * 2 },
		"G1 tangent kink": func(c *Certificate) { c.MaxAngleDev = seamAngularTol * 2 },
	}
	for name, violate := range mut {
		c := validCert(w)
		violate(&c)
		if c.Valid(scale) {
			t.Errorf("%s: certificate must be invalid", name)
		}
	}
}
