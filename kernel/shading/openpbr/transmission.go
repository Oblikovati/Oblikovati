// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// --- Nested-dielectric ambient-IOR stack (spec index.html line 130: "if the renderer
// keeps track of the dielectric medium in which the surface is embedded (via a scheme
// such as 'nested dielectrics' [#Budge2002])...") ---

// IORStack is a per-path stack of the dielectric media a ray is currently inside,
// innermost (most recently entered) on top. It starts holding just the ambient medium
// (IOR 1, vacuum/air) — the ["no tracking" fallback](index.html line 130) the spec
// describes for renderers that don't track nested dielectrics, generalized here so a
// renderer that DOES track them (e.g. glass submerged in water) gets the correct
// surrounding IOR at each interface instead.
type IORStack struct {
	iors []float64
}

// NewIORStack returns a stack holding only the ambient medium.
func NewIORStack() *IORStack { return &IORStack{iors: []float64{1}} }

// Top returns the IOR of the medium the ray is currently traveling through.
func (s *IORStack) Top() float64 { return s.iors[len(s.iors)-1] }

// Push enters a new dielectric medium (a refraction event crossing into a denser/rarer
// medium), e.g. a ray refracting from water into a submerged glass bead.
func (s *IORStack) Push(ior float64) { s.iors = append(s.iors, ior) }

// Pop exits the innermost medium, returning to whatever surrounded it (e.g. leaving the
// glass bead back into the water). A no-op below the ambient medium (mismatched
// push/pop from a non-manifold or misclassified boundary must never crash the stack).
func (s *IORStack) Pop() {
	if len(s.iors) > 1 {
		s.iors = s.iors[:len(s.iors)-1]
	}
}

// Depth reports how many nested media the ray is currently inside (1 = ambient only).
func (s *IORStack) Depth() int { return len(s.iors) }

// RelativeIOR is the ratio η_t/η_i a Fresnel/Snell evaluation needs at a boundary into a
// medium of IOR materialIOR, given the ray is currently in the stack's top medium.
func (s *IORStack) RelativeIOR(materialIOR float64) float64 { return materialIOR / s.Top() }

// --- Abbe-number dispersion (spec §Dispersion: Cauchy empirical formula) ---

// Fraunhofer C/d/F spectral line wavelengths (nm) the spec's Abbe-number definition uses
// (index.html line 811): λC=656.3 (long/red), λd=587.6 (medium/yellow, the reference
// wavelength specular_ior is defined at), λF=486.1 (short/blue).
const (
	fraunhoferLambdaCNM = 656.3
	fraunhoferLambdaDNM = 587.6
	fraunhoferLambdaFNM = 486.1
)

// abbeNumberEffective is spec eq. (index.html line 825): the artist-friendly
// transmission_dispersion_scale/transmission_dispersion_abbe_number pair mapped to the
// physical Abbe number V_d. scale=0 gives +Inf (no dispersion, B=0 below).
func abbeNumberEffective(abbeNumber, dispersionScale float64) float64 {
	if dispersionScale <= 0 {
		return math.Inf(1)
	}
	return abbeNumber / dispersionScale
}

// cauchyCoefficients returns the Cauchy formula's A/B coefficients (spec eq. following
// index.html line 815) for a dielectric whose IOR at the Fraunhofer d line is nd, given
// its Abbe number vd. B=0 (and thus a wavelength-independent IOR) when vd is infinite.
func cauchyCoefficients(nd, vd float64) (a, b float64) {
	if math.IsInf(vd, 1) {
		return nd, 0
	}
	invLambdaFSq := 1 / (fraunhoferLambdaFNM * fraunhoferLambdaFNM)
	invLambdaCSq := 1 / (fraunhoferLambdaCNM * fraunhoferLambdaCNM)
	b = (nd - 1) / (vd * (invLambdaFSq - invLambdaCSq))
	a = nd - b/(fraunhoferLambdaDNM*fraunhoferLambdaDNM)
	return a, b
}

