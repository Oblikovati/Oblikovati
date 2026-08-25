// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "testing"

// TestEnergyConservationSweepCoatAndFuzz extends PBI-340's energy-conservation sweep
// (specular_test.go) to coat+fuzz combinations, per PBI-341's explicit acceptance
// criterion: the integrated hemispherical reflectance of a diffuse base layered under a
// coat, and again under both coat and fuzz, must not exceed 1.
func TestEnergyConservationSweepCoatAndFuzz(t *testing.T) {
	const cosThetaO = 0.6
	const tolerance = 1.05 // quadrature slack across three stacked numerical integrals

	roughnesses := []float64{0.2, 0.5, 0.8}
	coatWeights := []float64{0.5, 1.0}
	fuzzWeights := []float64{0.5, 1.0}

	for _, roughness := range roughnesses {
		for _, coatWeight := range coatWeights {
			t.Run("diffuse+coat", func(t *testing.T) {
				e := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 {
					fSub := DiffuseEON(Gray(0.8), roughness, wi, wo)
					fCoat := SpecularCoat(wi, wo, 0.1, 1.6)
					darkening := CoatDarkeningFactor(0.8, coatWeight, 1, 1.6)
					return LayerCoat(fCoat, fSub, Gray(1), coatWeight, darkening, wo, 1.6).R
				}, cosThetaO)
				if e > tolerance {
					t.Errorf("roughness=%v coatWeight=%v: diffuse+coat reflectance = %v, want ≤ 1", roughness, coatWeight, e)
				}
			})
			for _, fuzzWeight := range fuzzWeights {
				t.Run("diffuse+coat+fuzz", func(t *testing.T) {
					e := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 {
						fSub := DiffuseEON(Gray(0.8), roughness, wi, wo)
						fCoat := SpecularCoat(wi, wo, 0.1, 1.6)
						darkening := CoatDarkeningFactor(0.8, coatWeight, 1, 1.6)
						coated := LayerCoat(fCoat, fSub, Gray(1), coatWeight, darkening, wo, 1.6)
						return LayerFuzz(wi, wo, 0.5, Gray(1), fuzzWeight, coated).R
					}, cosThetaO)
					if e > tolerance {
						t.Errorf("roughness=%v coatWeight=%v fuzzWeight=%v: diffuse+coat+fuzz reflectance = %v, want ≤ 1",
							roughness, coatWeight, fuzzWeight, e)
					}
				})
			}
		}
	}
}
