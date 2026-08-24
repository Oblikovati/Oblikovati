// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// microfacetSingleScatter evaluates the standard single-scattering microfacet BRDF form
// (spec eq. microfacet_brdf_ss, [#Walter2007]/[#Pharr2023]): F(wi,h)·D(h)·G(wi,wo) /
// (4·|wi·n|·|wo·n|). fresnel is the already-evaluated Fresnel term (scalar for dielectric,
// per-channel for conductor — callers multiply it in per channel).
func microfacetSingleScatter(wi, wo Vec3, alpha float64) (d, g, cosI, cosO float64) {
	cosI, cosO = wi.CosTheta(), wo.CosTheta()
	if cosI <= 0 || cosO <= 0 {
		return 0, 0, cosI, cosO
	}
	h := wi.Add(wo).Normalize()
	return DistributionGGX(h, alpha), SmithG2(wi, wo, alpha), cosI, cosO
}

// SpecularDielectric evaluates the OpenPBR Surface Specular lobe's reflection off the
// base dielectric interface (spec §Specular, f_dielectric): rough GGX with the exact
// Fresnel term, plus Kulla-Conty multi-scatter compensation (multiscatter.go) so a
// specular_roughness=1 surface does not visibly darken relative to a smooth one
// (the "white furnace test", spec line 425).
//
// wi/wo are in local shading space; ior is the relative IOR (specular_ior, already
// specular_weight-scaled per the spec's "apply specular weight to IOR" step, which is a
// caller concern — this function only evaluates the GGX+Fresnel math).
func SpecularDielectric(wi, wo Vec3, roughness, ior float64) float64 {
	alpha := AlphaFromRoughness(roughness)
	d, g, cosI, cosO := microfacetSingleScatter(wi, wo, alpha)
	if d == 0 {
		return 0
	}
	h := wi.Add(wo).Normalize()
	f := DielectricFresnel(ior, math.Abs(wi.Dot(h)))
	ss := f * d * g / (4 * cosI * cosO)
	return ss + dielectricMultiScatter(alpha, ior, cosI, cosO)
}

// SpecularConductor evaluates the OpenPBR Surface metal base lobe (spec §Metal,
// f_conductor): rough GGX with the F82-tint conductor Fresnel, plus Kulla-Conty
// multi-scatter compensation.
//
// f0 is the normal-incidence reflectivity (base_weight*base_color) and tint is the
// grazing-edge tint (specular_color); specular_weight scales the whole result per the
// spec's F_metal = specular_weight * F82(µ) (spec eq. F_metal), applied by the caller
// scaling f0/tint or the returned color — this function evaluates the unscaled F82-tint
// lobe.
func SpecularConductor(wi, wo Vec3, roughness float64, f0, tint Color3) Color3 {
	alpha := AlphaFromRoughness(roughness)
	d, g, cosI, cosO := microfacetSingleScatter(wi, wo, alpha)
	if d == 0 {
		return Color3{}
	}
	h := wi.Add(wo).Normalize()
	f := F82TintFresnel(f0, tint, math.Abs(wi.Dot(h)))
	ss := f.Scale(d * g / (4 * cosI * cosO))
	return ss.Add(conductorMultiScatter(alpha, f0, tint, cosI, cosO))
}