// DispersiveIOR evaluates the spec's Cauchy dispersion model (index.html §Dispersion):
// the wavelength-dependent IOR n(λ) of a dielectric whose reference IOR is nd (at the
// Fraunhofer d line — specular_ior, per the spec's own convention) with the given
// transmission_dispersion_scale/abbe_number pair. dispersionScale=0 returns nd unchanged
// at every wavelength — the PBI-344 regression guard for the dispersion sub-feature.
func DispersiveIOR(nd, abbeNumber, dispersionScale, wavelengthNM float64) float64 {
	vd := abbeNumberEffective(abbeNumber, dispersionScale)
	a, b := cauchyCoefficients(nd, vd)
	return a + b/(wavelengthNM*wavelengthNM)
}

// --- Vector Snell refraction (standard form, e.g. Pharr/Jakob/Humphreys "Physically
// Based Rendering" — the OpenPBR spec's own #Pharr2023 citation) ---

// Refract computes the refracted direction for wi (local shading space, pointing AWAY
// from the surface per this package's convention) crossing into a medium of relative IOR
// iorRatio = η_t/η_i. ok is false under total internal reflection (no refracted ray).
func Refract(wi Vec3, iorRatio float64) (wt Vec3, ok bool) {
	cosThetaI := wi.CosTheta()
	etaRel := 1 / iorRatio // η_i/η_t, the ratio Snell's law actually scales the tangential component by
	sinThetaISq := math.Max(0, 1-cosThetaI*cosThetaI)
	sinThetaTSq := etaRel * etaRel * sinThetaISq
	if sinThetaTSq >= 1 {
		return Vec3{}, false
	}
	cosThetaT := math.Sqrt(1 - sinThetaTSq)
	normal := Vec3{Z: 1}
	wt = wi.Scale(-etaRel).Add(normal.Scale(etaRel*cosThetaI - cosThetaT))
	return wt.Normalize(), true
}

// --- Transmission color/depth → volumetric extinction (Beer's law) ---

// TransmissionExtinction is Beer's law inverted (spec eq., index.html line 782): the
// per-channel extinction coefficient μt such that white light becomes exactly
// transmission_color after traveling transmission_depth through the medium. depth<=0
// means the interior medium is absent (spec: "transmission_color is used instead to
// non-physically tint... multiplicatively"), so this returns zero extinction.
func TransmissionExtinction(color Color3, depth float64) Color3 {
	if depth <= 0 {
		return Color3{}
	}
	channel := func(t float64) float64 {
		if t <= 0 {
			t = 1e-6 // fully opaque channel: clamp rather than -ln(0) = +Inf
		}
		return -math.Log(t) / depth
	}
	return Color3{R: channel(color.R), G: channel(color.G), B: channel(color.B)}
}

// --- Thin-walled evaluation mode (spec index.html line 1259: a geometrical series over
// internal reflections in an infinitesimally thin dielectric sheet) ---

// ThinWallFresnel is the total reflectance from BOTH faces of a thin-walled dielectric
// sheet, including every internal bounce (spec's geometric-series sum, index.html line
// 1259) — a direct algebraic sum of the front-face Fresnel R geometric series
// R + (1-R)²R + (1-R)²R³ + ... = 2R/(1+R), exact port of Adobe's openpbr_thin_wall_fresnel.
func ThinWallFresnel(ior, cosThetaI float64) float64 {
	r := DielectricFresnel(ior, cosThetaI)
	return (2 * r) / (1 + r)
}

// ThinWallTransmittance is the complementary total transmittance through a thin-walled
// sheet, (1-R)/(1+R) — the fraction of light that passes straight through undeflected
// (spec's "un-deflected refracted lobe").
func ThinWallTransmittance(ior, cosThetaI float64) float64 {
	r := DielectricFresnel(ior, cosThetaI)
	return (1 - r) / (1 + r)
}

// MixTransmission blends the transmission lobe over the opaque base (spec §Base
// Substrate: "M_dielectric-base = mix(M_opaque-base, S_translucent-base, transmission_
// weight)"). weight<=0 returns opaque unchanged — the PBI-344 regression guard that
// transmission_weight=0 reproduces PBI-343's output exactly.
func MixTransmission(opaque, transmission Color3, weight float64) Color3 {
	if weight <= 0 {
		return opaque
	}
	return lerpColor(opaque, transmission, weight)
}
