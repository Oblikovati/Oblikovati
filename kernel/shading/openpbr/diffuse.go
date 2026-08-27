// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"

	gmath "oblikovati.org/math"
)

// fonConstantA / fonConstantB are the Fujii-Oren-Nayar (FON) closed-form fit constants
// from the OpenPBR spec's energy-preserving Oren-Nayar formulation (index.html
// "Glossy-diffuse" section, equations FON_brdf/EON_comp), exact port of Adobe's
// impl/openpbr_diffuse_lobe.h OpenPBR_FONConstantA/B.
const (
	fonConstantA = 0.5 - 2.0/(3.0*math.Pi)
	fonConstantB = 2.0/3.0 - 28.0/(15.0*math.Pi)
)

// directionalAlbedoFON returns the Fujii Oren-Nayar directional albedo at mu = cos(theta)
// for the given roughness in [0,1] — the exact (non-approximated) closed form, ported
// from Adobe's openpbr_E_FON_exact. It is finite at mu=0 and cancellation-free near
// grazing (Si→1), per the reference's own derivation note.
func directionalAlbedoFON(mu, roughness float64) float64 {
	af := 1 / (1 + fonConstantA*roughness)
	bf := roughness * af
	clampedMu := gmath.Clamp(mu, 0, 1)
	si := math.Sqrt(1 - clampedMu*clampedMu)
	g := si*(math.Acos(clampedMu)-si*clampedMu) +
		(2.0/3.0)*(si*clampedMu*(1+si+si*si)/(1+si)-si)
	return af + (bf/math.Pi)*g
}

// DiffuseEON evaluates the OpenPBR Surface Base diffuse lobe (spec eq. EON_brdf): the
// energy-preserving Oren-Nayar BRDF, exact port of Adobe's openpbr_f_EON (the "exact",
// not "approx", directional-albedo branch — the CPU reference favors accuracy over the
// GLSL runtime's cheaper polynomial fit).
//
// rho is base_weight*base_color, roughness is base_diffuse_roughness in [0,1], and wi/wo
// are in local shading space (package doc). Returns f_ss + f_ms (single-scatter lobe plus
// the microfacet multiple-scattering compensation that makes a white rho pass the white
// furnace test — see the package doc's Kulla-Conty note, applied here to Oren-Nayar's own
// microsurface rather than a separate specular lobe).
func DiffuseEON(rho Color3, roughness float64, wi, wo Vec3) Color3 {
	muI, muO := wi.CosTheta(), wo.CosTheta()
	s := wi.Dot(wo) - muI*muO
	sOverT := s
	if s > 0 {
		sOverT = s / math.Max(muI, muO)
	}
	af := 1 / (1 + fonConstantA*roughness)
	fSS := rho.Scale(af * (1 + roughness*sOverT) / math.Pi)

	efO := directionalAlbedoFON(muO, roughness)
	efI := directionalAlbedoFON(muI, roughness)
	avgEF := af * (1 + fonConstantB*roughness)

	const eps = 1e-7
	// rho_ms = rho^2 * avgEF / (1 - rho*(1-avgEF)), channel-wise (Adobe's vec3 rho * rho).
	rhoMS := Color3{
		R: channelRhoMS(rho.R, avgEF),
		G: channelRhoMS(rho.G, avgEF),
		B: channelRhoMS(rho.B, avgEF),
	}
	msFactor := math.Max(eps, 1-efO) * math.Max(eps, 1-efI) / math.Max(eps, 1-avgEF) / math.Pi
	fMS := rhoMS.Scale(msFactor)

	return fSS.Add(fMS)
}

func channelRhoMS(rho, avgEF float64) float64 {
	denom := 1 - rho*(1-avgEF)
	if denom == 0 {
		return 0
	}
	return rho * rho * avgEF / denom
}
