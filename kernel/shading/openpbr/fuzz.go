// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// fuzzAlbedoQuadratureN is the hemisphere quadrature resolution for fuzzScalarAlbedo —
// the Charlie lobe (below) is broad and low-frequency (unlike GGX's peak), so it converges
// with far fewer samples than multiscatter.go's GGX integrals need.
const fuzzAlbedoQuadratureN = 24

// sheenDistributionCharlie is the "Charlie" sheen normal distribution (Estevez & Kulla,
// "Production Friendly Microfacet Sheen BRDF", 2017) — this package's closed-form
// substitute for the OpenPBR spec's recommended (not mandated — "We recommend the
// specific model of [#Zeltner2022]") Fuzz model. Zeltner's model needs a tabulated
// (μo,alpha)-grid LTC fit to a volumetric microflake simulation this package does not
// vendor (the same LUT-availability constraint as multiscatter.go's Kulla-Conty choice);
// Charlie is the closed-form velvet/sheen BRDF glTF's KHR_materials_sheen extension and
// Filament ship as their production alternative.
func sheenDistributionCharlie(sinThetaH, alpha float64) float64 {
	a := math.Max(alpha, 1e-3)
	invAlpha := 1 / a
	return (2 + invAlpha) * math.Pow(sinThetaH, invAlpha) / (2 * math.Pi)
}

// visibilityNeubelt is the simple visibility term Charlie is conventionally paired with
// (Neubelt et al., "Crafting a Next-Gen Material Pipeline for The Order: 1886", 2013).
func visibilityNeubelt(cosI, cosO float64) float64 {
	return 1 / (4 * (cosI + cosO - cosI*cosO))
}

// SpecularFuzz evaluates this package's Charlie+Neubelt fuzz BRDF (spec §Fuzz, f_fuzz):
// fuzz_color tints the D·V microflake-sheen lobe directly, matching the spec's own
// f_fuzz = fuzz_color · E_fuzz(μo,α) · D(...) shape (this closed-form folds the tabulated
// E_fuzz normalization into Charlie's own, already near-normalized, D).
func SpecularFuzz(wi, wo Vec3, roughness float64, color Color3) Color3 {
	cosI, cosO := wi.CosTheta(), wo.CosTheta()
	if cosI <= 0 || cosO <= 0 {
		return Color3{}
	}
	h := wi.Add(wo).Normalize()
	sinThetaH := math.Sqrt(math.Max(0, 1-h.Z*h.Z))
	d := sheenDistributionCharlie(sinThetaH, roughness)
	v := visibilityNeubelt(cosI, cosO)
	return color.Scale(d * v)
}

// fuzzScalarAlbedo numerically integrates the (colorless) Charlie+Neubelt lobe's
// hemispherical-directional reflectance at a fixed outgoing angle — the E_fuzz(μo,α) this
// package's fuzz layering weight needs (spec eq. fuzz-layering-approx explicitly excludes
// fuzz_color from this term, "not to tint the base").
func fuzzScalarAlbedo(cosThetaO, roughness float64) float64 {
	if cosThetaO <= 0 {
		return 0
	}
	sinThetaO := math.Sqrt(1 - cosThetaO*cosThetaO)
	wo := Vec3{X: sinThetaO, Z: cosThetaO}

	var sum float64
	for i := 0; i < fuzzAlbedoQuadratureN; i++ {
		thetaI := (float64(i) + 0.5) / fuzzAlbedoQuadratureN * (math.Pi / 2)
		sinThetaI, cosThetaI := math.Sin(thetaI), math.Cos(thetaI)
		for j := 0; j < fuzzAlbedoQuadratureN; j++ {
			phiI := (float64(j) + 0.5) / fuzzAlbedoQuadratureN * (2 * math.Pi)
			wi := Vec3{X: sinThetaI * math.Cos(phiI), Y: sinThetaI * math.Sin(phiI), Z: cosThetaI}
			f := SpecularFuzz(wi, wo, roughness, Gray(1))
			sum += f.R * cosThetaI * sinThetaI
		}
	}
	dTheta := (math.Pi / 2) / fuzzAlbedoQuadratureN
	dPhi := (2 * math.Pi) / fuzzAlbedoQuadratureN
	return sum * dTheta * dPhi
}

// LayerFuzz combines fuzz with the coated-base lobe beneath it (spec eq.
// fuzz-layering-approx): weight·f_fuzz + (1 - weight·E_fuzz(wo))·coatedBase.
//
// weight<=0 returns coatedBase unchanged — the PBI-341 regression guard that
// fuzz_weight=0 reproduces the coat-only output exactly.
func LayerFuzz(wi, wo Vec3, roughness float64, color Color3, weight float64, coatedBase Color3) Color3 {
	if weight <= 0 {
		return coatedBase
	}
	fFuzz := SpecularFuzz(wi, wo, roughness, color)
	eFuzz := fuzzScalarAlbedo(wo.CosTheta(), roughness)
	return fFuzz.Scale(weight).Add(coatedBase.Scale(1 - weight*eFuzz))
}
