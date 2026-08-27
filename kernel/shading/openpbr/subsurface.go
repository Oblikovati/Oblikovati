// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"

	gmath "oblikovati.org/math"
)

// SubsurfaceExtinction returns the per-channel volumetric extinction coefficient μt
// (spec §Subsurface: "the reciprocal of the MFP per channel"), regularized near a
// zero mean free path (radius·radiusScale → 0, i.e. an opaque-diffuse limit) to avoid a
// divide-by-zero rather than producing +Inf.
func SubsurfaceExtinction(radius float64, radiusScale Color3) Color3 {
	const minMFP = 1e-6
	channel := func(scale float64) float64 { return 1 / math.Max(radius*scale, minMFP) }
	return Color3{R: channel(radiusScale.R), G: channel(radiusScale.G), B: channel(radiusScale.B)}
}

// subsurfaceAlbedoFromColorChannel is the van de Hulst inversion (spec index.html,
// §Subsurface, the un-numbered equation following eq. subsurface_albedo_constraint):
// the per-channel single-scattering albedo α that generates observed subsurface_color c
// at anisotropy g, assuming an index-matched boundary. Exact port of the spec's own
// closed-form inversion (not re-derived independently).
func subsurfaceAlbedoFromColorChannel(c, g float64) float64 {
	s := 4.09712 + 4.20863*c - math.Sqrt(math.Max(0, 9.59217+41.6808*c+17.7126*c*c))
	sSq := s * s
	denom := 1 - g*sSq
	if math.Abs(denom) < 1e-9 {
		return 0
	}
	return (1 - sSq) / denom
}

// SubsurfaceSingleScatterAlbedo applies [subsurfaceAlbedoFromColorChannel] per RGB
// channel — the α the caller's volumetric medium (random-walk or BSSRDF) needs, given
// subsurface_color and subsurface_scatter_anisotropy.
func SubsurfaceSingleScatterAlbedo(color Color3, anisotropy float64) Color3 {
	return Color3{
		R: subsurfaceAlbedoFromColorChannel(color.R, anisotropy),
		G: subsurfaceAlbedoFromColorChannel(color.G, anisotropy),
		B: subsurfaceAlbedoFromColorChannel(color.B, anisotropy),
	}
}

// subsurfaceColorFromAlbedoChannel is the van de Hulst FORWARD formula (spec eq. between
// subsurface_albedo_constraint and its inversion): the observed color an index-matched
// medium of single-scattering albedo α and anisotropy g produces. Exists primarily to
// let subsurfaceAlbedoFromColorChannel's inversion be round-trip tested against the
// spec's own forward relationship (subsurface_test.go).
func subsurfaceColorFromAlbedoChannel(alpha, g float64) float64 {
	num := 1 - alpha
	denom := 1 - alpha*g
	if denom <= 0 {
		return 1
	}
	s := math.Sqrt(gmath.Clamp(num/denom, 0, 1))
	return (1 - s) * (1 - 0.139*s) / (1 + 1.17*s)
}

// PhaseHenyeyGreenstein evaluates the standard Henyey-Greenstein volumetric phase
// function (spec §Subsurface: "the phase function has the standard Henyey-Greenstein
// form") at the angle between incoming and outgoing directions, cosTheta = wi·wo,
// anisotropy g ∈ (-1,1) (the spec itself notes implementations should clamp away from
// the ±1 limits, where the phase function becomes degenerate).
func PhaseHenyeyGreenstein(cosTheta, g float64) float64 {
	g = gmath.Clamp(g, -0.999, 0.999)
	gSq := g * g
	denom := 1 + gSq - 2*g*cosTheta
	return (1 - gSq) / (4 * math.Pi * math.Pow(math.Max(denom, 1e-9), 1.5))
}

// SubsurfaceMedium bundles the resolved per-channel volumetric parameters (spec
// §Subsurface: extinction μt, single-scattering albedo α, scalar phase-function
// anisotropy g) a random-walk or BSSRDF subsurface integrator consumes. Building the
// actual walk needs the F04 path integrator's ray-bouncing support (PBI-347); this
// package only resolves the OpenPBR parameters into these physical quantities.
type SubsurfaceMedium struct {
	Extinction Color3
	Albedo     Color3
	Anisotropy float64
}

// ResolveSubsurfaceMedium builds a [SubsurfaceMedium] from the OpenPBR Subsurface
// parameter group.
func ResolveSubsurfaceMedium(color Color3, radius float64, radiusScale Color3, anisotropy float64) SubsurfaceMedium {
	return SubsurfaceMedium{
		Extinction: SubsurfaceExtinction(radius, radiusScale),
		Albedo:     SubsurfaceSingleScatterAlbedo(color, anisotropy),
		Anisotropy: anisotropy,
	}
}

// MixSubsurface blends the subsurface term over the glossy-diffuse base (spec params
// table: "Mix weight between subsurface and diffuse slabs"). weight<=0 returns diffuse
// unchanged — the PBI-343 regression guard that subsurface_weight=0 reproduces PBI-342's
// output exactly.
func MixSubsurface(diffuse, subsurface Color3, weight float64) Color3 {
	if weight <= 0 {
		return diffuse
	}
	return lerpColor(diffuse, subsurface, weight)
}
