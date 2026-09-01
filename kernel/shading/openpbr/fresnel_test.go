// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestDielectricFresnelNormalIncidenceMatchesF0 checks the hand-derivable identity: at
// normal incidence the exact Fresnel equations reduce to F0 = ((eta-1)/(eta+1))².
func TestDielectricFresnelNormalIncidenceMatchesF0(t *testing.T) {
	t.Parallel()
	const ior = 1.5
	got := DielectricFresnel(ior, 1)
	want := F0FromIOR(ior)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("DielectricFresnel(1.5, cos=1) = %v, want F0 = %v", got, want)
	}
}

// TestDielectricFresnelGrazingIsOne checks total reflection at exactly grazing incidence.
func TestDielectricFresnelGrazingIsOne(t *testing.T) {
	t.Parallel()
	if got := DielectricFresnel(1.5, 0); math.Abs(got-1) > 1e-9 {
		t.Errorf("DielectricFresnel(1.5, cos=0) = %v, want 1", got)
	}
}

// TestDielectricFresnelIndexMatchedIsZero checks the eta=1 short-circuit.
func TestDielectricFresnelIndexMatchedIsZero(t *testing.T) {
	t.Parallel()
	if got := DielectricFresnel(1, 0.5); got != 0 {
		t.Errorf("DielectricFresnel(1, cos=0.5) = %v, want 0 (index-matched)", got)
	}
}

// TestF82TintFresnelNormalIncidenceIsF0 checks the analytic limit: at cosTheta=1 the
// (1-cosTheta)^5 correction vanishes, so F82TintFresnel must equal f0 exactly regardless
// of tint.
func TestF82TintFresnelNormalIncidenceIsF0(t *testing.T) {
	t.Parallel()
	f0 := NewColor3(0.9, 0.6, 0.2)
	tint := NewColor3(0.3, 0.8, 0.95)
	got := F82TintFresnel(f0, tint, 1)
	if math.Abs(got.R-f0.R) > 1e-9 || math.Abs(got.G-f0.G) > 1e-9 || math.Abs(got.B-f0.B) > 1e-9 {
		t.Errorf("F82TintFresnel(f0, tint, cos=1) = %+v, want f0 = %+v", got, f0)
	}
}

// TestF82TintFresnelMatchesTintAtReferenceAngle checks the spec's own defining property
// (index.html eq. F_82, "ensuring F82(μ̄) = F(μ̄)"): at the 82° reference angle, the
// F82-tint curve must equal tint * Schlick(82°) exactly.
func TestF82TintFresnelMatchesTintAtReferenceAngle(t *testing.T) {
	t.Parallel()
	f0 := NewColor3(0.9, 0.6, 0.2)
	tint := NewColor3(0.3, 0.8, 0.95)
	got := F82TintFresnel(f0, tint, f82CosThetaMax)

	schlick := func(r float64) float64 {
		return r + (1-r)*math.Pow(1-f82CosThetaMax, 5)
	}
	want := Color3{R: tint.R * schlick(f0.R), G: tint.G * schlick(f0.G), B: tint.B * schlick(f0.B)}
	if math.Abs(got.R-want.R) > 1e-6 || math.Abs(got.G-want.G) > 1e-6 || math.Abs(got.B-want.B) > 1e-6 {
		t.Errorf("F82TintFresnel at reference angle = %+v, want tint*Schlick(82°) = %+v", got, want)
	}
}

// TestF82TintFresnelDefaultTintReducesToSchlick checks the spec's other stated property
// (index.html line 558): a white (default) tint reduces the model exactly to the plain
// Schlick curve at every angle.
func TestF82TintFresnelDefaultTintReducesToSchlick(t *testing.T) {
	t.Parallel()
	f0 := NewColor3(0.9, 0.6, 0.2)
	white := Gray(1)
	for _, cos := range []float64{1, 0.7, f82CosThetaMax, 0.05} {
		got := F82TintFresnel(f0, white, cos)
		schlick := func(r float64) float64 { return r + (1-r)*math.Pow(1-cos, 5) }
		want := Color3{R: schlick(f0.R), G: schlick(f0.G), B: schlick(f0.B)}
		if math.Abs(got.R-want.R) > 1e-6 || math.Abs(got.G-want.G) > 1e-6 || math.Abs(got.B-want.B) > 1e-6 {
			t.Errorf("cos=%v: F82TintFresnel(f0, white, cos) = %+v, want Schlick = %+v", cos, got, want)
		}
	}
}
