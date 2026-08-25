// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"math"
	"sort"
)

// EnvironmentDistribution is a 2D piecewise-constant importance-sampling distribution
// over an equirectangular HDR environment map (a row marginal CDF plus a per-row
// conditional CDF, the standard technique — e.g. PBRT §14.2.4), so a path integrator can
// draw directions proportional to environment radiance instead of sampling uniformly
// and wasting most paths on dim sky. It consumes a plain row-major pixel buffer rather
// than head/internal/envimage's Equirect type directly, so this package (renderer)
// never depends on the head-side image loader (envimage already imports renderer; the
// reverse would cycle) — the caller passes envimage.Equirect.Pixels straight through.
type EnvironmentDistribution struct {
	w, h int

	luma        [][]float64 // luma[y][x]: per-texel luminance, kept for PDF lookups
	marginalCDF []float64   // marginalCDF[y]: cumulative row mass over [0,y], scaled to end at 1
	rowCDF      [][]float64 // rowCDF[y][x]: row y's cumulative column mass, scaled to end at 1
	totalMass   float64     // sum over all texels of luma(x,y) * sin(theta(y))
}

// NewEnvironmentDistribution builds an importance-sampling distribution from a w×h
// equirectangular image's interleaved pixels (RGB or RGBA — stride is len(pixels)/(w*h)
// floats per texel; only the first 3 channels are read). Row weighting includes a
// sin(polar angle) solid-angle correction, since equirect rows near the poles cover far
// less sphere area than rows near the equator (without it, pole pixels would be
// over-sampled relative to their true solid angle).
func NewEnvironmentDistribution(w, h int, pixels []float32) *EnvironmentDistribution {
	stride := len(pixels) / (w * h)
	d := &EnvironmentDistribution{w: w, h: h, luma: make([][]float64, h), rowCDF: make([][]float64, h), marginalCDF: make([]float64, h)}

	var totalMass float64
	for y := 0; y < h; y++ {
		sinTheta := math.Sin(math.Pi * (float64(y) + 0.5) / float64(h))
		row, rowCDF := make([]float64, w), make([]float64, w)
		var rowMass float64
		for x := 0; x < w; x++ {
			i := (y*w + x) * stride
			row[x] = 0.2126*float64(pixels[i]) + 0.7152*float64(pixels[i+1]) + 0.0722*float64(pixels[i+2])
			rowMass += row[x] * sinTheta
			rowCDF[x] = rowMass
		}
		d.luma[y], d.rowCDF[y] = row, normalizedOrUniform(rowCDF, rowMass)
		totalMass += rowMass
		d.marginalCDF[y] = totalMass
	}
	d.totalMass = totalMass
	for y := range d.marginalCDF {
		d.marginalCDF[y] = safeDiv(d.marginalCDF[y], totalMass)
	}
	return d
}

