// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// Quadrature resolution for the hemisphere integrals below. This package runs only in
// CPU-reference tests, never per-pixel, so a modest fixed grid trades a little precision
// for keeping test runtime negligible; nothing here is on the F04 rendering hot path
// (see the package doc's Kulla-Conty note).
const (
	albedoQuadratureTheta = 64
	albedoQuadraturePhi   = 64
	averageQuadratureN    = 16
)

// minAlphaForMultiScatter is this package's own numerical-resolution floor: below it, the
// GGX peak (angular width ~alpha) is narrower than the fixed hemisphere quadrature's grid
// spacing, so DirectionalAlbedoGGX/AverageAlbedoGGX become unreliable (measured: alpha=
// 0.01 overshoots the physical [0,1] bound by ~8%; alpha≥0.04 stays within ~0.2%).
// Adobe's reference has an analogous cutoff (impl/openpbr_microfacet_multiple_scattering_
// lobes.h's OpenPBR_MinAlphaWithVisibleEnergyLoss, "empirically derived") for the same
// physical reason — a smooth surface loses little energy to interreflection — but at a
// much smaller value their baked, adaptively-sampled LUT can resolve accurately and this
// package's brute-force quadrature cannot. Below this floor, single-scatter alone is
// already an accurate result (multi-scatter loss is negligible at low roughness), so
// skipping compensation costs no real energy conservation, only importance-sampling-grade
// convergence — a LUT-backed replacement (porting Adobe's data/openpbr_energy_arrays.h)
// is the natural follow-up if that precision is ever needed on the hot path.
const minAlphaForMultiScatter = 0.04

// DirectionalAlbedoGGX numerically integrates the single-scattering GGX BRDF's
// hemispherical-directional reflectance E(wo) = ∫ f(wi,wo)·cosθi dωi, with Fresnel≡1 (the
// energy lost to D·G alone, independent of material color) — the quantity the Kulla-Conty
// multi-scatter compensation (below) is built from. cosThetaO is the outgoing polar
// cosine; alpha is AlphaFromRoughness(roughness).
func DirectionalAlbedoGGX(cosThetaO, alpha float64) float64 {
	if cosThetaO <= 0 {
		return 0
	}
	sinThetaO := math.Sqrt(1 - cosThetaO*cosThetaO)
	wo := Vec3{X: sinThetaO, Z: cosThetaO}

	var sum float64
	for i := 0; i < albedoQuadratureTheta; i++ {
		thetaI := (float64(i) + 0.5) / albedoQuadratureTheta * (math.Pi / 2)
		sinThetaI, cosThetaI := math.Sin(thetaI), math.Cos(thetaI)
		for j := 0; j < albedoQuadraturePhi; j++ {
			phiI := (float64(j) + 0.5) / albedoQuadraturePhi * (2 * math.Pi)
			wi := Vec3{X: sinThetaI * math.Cos(phiI), Y: sinThetaI * math.Sin(phiI), Z: cosThetaI}
			h := wi.Add(wo).Normalize()
			sum += DistributionGGX(h, alpha) * SmithG2(wi, wo, alpha) * sinThetaI
		}
	}
	dTheta := (math.Pi / 2) / albedoQuadratureTheta
	dPhi := (2 * math.Pi) / albedoQuadraturePhi
	// The cosθi factor in the reflectance integral cancels the 1/cosθi in the microfacet
	// BRDF's single-scatter denominator, leaving only 1/(4·cosθo).
	return sum * dTheta * dPhi / (4 * cosThetaO)
}

// AverageAlbedoGGX is the cosine-weighted hemispherical average of [DirectionalAlbedoGGX],
// ⟨E⟩ = 2∫₀¹ E(μ)·μ dμ — the same weighting the EON diffuse compensation uses for its own
// average albedo (diffuse.go), so a fully diffuse (rho=1) and fully specular (F=1) surface
// both reach the same white-furnace-test guarantee.
func AverageAlbedoGGX(alpha float64) float64 {
	var sum float64
	for i := 0; i < averageQuadratureN; i++ {
		mu := (float64(i) + 0.5) / averageQuadratureN
		sum += DirectionalAlbedoGGX(mu, alpha) * mu
	}
	return 2 * sum / averageQuadratureN
}

// kullaContyCompensation is the shared shape of the Kulla-Conty/Turquin-style
// energy-compensation term: given the average Fresnel reflectance fAvg, the average
// directional albedo eAvg, and the two directions' own directional albedos eI/eO, it
// returns the extra reflectance that restores energy the single-scatter lobe lost to
// microfacet interreflection. Structurally identical to diffuse.go's EON compensation
// (rho↔fAvg, the Oren-Nayar directional albedo↔the GGX one) — both are the same
// microfacet multi-scatter correction applied to different single-scatter kernels.
func kullaContyCompensation(fAvg, eAvg, eI, eO float64) float64 {
	const eps = 1e-7
	denom := math.Pi * math.Max(eps, 1-eAvg) * math.Max(eps, 1-fAvg*(1-eAvg))
	return fAvg * fAvg * eAvg * math.Max(eps, 1-eO) * math.Max(eps, 1-eI) / denom
}

// dielectricMultiScatter is [SpecularDielectric]'s Kulla-Conty compensation term. Zero
// below minAlphaForMultiScatter (see its doc).
func dielectricMultiScatter(alpha, ior, cosI, cosO float64) float64 {
	if alpha < minAlphaForMultiScatter {
		return 0
	}
	eAvg := AverageAlbedoGGX(alpha)
	fAvg := DielectricAverageFresnel(ior)
	return kullaContyCompensation(fAvg, eAvg, DirectionalAlbedoGGX(cosI, alpha), DirectionalAlbedoGGX(cosO, alpha))
}

// conductorMultiScatter is [SpecularConductor]'s Kulla-Conty compensation term, tinted
// per-channel by the F82-tint average Fresnel. Zero below minAlphaForMultiScatter.
func conductorMultiScatter(alpha float64, f0, tint Color3, cosI, cosO float64) Color3 {
	if alpha < minAlphaForMultiScatter {
		return Color3{}
	}
	eAvg := AverageAlbedoGGX(alpha)
	eI, eO := DirectionalAlbedoGGX(cosI, alpha), DirectionalAlbedoGGX(cosO, alpha)
	fAvg := F82TintAverageFresnel(f0, tint)
	return Color3{
		R: kullaContyCompensation(fAvg.R, eAvg, eI, eO),
		G: kullaContyCompensation(fAvg.G, eAvg, eI, eO),
		B: kullaContyCompensation(fAvg.B, eAvg, eI, eO),
	}
}
