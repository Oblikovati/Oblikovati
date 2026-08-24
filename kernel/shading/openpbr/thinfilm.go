// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"math/cmplx"
)

// rgbWavelengthsNM are the fixed representative R/G/B wavelengths (nanometers) this
// package evaluates thin-film interference at — the spec's own recommended default for
// "typical RGB-based production rendering" (index.html §Thin-film-iridescence, contrasted
// there with the optional per-path stochastic-spectral alternative for spectral
// renderers, which this package does not implement). Values are the conventional sRGB
// primary dominant wavelengths used throughout the thin-film literature (Belcour & Barla
// 2017 and its production ports).
var rgbWavelengthsNM = Color3{R: 611, G: 549, B: 466}

// minDenom guards the same near-zero denominators Adobe's reference guards (their
// OpenPBR_MinDenomMag), for the same reason: grazing incidence and TIR boundaries produce
// exact zeros in these formulas' denominators.
const minDenom = 1e-9

// snellCosComplex is the unified Snell's-law cosine, ported from Adobe's
// openpbr_snell_cos_unified: works for a real (dielectric) or complex (conductor) etaT,
// producing a complex cos(theta_t) under total internal reflection so phase is preserved.
func snellCosComplex(cosThetaI, etaI float64, etaT complex128) complex128 {
	sinThetaI := math.Sqrt(clamp(1-cosThetaI*cosThetaI, 0, 1))
	etaIOverEtaT := complex(etaI, 0) / etaT
	sinThetaT := complex(sinThetaI, 0) * etaIOverEtaT
	cosThetaTSq := complex(1, 0) - sinThetaT*sinThetaT
	cosThetaT := cmplx.Sqrt(cosThetaTSq)
	// Enforce the physical (decaying, not growing) branch: Im(etaT·cosThetaT) >= 0.
	if imag(etaT*cosThetaT) < 0 {
		cosThetaT = -cosThetaT
	}
	return cosThetaT
}

// fresnelAmplitudeComplex is the unified complex Fresnel reflection amplitude (s/p),
// ported from Adobe's openpbr_compute_fresnel_unified_polarized_reflection_amplitude.
// etaT may be complex (a conductor's n+ik) or real (a dielectric, imaginary part 0).
func fresnelAmplitudeComplex(cosThetaI, etaI float64, etaT complex128) (rs, rp complex128) {
	cosThetaT := snellCosComplex(cosThetaI, etaI, etaT)
	etaICosThetaI := complex(etaI*cosThetaI, 0)
	etaTCosThetaT := etaT * cosThetaT
	etaTCosThetaI := etaT * complex(cosThetaI, 0)
	etaICosThetaT := complex(etaI, 0) * cosThetaT

	rs = safeComplexDivide(etaICosThetaI-etaTCosThetaT, etaICosThetaI+etaTCosThetaT)
	rp = safeComplexDivide(etaTCosThetaI-etaICosThetaT, etaTCosThetaI+etaICosThetaT)
	return rs, rp
}

func safeComplexDivide(numer, denom complex128) complex128 {
	if cmplx.Abs(denom) < minDenom {
		return 1
	}
	return numer / denom
}

// fresnelAmplitudeDielectric is the real-valued fast path for an exterior-to-film (or
// film-to-exterior) dielectric interface, ported from Adobe's
// openpbr_compute_fresnel_dielectric_polarized_amplitude. Returns the s/p amplitude
// reflection AND transmission coefficients, plus cos(theta_t) (0 under TIR).
func fresnelAmplitudeDielectric(cosThetaI, etaI, etaT float64) (rs, rp, ts, tp, cosThetaT float64) {
	sinThetaI := math.Sqrt(clamp(1-cosThetaI*cosThetaI, 0, 1))
	sinThetaT := etaI / etaT * sinThetaI
	if sinThetaT >= 1 {
		return 1, 1, 0, 0, 0
	}
	cosThetaT = math.Sqrt(clamp(1-sinThetaT*sinThetaT, 0, 1))

	etaICosThetaI := etaI * cosThetaI
	etaTCosThetaT := etaT * cosThetaT
	etaTCosThetaI := etaT * cosThetaI
	etaICosThetaT := etaI * cosThetaT

	denomS := safeDenom(etaICosThetaI + etaTCosThetaT)
	denomP := safeDenom(etaTCosThetaI + etaICosThetaT)

	rs = (etaICosThetaI - etaTCosThetaT) / denomS
	rp = (etaTCosThetaI - etaICosThetaT) / denomP
	ts = (2 * etaICosThetaI) / denomS
	tp = (2 * etaICosThetaI) / denomP
	return rs, rp, ts, tp, cosThetaT
}

func safeDenom(d float64) float64 {
	if math.Abs(d) < minDenom {
		if d >= 0 {
			return minDenom
		}
		return -minDenom
	}
	return d
}

