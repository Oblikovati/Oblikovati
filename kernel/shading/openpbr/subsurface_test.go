// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestMixSubsurfaceZeroWeightReproducesDiffuseExactly is PBI-343's explicit regression
// guard: subsurface_weight=0 must reproduce PBI-342's (diffuse) output exactly.
func TestMixSubsurfaceZeroWeightReproducesDiffuseExactly(t *testing.T) {
	diffuse := NewColor3(0.31, 0.22, 0.05)
	got := MixSubsurface(diffuse, Gray(0.9), 0)
	if got != diffuse {
		t.Errorf("MixSubsurface(weight=0) = %+v, want diffuse unchanged = %+v", got, diffuse)
	}
}

// TestSubsurfaceExtinctionIsReciprocalMFP checks the hand-derivable identity (spec:
// "the reciprocal of the MFP per channel").
func TestSubsurfaceExtinctionIsReciprocalMFP(t *testing.T) {
	got := SubsurfaceExtinction(2, Color3{R: 1, G: 0.5, B: 0.25})
	want := Color3{R: 0.5, G: 1, B: 2}
	if math.Abs(got.R-want.R) > 1e-9 || math.Abs(got.G-want.G) > 1e-9 || math.Abs(got.B-want.B) > 1e-9 {
		t.Errorf("SubsurfaceExtinction(radius=2, scale=(1,0.5,0.25)) = %+v, want %+v", got, want)
	}
}

// TestSubsurfaceExtinctionRegularizesZeroMFP checks the zero-MFP guard produces a large,
// finite extinction (opaque-diffuse limit) rather than +Inf/NaN.
func TestSubsurfaceExtinctionRegularizesZeroMFP(t *testing.T) {
	got := SubsurfaceExtinction(0, Gray(1))
	if math.IsInf(got.R, 1) || math.IsNaN(got.R) {
		t.Errorf("SubsurfaceExtinction(radius=0) = %v, want a large finite value", got.R)
	}
}

// TestVanDeHulstRoundTrip is PBI-343's core diffusion-profile oracle: the spec defines
// subsurfaceAlbedoFromColorChannel as the explicit inversion of
// subsurfaceColorFromAlbedoChannel (its own forward van de Hulst formula) — composing
// color→albedo→color must be the identity, for every anisotropy and color in range.
func TestVanDeHulstRoundTrip(t *testing.T) {
	for _, g := range []float64{-0.5, 0, 0.3, 0.7} {
		for _, c := range []float64{0.05, 0.3, 0.5, 0.8, 0.95} {
			alpha := subsurfaceAlbedoFromColorChannel(c, g)
			gotC := subsurfaceColorFromAlbedoChannel(alpha, g)
			if math.Abs(gotC-c) > 1e-3 {
				t.Errorf("g=%v c=%v: round trip color→α(%v)→color = %v, want ≈ %v", g, c, alpha, gotC, c)
			}
		}
	}
}

// TestSubsurfaceSingleScatterAlbedoWhiteApproachesOne checks the spec's stated furnace
// property (index.html line 730): a white subsurface_color (C=1) must yield a
// single-scattering albedo close to 1 (near-conservative scattering), matching the
// "energy always conserved... white passes the furnace test" guarantee.
func TestSubsurfaceSingleScatterAlbedoWhiteApproachesOne(t *testing.T) {
	got := SubsurfaceSingleScatterAlbedo(Gray(1), 0)
	if got.R < 0.99 {
		t.Errorf("SubsurfaceSingleScatterAlbedo(color=1) = %v, want close to 1", got.R)
	}
}

// TestPhaseHenyeyGreensteinIsotropicAtZeroAnisotropy checks the closed-form reduction:
// g=0 must give the exactly isotropic 1/(4π) phase function at every angle.
func TestPhaseHenyeyGreensteinIsotropicAtZeroAnisotropy(t *testing.T) {
	want := 1 / (4 * math.Pi)
	for _, cosTheta := range []float64{-1, -0.3, 0, 0.5, 1} {
		if got := PhaseHenyeyGreenstein(cosTheta, 0); math.Abs(got-want) > 1e-9 {
			t.Errorf("PhaseHenyeyGreenstein(cos=%v, g=0) = %v, want %v", cosTheta, got, want)
		}
	}
}

// TestPhaseHenyeyGreensteinIntegratesToOne checks the phase function's defining
// normalization property (∫ p(cosθ) dω = 1 over the full sphere) via numerical
// integration, for a sweep of anisotropy values.
func TestPhaseHenyeyGreensteinIntegratesToOne(t *testing.T) {
	for _, g := range []float64{-0.7, -0.3, 0, 0.3, 0.7} {
		const n = 256
		var sum float64
		for i := range n {
			cosTheta := -1 + (float64(i)+0.5)/n*2
			sum += PhaseHenyeyGreenstein(cosTheta, g) * 2 * math.Pi
		}
		integral := sum * (2.0 / n)
		if math.Abs(integral-1) > 0.01 {
			t.Errorf("g=%v: ∫ HG dω = %v, want ≈ 1", g, integral)
		}
	}
}

// TestPhaseHenyeyGreensteinForwardScatteringPeak checks the qualitative shape at g>0:
// forward scattering (cosTheta=1, same direction as travel) must be more probable than
// backscattering (cosTheta=-1).
func TestPhaseHenyeyGreensteinForwardScatteringPeak(t *testing.T) {
	forward := PhaseHenyeyGreenstein(1, 0.7)
	back := PhaseHenyeyGreenstein(-1, 0.7)
	if forward <= back {
		t.Errorf("g=0.7: forward=%v, back=%v, want forward > back", forward, back)
	}
}

// TestResolveSubsurfaceMediumBundlesTheResolvedParameters is a smoke test that the
// bundling function wires its three resolvers correctly.
func TestResolveSubsurfaceMediumBundlesTheResolvedParameters(t *testing.T) {
	m := ResolveSubsurfaceMedium(NewColor3(0.8, 0.5, 0.3), 1, Color3{R: 1, G: 0.5, B: 0.25}, 0.2)
	if m.Anisotropy != 0.2 {
		t.Errorf("Anisotropy = %v, want 0.2", m.Anisotropy)
	}
	wantExtinction := SubsurfaceExtinction(1, Color3{R: 1, G: 0.5, B: 0.25})
	if m.Extinction != wantExtinction {
		t.Errorf("Extinction = %+v, want %+v", m.Extinction, wantExtinction)
	}
	wantAlbedo := SubsurfaceSingleScatterAlbedo(NewColor3(0.8, 0.5, 0.3), 0.2)
	if m.Albedo != wantAlbedo {
		t.Errorf("Albedo = %+v, want %+v", m.Albedo, wantAlbedo)
	}
}
