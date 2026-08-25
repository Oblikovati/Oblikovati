// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// SpecularCoat evaluates the OpenPBR Surface Coat lobe's own GGX dielectric reflection
// (spec §Coat, f_coat) — the spec's Microfacet model section says the coat interface
// shares the same GGX+exact-Fresnel form as the base dielectric, so this is
// [SpecularDielectric] evaluated at the coat's own roughness/IOR (coat_roughness,
// coat_ior — distinct parameters from the base's specular_roughness/specular_ior).
func SpecularCoat(wi, wo Vec3, roughness, ior float64) float64 {
	return SpecularDielectric(wi, wo, roughness, ior)
}

// CoatDarkeningFactor returns the spec's "modulated darkening factor" (index.html eq.
// modulated_darkening_factor): lerp(1, Δ, weight*coatDarkening), the multiplier the coated
// base's BSDF is scaled by to account for extra absorption from internal reflections
// inside the coat. baseAlbedoNormal is the base lobe's own normal-incidence directional
// albedo (E_b in the spec).
//
// Δ always uses the Lambertian/rough-base K_r form (spec eq.
// internal_diffuse_reflection_coefficient_for_rough_base), not the full smooth/rough blend
// by base_metalness and specular Fresnel (eq. ...for_general_base): this package's base
// lobes are evaluated at nonzero roughness in practice, making K_r the representative
// case, and the smooth-base K_s refinement needs the same per-direction Fresnel machinery
// SpecularDielectric already has available to a caller that wants to blend it in later.
func CoatDarkeningFactor(baseAlbedoNormal, weight, coatDarkening, ior float64) float64 {
	if weight <= 0 || coatDarkening <= 0 {
		return 1
	}
	fAvg := DielectricAverageFresnel(ior)
	k := 1 - (1-fAvg)/(ior*ior)
	delta := (1 - k) / math.Max(1e-4, 1-baseAlbedoNormal*k)
	return lerp(1, delta, weight*coatDarkening)
}

// LayerCoat combines the coat's own reflection with the lobe beneath it (spec §Coat,
// generalizing the same non-reciprocal albedo-scaling pattern used for glossy-diffuse,
// eq. non-reciprocal-albedo-scaling): f_layer = f_coat + (1-E_coat(wo))·darkening·fSub·T_coat.
//
// E_coat(wo) is approximated by the coat's own Fresnel reflectance at wo — the fraction of
// energy the coat interface itself claims before anything can reach (and return from) the
// base — rather than a full hemisphere-integrated GGX directional albedo; this keeps the
// coat layer well-defined at coat_roughness=0 (the common case, spec default) without a
// hemisphere quadrature. T_coat = sqrt(coatColor): coat_color is defined as the SQUARE of
// the coat medium's normal-incidence transmittance (index.html line 920), i.e. the
// round-trip (in+out) transmittance; the per-pass factor applied once here is its square
// root. darkening is [CoatDarkeningFactor]'s output (1 = no darkening applied).
//
// weight<=0 returns fSub unchanged — the PBI-341 regression guard that coat_weight=0
// reproduces the base lobes' output exactly.
func LayerCoat(fCoat float64, fSub, coatColor Color3, weight, darkening float64, wo Vec3, ior float64) Color3 {
	if weight <= 0 {
		return fSub
	}
	eCoat := DielectricFresnel(ior, math.Max(wo.CosTheta(), 0))
	tCoat := Color3{R: math.Sqrt(coatColor.R), G: math.Sqrt(coatColor.G), B: math.Sqrt(coatColor.B)}
	layered := fSub.Mul(tCoat).Scale((1 - eCoat) * darkening).Add(Gray(fCoat))
	return lerpColor(fSub, layered, weight)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func lerpColor(a, b Color3, t float64) Color3 {
	return Color3{R: lerp(a.R, b.R, t), G: lerp(a.G, b.G, t), B: lerp(a.B, b.B, t)}
}
