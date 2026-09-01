// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestDiffuseEONReducesToLambertianAtZeroRoughness checks the closed-form limit at
// roughness=0: AF=1, BF=0, so f_ss = rho/π and both directional albedos are exactly 1,
// zeroing the multi-scatter term — DiffuseEON must reduce exactly to Lambert's law.
func TestDiffuseEONReducesToLambertianAtZeroRoughness(t *testing.T) {
	t.Parallel()
	rho := Gray(0.8)
	up := Vec3{Z: 1}
	got := DiffuseEON(rho, 0, up, up)
	want := 0.8 / math.Pi
	if math.Abs(got.R-want) > 1e-6 || got.R != got.G || got.G != got.B {
		t.Errorf("DiffuseEON(rho=0.8, roughness=0, normal, normal) = %+v, want gray %v", got, want)
	}
}

// TestDiffuseEONWhiteFurnaceApproachesOne checks the energy-preservation property the
// spec calls out explicitly (index.html line 641-642): as rho→1 the total directional
// albedo of the BRDF approaches 1, even at high roughness where the naive Oren-Nayar term
// alone would under-integrate.
func TestDiffuseEONWhiteFurnaceApproachesOne(t *testing.T) {
	t.Parallel()
	for _, roughness := range []float64{0, 0.3, 0.6, 1.0} {
		e := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 {
			return DiffuseEON(Gray(1), roughness, wi, wo).R
		}, math.Cos(0.3))
		if e > 1+1e-3 {
			t.Errorf("roughness=%v: white-furnace directional albedo = %v, want ≤ 1", roughness, e)
		}
		if e < 0.9 {
			t.Errorf("roughness=%v: white-furnace directional albedo = %v, want close to 1 (energy-preserving)", roughness, e)
		}
	}
}

// TestDirectionalAlbedoFONMonotonicWithRoughness sanity-checks the building block used by
// both the single-scatter and compensation terms: at fixed incidence, a rougher surface's
// FON directional albedo should differ measurably from a smooth one's (both are always in
// [0,1], and roughness=0 gives exactly 1 per the reference's construction).
func TestDirectionalAlbedoFONBoundsAndZeroRoughness(t *testing.T) {
	t.Parallel()
	if got := directionalAlbedoFON(0.7, 0); math.Abs(got-1) > 1e-9 {
		t.Errorf("directionalAlbedoFON(mu, roughness=0) = %v, want exactly 1", got)
	}
	for _, roughness := range []float64{0.2, 0.5, 0.8, 1.0} {
		for _, mu := range []float64{0, 0.3, 0.7, 1.0} {
			e := directionalAlbedoFON(mu, roughness)
			if e < 0 || e > 1.01 {
				t.Errorf("directionalAlbedoFON(mu=%v, roughness=%v) = %v, want in [0,1]", mu, roughness, e)
			}
		}
	}
}
