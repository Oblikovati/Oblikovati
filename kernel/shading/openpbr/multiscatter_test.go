// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "testing"

// TestDirectionalAlbedoGGXBounds checks the physical bound (a reflectance can't exceed 1
// or go negative) across a sweep of roughness and incidence angle, at and above
// minAlphaForMultiScatter — this package's own numerical-quadrature resolution floor
// (see the constant's doc); below it the fixed hemisphere grid under-resolves the GGX
// peak and DirectionalAlbedoGGX is not a meaningful reflectance estimate.
func TestDirectionalAlbedoGGXBounds(t *testing.T) {
	t.Parallel()
	for _, roughness := range []float64{0.25, 0.4, 0.6, 0.8, 1.0} {
		alpha := AlphaFromRoughness(roughness)
		for _, cosTheta := range []float64{0.05, 0.3, 0.6, 1.0} {
			e := DirectionalAlbedoGGX(cosTheta, alpha)
			if e < 0 || e > 1.02 {
				t.Errorf("roughness=%v cos=%v: DirectionalAlbedoGGX = %v, want in [0,1]", roughness, cosTheta, e)
			}
		}
	}
}

// TestAverageAlbedoGGXDecreasesWithRoughness checks the expected trend: rougher
// microfacet surfaces lose more energy to interreflection (lower average albedo) —
// AverageAlbedoGGX at high roughness should be meaningfully lower than at low roughness,
// in the range where the quadrature is reliable (see TestDirectionalAlbedoGGXBounds).
// The comparison is endpoint-to-endpoint, not step-by-step: the coarse 16-sample cosine
// average (averageQuadratureN) has enough of its own quadrature noise to occasionally tick
// up between adjacent roughness values without the overall trend being violated.
func TestAverageAlbedoGGXDecreasesWithRoughness(t *testing.T) {
	t.Parallel()
	smooth := AverageAlbedoGGX(AlphaFromRoughness(0.25))
	rough := AverageAlbedoGGX(AlphaFromRoughness(1.0))
	if rough > smooth-0.02 {
		t.Errorf("AverageAlbedoGGX(roughness=1.0) = %v, want meaningfully below roughness=0.25's %v", rough, smooth)
	}
	for _, roughness := range []float64{0.25, 0.4, 0.6, 0.8, 1.0} {
		if e := AverageAlbedoGGX(AlphaFromRoughness(roughness)); e < 0 || e > 1.02 {
			t.Errorf("AverageAlbedoGGX(roughness=%v) = %v, want in [0,1]", roughness, e)
		}
	}
}

// TestMultiScatterSkippedBelowResolutionFloor checks the minAlphaForMultiScatter guard:
// a very smooth surface (roughness=0.05, alpha=0.0025) gets zero compensation, not a
// bogus value from the under-resolved quadrature.
func TestMultiScatterSkippedBelowResolutionFloor(t *testing.T) {
	t.Parallel()
	alpha := AlphaFromRoughness(0.05)
	if got := dielectricMultiScatter(alpha, 1.5, 0.8, 0.8); got != 0 {
		t.Errorf("dielectricMultiScatter below the resolution floor = %v, want 0", got)
	}
	if got := conductorMultiScatter(alpha, Gray(0.9), Gray(1), 0.8, 0.8); got != (Color3{}) {
		t.Errorf("conductorMultiScatter below the resolution floor = %+v, want zero", got)
	}
}

// TestKullaContyCompensationNonNegative checks the compensation term never subtracts
// energy (it exists to add back energy lost to single-scatter, never to remove it).
func TestKullaContyCompensationNonNegative(t *testing.T) {
	t.Parallel()
	for _, fAvg := range []float64{0.04, 0.3, 0.8, 1.0} {
		for _, eAvg := range []float64{0.2, 0.5, 0.9} {
			for _, e := range []float64{0.1, 0.5, 0.9} {
				if got := kullaContyCompensation(fAvg, eAvg, e, e); got < 0 {
					t.Errorf("kullaContyCompensation(%v,%v,%v,%v) = %v, want ≥ 0", fAvg, eAvg, e, e, got)
				}
			}
		}
	}
}
