// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// F0FromIOR converts a relative IOR (eta_t/eta_i) to normal-incidence reflectance F0,
// exact port of Adobe's openpbr_f0_from_ior.
func F0FromIOR(iorRatio float64) float64 {
	return sq((iorRatio - 1) / (iorRatio + 1))
}

// DielectricFresnel is the exact (non-Schlick) unpolarized Fresnel reflectance for a
// dielectric interface of relative IOR iorRatio = eta_t/eta_i, at incidence cosine
// cosThetaI ≥ 0 — exact port of Adobe's openpbr_fresnel. Total internal reflection
// (sin²θt ≥ 1) returns 1.
func DielectricFresnel(iorRatio, cosThetaI float64) float64 {
	if iorRatio == 1 {
		return 0
	}
	sinThetaISq := 1 - sq(cosThetaI)
	sinThetaTSq := sinThetaISq / sq(iorRatio)
	if sinThetaTSq >= 1 {
		return 1
	}
	cosThetaT := math.Sqrt(1 - sinThetaTSq)
	rs := sq((cosThetaI - iorRatio*cosThetaT) / (cosThetaI + iorRatio*cosThetaT))
	rp := sq((cosThetaT - iorRatio*cosThetaI) / (cosThetaT + iorRatio*cosThetaI))
	return 0.5 * (rs + rp)
}

// DielectricAverageFresnel approximates the cosine-weighted hemispherical average of
// [DielectricFresnel] over incidence angle, via the closed-form polynomial fit from the
// "Fresnel Equations Considered Harmful" course notes (Imageworks, SIGGRAPH 2017) — exact
// port of Adobe's openpbr_average_fresnel. Used by the Kulla-Conty multi-scatter
// compensation (multiscatter.go) so it needs no numerical integration over Fresnel.
func DielectricAverageFresnel(iorRatio float64) float64 {
	if iorRatio > 1 {
		return (iorRatio - 1) / (4.08567 + 1.00071*iorRatio)
	}
	return 0.997118 + 0.1014*iorRatio - 0.965241*sq(iorRatio) - 0.130607*sq(iorRatio)*iorRatio
}

// f82CosThetaMax is the "82 degree" reference angle cosine (1/7) the F82-tint model is
// pinned at, per the OpenPBR spec's Metal section.
const f82CosThetaMax = 1.0 / 7.0

// f82SchlickBFactor is the correction term "b" that makes F82Tint(f0, tint, f82CosThetaMax)
// equal tint*Schlick(f82CosThetaMax) exactly — exact port of Adobe's
// openpbr_compute_metal_schlick_b_factor.
func f82SchlickBFactor(f0, tint Color3) Color3 {
	const (
		oneMinusMax        = 1 - f82CosThetaMax
		oneMinusMaxToFifth = oneMinusMax * oneMinusMax * oneMinusMax * oneMinusMax * oneMinusMax
		oneMinusMaxToSixth = oneMinusMaxToFifth * oneMinusMax
		denom              = f82CosThetaMax * oneMinusMaxToSixth
	)
	numer := func(r, t float64) float64 { return (r + (1-r)*oneMinusMaxToFifth) * (1 - t) }
	return Color3{
		R: numer(f0.R, tint.R) / denom,
		G: numer(f0.G, tint.G) / denom,
		B: numer(f0.B, tint.B) / denom,
	}
}

// F82TintFresnel is the OpenPBR Metal section's conductor Fresnel model (spec eq.
// F82_tint): a Schlick curve corrected so its value at the 82° reference angle equals
// tint*Schlick(82°) exactly, letting specular_color art-direct the grazing-edge color of
// a metal independently of its normal-incidence color f0 — exact port of Adobe's
// openpbr_metal_schlick_with_f82_tint. cosTheta must be ≥ 0.
func F82TintFresnel(f0, tint Color3, cosTheta float64) Color3 {
	b := f82SchlickBFactor(f0, tint)
	oneMinusCos := 1 - cosTheta
	oneMinusCosToFifth := oneMinusCos * oneMinusCos * oneMinusCos * oneMinusCos * oneMinusCos
	channel := func(r, bC float64) float64 {
		v := r + ((1-r)-bC*cosTheta*oneMinusCos)*oneMinusCosToFifth
		return clamp(v, 0, 1)
	}
	return Color3{R: channel(f0.R, b.R), G: channel(f0.G, b.G), B: channel(f0.B, b.B)}
}

// F82TintAverageFresnel is the cosine-weighted hemispherical average of
// [F82TintFresnel] — exact port of Adobe's openpbr_metal_average_fresnel_with_f82_tint.
// Used by the Kulla-Conty multi-scatter compensation (multiscatter.go).
func F82TintAverageFresnel(f0, tint Color3) Color3 {
	b := f82SchlickBFactor(f0, tint)
	channel := func(r, bC float64) float64 { return clamp(r+(1-r)*(1.0/21)-bC*(1.0/126), 0, 1) }
	return Color3{R: channel(f0.R, b.R), G: channel(f0.G, b.G), B: channel(f0.B, b.B)}
}

func sq(x float64) float64 { return x * x }
