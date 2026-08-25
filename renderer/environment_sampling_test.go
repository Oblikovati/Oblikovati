// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"math"
	"math/rand"
	"testing"
)

// TestEnvironmentDistributionPDFIntegratesToOne checks the PDF is properly normalized:
// summing PDF(x,y) * texel-solid-angle over every texel must equal 1, exactly (a
// piecewise-constant density integrates exactly at texel resolution, no Monte Carlo
// needed). A bug in the sin(theta) cancellation derivation would show up here directly.
func TestEnvironmentDistributionPDFIntegratesToOne(t *testing.T) {
	const w, h = 64, 32
	pixels := make([]float32, w*h*4)
	rng := rand.New(rand.NewSource(11))
	for i := 0; i < w*h; i++ {
		v := rng.Float32() * 5
		pixels[i*4], pixels[i*4+1], pixels[i*4+2], pixels[i*4+3] = v, v*0.8, v*0.6, 1
	}
	d := NewEnvironmentDistribution(w, h, pixels)

	var integral float64
	for y := 0; y < h; y++ {
		solidAngle := math.Sin(math.Pi*(float64(y)+0.5)/float64(h)) * (math.Pi / float64(h)) * (2 * math.Pi / float64(w))
		for x := 0; x < w; x++ {
			integral += d.PDF(x, y) * solidAngle
		}
	}
	if math.Abs(integral-1) > 1e-9 {
		t.Errorf("PDF integrated over the sphere = %v, want 1", integral)
	}
}

// TestEnvironmentDistributionConcentratesOnBrightRegion is PBI-348's environment-map
// scope item: samples must concentrate in bright texels proportional to their radiance,
// not uniformly. Half the map (left column half) is bright, half is dim; the analytic
// expected fraction of solid-angle-weighted radiance mass in the bright half is
// computed directly from the same weighting the distribution uses, then compared
// against what empirically gets sampled.
func TestEnvironmentDistributionConcentratesOnBrightRegion(t *testing.T) {
	const w, h = 64, 32
	const bright, dim = 10.0, 1.0
	pixels := make([]float32, w*h*4)
	var brightMass, totalMass float64
	for y := 0; y < h; y++ {
		sinTheta := math.Sin(math.Pi * (float64(y) + 0.5) / float64(h))
		for x := 0; x < w; x++ {
			v := float32(dim)
			if x < w/4 { // a quarter of the columns are the bright region
				v = float32(bright)
			}
			i := (y*w + x) * 4
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = v, v, v, 1
			totalMass += float64(v) * sinTheta
			if x < w/4 {
				brightMass += float64(v) * sinTheta
			}
		}
	}
	expectedBrightFraction := brightMass / totalMass

	d := NewEnvironmentDistribution(w, h, pixels)
	rng := rand.New(rand.NewSource(23))
	const n = 200_000
	hits := 0
	for i := 0; i < n; i++ {
		x := sampleColumnIndex(d, rng.Float64(), rng.Float64())
		if x < w/4 {
			hits++
		}
	}
	got := float64(hits) / n
	if diff := math.Abs(got - expectedBrightFraction); diff > 0.01 {
		t.Errorf("empirical bright-region sample fraction = %v, want %v (analytic weighted share), diff %v > 0.01",
			got, expectedBrightFraction, diff)
	}
}

// sampleColumnIndex draws one sample and recovers which column it landed in, by
// inverting direction() well enough for this test's purposes: azimuth back to x.
func sampleColumnIndex(d *EnvironmentDistribution, u1, u2 float64) int {
	dir, _ := d.Sample(u1, u2)
	phi := math.Atan2(float64(dir[1]), float64(dir[0]))
	if phi < 0 {
		phi += 2 * math.Pi
	}
	x := int(phi / (2 * math.Pi) * float64(d.w))
	return clampIndex(x, d.w)
}

func TestEnvironmentDistributionDirectionIsUnitLength(t *testing.T) {
	const w, h = 16, 8
	pixels := make([]float32, w*h*4)
	for i := range pixels {
		pixels[i] = 1
	}
	d := NewEnvironmentDistribution(w, h, pixels)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dir := d.direction(x, y)
			length := math.Sqrt(float64(dir[0]*dir[0] + dir[1]*dir[1] + dir[2]*dir[2]))
			if math.Abs(length-1) > 1e-5 {
				t.Fatalf("direction(%d,%d) = %v, length %v, want 1", x, y, dir, length)
			}
		}
	}
}

func TestEnvironmentDistributionAllBlackNeverPanics(t *testing.T) {
	const w, h = 8, 4
	pixels := make([]float32, w*h*4)
	d := NewEnvironmentDistribution(w, h, pixels)
	if pdf := d.PDF(0, 0); pdf != 0 {
		t.Errorf("PDF on an all-black map = %v, want 0", pdf)
	}
	// Must not divide by zero / panic even though every row is zero mass.
	_, pdf := d.Sample(0.5, 0.5)
	if pdf != 0 {
		t.Errorf("Sample PDF on an all-black map = %v, want 0", pdf)
	}
}
