// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"math"
	"math/rand"
	"testing"
)

// TestLightDistributionMatchesBruteForceRelativeContribution is PBI-348's core
// acceptance criterion: a scene lit by the existing multi-light rig
// ([LightingThreePoint], renderer/lighting_styles.go) must converge, via power-weighted
// importance sampling, to the same RELATIVE per-light contributions as a brute-force
// (evaluate-every-light) reference — a correctness check on the estimator, not a speed
// benchmark.
func TestLightDistributionMatchesBruteForceRelativeContribution(t *testing.T) {
	lights := SceneLightingFor(LightingThreePoint).Lights
	albedo := [3]float32{0.8, 0.8, 0.8}
	// Faces roughly between the key and fill lights, so both contribute (the back/rim
	// light faces the opposite way and legitimately contributes zero from here — that
	// is itself part of what this test must tolerate without biasing the estimate).
	normal := normalize32([3]float32{-0.2, 0.8, 1.5})

	bruteForce := make([][3]float32, len(lights))
	for i, l := range lights {
		bruteForce[i] = DirectContribution(l, normal, albedo)
	}

	dist := NewLightDistribution(lights)
	estimate := make([][3]float32, len(lights))
	const n = 200_000
	rng := rand.New(rand.NewSource(7))
	for s := 0; s < n; s++ {
		i, light, pdf := dist.Sample(rng.Float64())
		c := DirectContribution(light, normal, albedo)
		estimate[i][0] += c[0] / float32(pdf)
		estimate[i][1] += c[1] / float32(pdf)
		estimate[i][2] += c[2] / float32(pdf)
	}
	for i := range estimate {
		for c := 0; c < 3; c++ {
			estimate[i][c] /= n
		}
	}

	// Relative contribution: each light's share of the total (brute force) vs. share of
	// the total (importance-sampled estimate), compared channel by channel.
	var bruteTotal, estTotal float64
	for i := range lights {
		bruteTotal += float64(bruteForce[i][1]) // green channel is representative (albedo is grey)
		estTotal += float64(estimate[i][1])
	}
	if bruteTotal == 0 {
		t.Fatal("test fixture produced zero total contribution — normal/light setup is degenerate")
	}
	for i, l := range lights {
		bruteShare := float64(bruteForce[i][1]) / bruteTotal
		estShare := float64(estimate[i][1]) / estTotal
		if diff := math.Abs(bruteShare - estShare); diff > 0.01 {
			t.Errorf("light %d (kind %v, intensity %v): brute-force share %v, importance-sampled share %v, diff %v > 0.01",
				i, l.Kind, l.Intensity, bruteShare, estShare, diff)
		}
	}
}

func TestLightDistributionSampleAlwaysReturnsAPositiveWeightLight(t *testing.T) {
	lights := []SceneLight{
		{Kind: DirectionalLight, Direction: [3]float32{0, 0, 1}, Color: [3]float32{1, 1, 1}, Intensity: 0, On: true},
		{Kind: DirectionalLight, Direction: [3]float32{0, 1, 0}, Color: [3]float32{1, 1, 1}, Intensity: 5, On: true},
	}
	dist := NewLightDistribution(lights)
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 1000; i++ {
		index, _, pdf := dist.Sample(rng.Float64())
		if index != 1 {
			t.Fatalf("Sample picked the zero-weight light (index %d) with pdf %v", index, pdf)
		}
	}
}

func TestLightDistributionPDFSumsToOne(t *testing.T) {
	lights := SceneLightingFor(LightingThreePoint).Lights
	dist := NewLightDistribution(lights)
	var sum float64
	for i := 0; i < dist.Len(); i++ {
		sum += dist.PDF(i)
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Errorf("sum of PDF(i) over all lights = %v, want 1", sum)
	}
}

func TestLightDistributionSampleAllZeroWeightPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Sample on an all-zero-weight distribution did not panic")
		}
	}()
	dist := NewLightDistribution([]SceneLight{{Kind: DirectionalLight, Intensity: 0, On: true}})
	dist.Sample(0.5)
}

func TestDirectContributionSkipsOffOrNonDirectionalLights(t *testing.T) {
	normal := [3]float32{0, 0, 1}
	albedo := [3]float32{1, 1, 1}
	off := SceneLight{Kind: DirectionalLight, Direction: [3]float32{0, 0, 1}, Color: [3]float32{1, 1, 1}, Intensity: 5, On: false}
	point := SceneLight{Kind: PointLight, Direction: [3]float32{0, 0, 1}, Color: [3]float32{1, 1, 1}, Intensity: 5, On: true}

	if c := DirectContribution(off, normal, albedo); c != ([3]float32{}) {
		t.Errorf("off light contributed %v, want zero", c)
	}
	if c := DirectContribution(point, normal, albedo); c != ([3]float32{}) {
		t.Errorf("point light (unhandled by this analytic path) contributed %v, want zero", c)
	}
}
