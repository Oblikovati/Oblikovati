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

// direction converts texel (x,y) to a unit direction: row 0 is the zenith (+Z), the
// last row the nadir, columns sweep azimuth 0..2π — matching envimage.Equirect's own
// documented convention exactly, so callers can pass its pixels through unmodified.
func (d *EnvironmentDistribution) direction(x, y int) [3]float32 {
	theta := math.Pi * (float64(y) + 0.5) / float64(d.h)
	phi := 2 * math.Pi * (float64(x) + 0.5) / float64(d.w)
	sinTheta := math.Sin(theta)
	return [3]float32{
		float32(sinTheta * math.Cos(phi)),
		float32(sinTheta * math.Sin(phi)),
		float32(math.Cos(theta)),
	}
}
