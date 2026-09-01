// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestSpecularDielectricMirrorReflectsExactly checks the roughness=0 canonical case: the
// GGX lobe collapses toward a delta function at the mirror direction, but away from it
// (wi != reflect(wo)) the single-scatter term must be exactly zero (h != macronormal, so
// D→0 in the limit) — checked at a direction well off the mirror reflection.
func TestSpecularDielectricOffMirrorDirectionIsNegligible(t *testing.T) {
	t.Parallel()
	wo := Vec3{Z: 1}
	// wi far from the mirror direction (which is also straight up when wo is straight up).
	wi := Vec3{X: 0.9, Z: math.Sqrt(1 - 0.81)}
	got := SpecularDielectric(wi, wo, 0.01, 1.5)
	if got > 1e-3 {
		t.Errorf("SpecularDielectric(off-mirror, roughness=0.01) = %v, want ≈ 0", got)
	}
}

// TestSpecularConductorZeroAtGrazingCosine checks the microfacetSingleScatter guard: a
// direction with cosTheta<=0 (below the surface) must contribute nothing.
func TestSpecularConductorZeroBelowSurface(t *testing.T) {
	t.Parallel()
	wo := Vec3{Z: -1} // below the surface
	wi := Vec3{Z: 1}
	got := SpecularConductor(wi, wo, 0.3, Gray(0.9), Gray(1))
	if got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("SpecularConductor(wo below surface) = %+v, want zero", got)
	}
}

// TestEnergyConservationSweep is PBI-340's explicit acceptance criterion: the integrated
// hemispherical reflectance of every base lobe must not exceed 1, across a sweep of
// roughness/metalness-adjacent parameter values, at a representative view angle.
func TestEnergyConservationSweep(t *testing.T) {
	t.Parallel()
	const cosThetaO = 0.6
	const tolerance = 1.03 // numerical-quadrature slack, not a physical allowance

	for _, roughness := range []float64{0.1, 0.3, 0.5, 0.7, 0.9} {
		t.Run("diffuse", func(t *testing.T) {
			e := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 {
				return DiffuseEON(Gray(1), roughness, wi, wo).R
			}, cosThetaO)
			if e > tolerance {
				t.Errorf("roughness=%v: diffuse hemispherical reflectance = %v, want ≤ 1", roughness, e)
			}
		})
		t.Run("dielectric", func(t *testing.T) {
			e := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 {
				return SpecularDielectric(wi, wo, roughness, 1.5)
			}, cosThetaO)
			if e > tolerance {
				t.Errorf("roughness=%v: dielectric hemispherical reflectance = %v, want ≤ 1", roughness, e)
			}
		})
		t.Run("conductor_white", func(t *testing.T) {
			// A white metal (f0=1) is the most energy-demanding case: single-scatter alone
			// already reflects ~100% near normal, so the compensation term must not push it
			// over the top.
			e := hemisphericalReflectanceColor(func(wi, wo Vec3) Color3 {
				return SpecularConductor(wi, wo, roughness, Gray(1), Gray(1))
			}, cosThetaO)
			if e > tolerance {
				t.Errorf("roughness=%v: white-conductor hemispherical reflectance = %v, want ≤ 1", roughness, e)
			}
		})
	}
}
