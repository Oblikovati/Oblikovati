// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "sort"

// lightWeight is the power-based, position-independent picking weight the light
// importance sampler uses to choose which light to evaluate on a given path sample:
// luminance(Color) * Intensity. This is the standard path-tracing baseline (PBRT's
// PowerLightSampler) — not a full spatial light BVH/ReSTIR, which a fixed few-light CAD
// rig's "efficient convergence" scope (PBI-348) does not need.
func lightWeight(l SceneLight) float64 {
	luma := 0.2126*float64(l.Color[0]) + 0.7152*float64(l.Color[1]) + 0.0722*float64(l.Color[2])
	return luma * float64(l.Intensity)
}

// LightDistribution is a power-weighted discrete distribution over a fixed light list,
// used to pick one light per path sample with the probability an unbiased Monte Carlo
// light-selection estimator needs (evaluated contribution / PDF).
type LightDistribution struct {
	lights []SceneLight
	cdf    []float64 // cdf[i] = cumulative weight of lights[0..i]; cdf[len-1] == total
	total  float64
}

// NewLightDistribution builds a power-weighted distribution over lights.
func NewLightDistribution(lights []SceneLight) *LightDistribution {
	d := &LightDistribution{lights: lights, cdf: make([]float64, len(lights))}
	sum := 0.0
	for i, l := range lights {
		sum += lightWeight(l)
		d.cdf[i] = sum
	}
	d.total = sum
	return d
}

// Len is the number of lights in the distribution.
func (d *LightDistribution) Len() int { return len(d.lights) }

// TotalWeight is the distribution's total power-based weight — a caller combining this
// distribution with EnvironmentDistribution (pEnv = envWeight / (envWeight + this)) needs
// it to weight the two selection strategies against each other.
func (d *LightDistribution) TotalWeight() float64 { return d.total }

// PDF is light index i's discrete selection probability (weight_i / total weight).
func (d *LightDistribution) PDF(i int) float64 {
	if d.total <= 0 {
		return 0
	}
	return lightWeight(d.lights[i]) / d.total
}

// Sample picks one light using u ∈ [0,1) and returns its index, the light itself, and
// its selection PDF — the caller divides the light's evaluated contribution by PDF to
// keep the estimator unbiased. Panics if every light has zero weight (nothing to pick).
func (d *LightDistribution) Sample(u float64) (index int, light SceneLight, pdf float64) {
	if d.total <= 0 {
		panic("renderer: LightDistribution.Sample: no lights with positive weight")
	}
	target := u * d.total
	i := sort.Search(len(d.cdf), func(i int) bool { return d.cdf[i] >= target })
	if i == len(d.cdf) {
		i = len(d.cdf) - 1
	}
	return i, d.lights[i], d.PDF(i)
}

// DirectContribution is the analytic direct-lighting contribution of a single light at
// a Lambertian surface point with the given unit normal and base color: the incoming
// radiance a brute-force per-light sum would add for l, before any visibility test.
// Only DirectionalLight is evaluated exactly (matches this PBI's fixture rigs,
// [lighting_styles.go] — Point/Spot distance attenuation feeds the same formula once a
// path integrator needs it, tracked as part of the F05 wiring PBI).
func DirectContribution(l SceneLight, normal, albedo [3]float32) [3]float32 {
	if !l.On || l.Kind != DirectionalLight {
		return [3]float32{}
	}
	ndotl := dot32(normal, normalize32(l.Direction))
	if ndotl <= 0 {
		return [3]float32{}
	}
	scale := ndotl * l.Intensity
	return [3]float32{
		albedo[0] * l.Color[0] * scale,
		albedo[1] * l.Color[1] * scale,
		albedo[2] * l.Color[2] * scale,
	}
}