// airyReflectance sums the infinite geometric series of internal reflections inside the
// film (Belcour & Barla 2017, eq. 3) into a single closed-form total reflectance for one
// polarization — ported from Adobe's openpbr_compute_airy_reflectance.
func airyReflectance(r12, t12, r21, t21 float64, r23, expIDeltaPhi complex128) float64 {
	r23Exp := r23 * expIDeltaPhi
	numerator := complex(t12*t21, 0) * r23Exp
	denominator := complex(1, 0) - complex(r21, 0)*r23Exp
	rTotal := complex(r12, 0) + safeComplexDivide(numerator, denominator)
	m := cmplx.Abs(rTotal)
	return m * m
}

// thinFilmPresenceMultiplier ramps the film out below ~30nm (Adobe's
// openpbr_thin_film_presence_multiplier): a film thinner than that is visually
// indistinguishable from absent, and the Airy formula's opd/deltaPhi terms become
// numerically degenerate as thickness→0.
func thinFilmPresenceMultiplier(thicknessNM float64) float64 {
	const fullInvisibleNM = minDenom
	const fullVisibleNM = 30.0
	return smoothstep(fullInvisibleNM, fullVisibleNM, thicknessNM)
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

// ThinFilmReflectanceDielectric evaluates the OpenPBR thin-film interference term (spec
// §Thin-film-iridescence, "embedded in the Fresnel term, not a separate slab") over a
// dielectric base, via Airy summation at [rgbWavelengthsNM] — a direct port of Adobe's
// openpbr_thin_film_and_base_reflectance's dielectric-base path (the metal-base path,
// which needs Gulbrandsen n/k reconstruction from base_color/specular_color, is a
// documented follow-up: this package's SpecularConductor doesn't yet expose a complex IOR
// for it to feed).
//
// cosThetaI is the incidence cosine (≥0); etaExterior/etaFilm/etaBase are relative IORs
// (ambient=1 by convention); thicknessMicrons is thin_film_thickness (spec unit, µm).
// Returns the unpolarized power reflectance per RGB channel, already ramped by
// [thinFilmPresenceMultiplier] for very thin films.
func ThinFilmReflectanceDielectric(cosThetaI, etaExterior, etaFilm, etaBase, thicknessMicrons float64) Color3 {
	if thicknessMicrons <= 0 {
		return Color3{}
	}
	thicknessNM := thicknessMicrons * 1000
	presence := thinFilmPresenceMultiplier(thicknessNM)

	r12s, r12p, t12s, t12p, cosThetaTFilm := fresnelAmplitudeDielectric(cosThetaI, etaExterior, etaFilm)
	if cosThetaTFilm <= 0 {
		// Total internal reflection on the outside of the film.
		return Gray(presence)
	}
	r21s, r21p := -r12s, -r12p
	safeCosI := math.Max(cosThetaI, minDenom)
	t21Scale := (etaFilm * cosThetaTFilm) / (etaExterior * safeCosI)
	t21s, t21p := t12s*t21Scale, t12p*t21Scale

	opd := 2 * etaFilm * thicknessNM * cosThetaTFilm
	deltaPhi := func(lambdaNM float64) float64 { return 2 * math.Pi * opd / lambdaNM }

	r23s, r23p := fresnelAmplitudeComplex(cosThetaTFilm, etaFilm, complex(etaBase, 0))

	reflectance := func(lambdaNM float64) float64 {
		expI := cmplx.Exp(complex(0, deltaPhi(lambdaNM)))
		rs := airyReflectance(r12s, t12s, r21s, t21s, r23s, expI)
		rp := airyReflectance(r12p, t12p, r21p, t21p, r23p, expI)
		return 0.5 * (rs + rp)
	}

	return Color3{
		R: presence * reflectance(rgbWavelengthsNM.R),
		G: presence * reflectance(rgbWavelengthsNM.G),
		B: presence * reflectance(rgbWavelengthsNM.B),
	}
}

// FresnelWithThinFilm blends the thin-film-modulated Fresnel reflectance over the plain
// dielectric Fresnel term by thin_film_weight — the spec's "embedded in the Fresnel term,
// not a separate slab" integration point (§Thin-film-iridescence). etaExterior is fixed at
// 1 (ambient) here; a nested-dielectric caller (PBI-344) composes its own ambient-IOR
// stack on top of this.
//
// weight<=0 returns [DielectricFresnel] broadcast to all channels, unchanged — the
// PBI-342 regression guard that thin_film_weight=0 reproduces the plain (PBI-341) Fresnel
// term exactly.
func FresnelWithThinFilm(cosThetaI, ior, filmIOR, thicknessMicrons, weight float64) Color3 {
	plain := Gray(DielectricFresnel(ior, cosThetaI))
	if weight <= 0 {
		return plain
	}
	film := ThinFilmReflectanceDielectric(cosThetaI, 1, filmIOR, ior, thicknessMicrons)
	return lerpColor(plain, film, weight)
}
