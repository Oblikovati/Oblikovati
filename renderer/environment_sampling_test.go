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
// inverting direction()'s `phi = 2π(x+0.5)/w - π` well enough for this test's purposes:
// azimuth back to x (see direction's own doc comment for why that specific formula, not
// the naive phi=2π*x/w a reader might otherwise reach for here).
func sampleColumnIndex(d *EnvironmentDistribution, u1, u2 float64) int {
	dir, _ := d.Sample(u1, u2)
	phi := math.Atan2(float64(dir[1]), float64(dir[0])) // (-π, π]
	x := int((phi + math.Pi) / (2 * math.Pi) * float64(d.w))
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

// dirUV ports openpbr/env_sample.glsl's openpbrEnvDirUV (== skybox.frag's dirUV) to Go
// byte-for-byte, so this test can assert direction(x,y) is its algebraic inverse at
// rot=0 — the exact property EnvironmentDistribution.direction's own doc comment argues
// for. A mismatch here means the live path tracer would divide one texel's GPU-sampled
// radiance by an unrelated texel's CPU-computed pdf: a biased estimator, not merely a
// less-efficient one.
func dirUV(d [3]float32, rot float64) (u, v float64) {
	u = math.Atan2(float64(d[1]), float64(d[0]))/(2*math.Pi) + 0.5 + rot/(2*math.Pi)
	u -= math.Floor(u) // wrap into [0,1), matching GLSL's default texture-sampler REPEAT wrap
	v = math.Acos(clampFloat(float64(d[2]), -1, 1)) / math.Pi
	return u, v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func TestEnvironmentDistributionDirectionMatchesShaderSampling(t *testing.T) {
	const w, h = 32, 16
	pixels := make([]float32, w*h*4)
	for i := range pixels {
		pixels[i] = 1
	}
	d := NewEnvironmentDistribution(w, h, pixels)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dir := d.direction(x, y)
			u, v := dirUV(dir, 0)
			wantU, wantV := (float64(x)+0.5)/float64(w), (float64(y)+0.5)/float64(h)
			if diff := math.Abs(u - wantU); diff > 1e-6 {
				t.Errorf("direction(%d,%d) -> dirUV u=%v, want %v (diff %v)", x, y, u, wantU, diff)
			}
			if diff := math.Abs(v - wantV); diff > 1e-6 {
				t.Errorf("direction(%d,%d) -> dirUV v=%v, want %v (diff %v)", x, y, v, wantV, diff)
			}
		}
	}
}

// TestRotateAroundZMatchesShaderRotation checks the invariant RotateAroundZ exists to
// establish: sampling the ROTATED direction through dirUV WITH the real rotation applied
// (as env_sample.glsl's shaders actually do, texel lookups at runtime rotation) lands on
// the SAME texel as sampling the original (rot=0) direction with no rotation — i.e.
// RotateAroundZ pre-compensates the direction so the shader's own `u += rot/(2π)` term
// exactly cancels back out to the originally-picked texel. Confirms the sign (rotate by
// -rot, not +rot): dirUV(RotateAroundZ(dir,rot), rot) == dirUV(dir, 0).
func TestRotateAroundZMatchesShaderRotation(t *testing.T) {
	const w, h = 32, 16
	pixels := make([]float32, w*h*4)
	for i := range pixels {
		pixels[i] = 1
	}
	d := NewEnvironmentDistribution(w, h, pixels)
	x, y := 5, 4
	base := d.direction(x, y)
	wantU, wantV := dirUV(base, 0)

	const rotation = 0.7 // radians
	rotated := RotateAroundZ(base, rotation)
	gotU, gotV := dirUV(rotated, rotation)

	if diff := math.Abs(gotU - wantU); diff > 1e-6 {
		t.Errorf("RotateAroundZ(rot=%v): dirUV(rotated, rot).u=%v, want %v (diff %v)", rotation, gotU, wantU, diff)
	}
	if diff := math.Abs(gotV - wantV); diff > 1e-6 {
		t.Errorf("RotateAroundZ(rot=%v): dirUV(rotated, rot).v=%v, want %v (diff %v)", rotation, gotV, wantV, diff)
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