// normalizedOrUniform scales cdf so it ends at 1, or — if the row's total mass is
// zero (a pure-black row) — returns a uniform CDF so Sample never divides by zero and
// still returns a valid (never-selected-in-practice) column.
func normalizedOrUniform(cdf []float64, mass float64) []float64 {
	if mass <= 0 {
		for x := range cdf {
			cdf[x] = float64(x+1) / float64(len(cdf))
		}
		return cdf
	}
	for x := range cdf {
		cdf[x] /= mass
	}
	return cdf
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// Sample draws a texel proportional to the map's solid-angle-weighted radiance using
// two independent uniforms u1 (picks the row), u2 (picks the column within it), and
// returns the direction that texel maps to plus its PDF in solid-angle measure — ready
// for a path integrator to divide the sampled radiance by.
func (d *EnvironmentDistribution) Sample(u1, u2 float64) (dir [3]float32, pdf float64) {
	y := clampIndex(sort.SearchFloat64s(d.marginalCDF, u1), d.h)
	x := clampIndex(sort.SearchFloat64s(d.rowCDF[y], u2), d.w)
	return d.direction(x, y), d.PDF(x, y)
}

func clampIndex(i, n int) int {
	if i >= n {
		return n - 1
	}
	return i
}

// PDF is texel (x,y)'s solid-angle probability density. The equirect parametrization's
// sin(theta) Jacobian appears both in a texel's selection mass and in the solid angle it
// subtends, so it cancels: PDF = luma(x,y) * W * H / (totalMass * 2π²) — see the
// package doc comment's derivation reference (PBRT §14.2.4, eq. 14.11 specialized to a
// piecewise-constant equirect map).
func (d *EnvironmentDistribution) PDF(x, y int) float64 {
	if d.totalMass <= 0 {
		return 0
	}
	return d.luma[y][x] * float64(d.w) * float64(d.h) / (d.totalMass * 2 * math.Pi * math.Pi)
}

// direction converts texel (x,y) to a unit direction, at the environment's own zero
// rotation: row 0 is the zenith (+Z), the last row the nadir. The azimuth formula below
// is the EXACT algebraic inverse (at rot=0) of the shader-side sampling convention this
// package must stay consistent with — skybox.frag's/openpbr/env_sample.glsl's
// `u = atan(d.y,d.x)/(2π) + 0.5 + rot/(2π)` — NOT the naive `2π*(x+0.5)/w` a reader might
// expect from envimage.Equirect's "columns sweep azimuth 0..2π" doc comment: that phrasing
// describes generate()'s arbitrary [0,1) authoring parameter, not the azimuth a SAMPLER
// recovers from a world direction. Solving u's formula for atan(d.y,d.x) at rot=0 with
// u=(x+0.5)/w gives phi = 2π*(x+0.5)/w - π (see TestEnvironmentDistributionMatchesShaderSampling,
// which round-trips this exactly). Getting this wrong doesn't just mis-align an unrelated
// texel — since Sample's returned pdf is computed from the (x,y) texel this function
// converts, a caller that later re-samples the environment TEXTURE at the returned
// direction (rather than reading pixels[] at (x,y) directly, as the live path tracer
// does — GPU texture, no CPU-side lookup) would divide one texel's radiance by a
// DIFFERENT texel's pdf: not merely inefficient importance sampling, but a biased
// estimator.
func (d *EnvironmentDistribution) direction(x, y int) [3]float32 {
	theta := math.Pi * (float64(y) + 0.5) / float64(d.h)
	phi := 2*math.Pi*(float64(x)+0.5)/float64(d.w) - math.Pi
	sinTheta := math.Sin(theta)
	return [3]float32{
		float32(sinTheta * math.Cos(phi)),
		float32(sinTheta * math.Sin(phi)),
		float32(math.Cos(theta)),
	}
}

// TotalWeight is the distribution's total solid-angle-weighted radiance mass — the
// environment's counterpart to LightDistribution's own (private) discrete-light total,
// exposed so a caller combining both into one selection probability (pEnv = envWeight /
// (envWeight + lightsWeight)) can read it without re-deriving the pixel sum itself.
func (d *EnvironmentDistribution) TotalWeight() float64 { return d.totalMass }

// RotateAroundZ applies the environment's runtime azimuthal rotation (app.EnvironmentState
// .Rotation, radians) to a direction already resolved at the distribution's zero-rotation
// baseline (direction/Sample above): solving env_sample.glsl's `u += rot/(2π)` for a FIXED
// texel shows the world-space azimuth shifts by -rot as rotation increases (a larger rot
// shows an EARLIER part of the map at a given screen direction), so this rotates by -rot,
// not +rot — see EnvironmentDistribution.direction's own doc comment for the u/phi algebra
// this inverts.
func RotateAroundZ(dir [3]float32, rotation float32) [3]float32 {
	s, c := math.Sincos(float64(-rotation))
	x, y := float64(dir[0]), float64(dir[1])
	return [3]float32{float32(x*c - y*s), float32(x*s + y*c), dir[2]}
}
