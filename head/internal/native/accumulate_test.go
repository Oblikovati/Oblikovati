//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"math"
	"math/rand"
	"testing"

	"oblikovati.org/renderer"
)

// TestAccumulatorConvergesOverRealTraces is PBI-347's acceptance criterion exercised
// against genuine hardware-traced samples, not synthetic noise (renderer's own
// accumulator_test.go already covers the pure-math convergence property in isolation).
// A ray is aimed at the quad's x=5 edge with a small sub-pixel jitter, so roughly half
// the real SWScene.Trace queries hit and half miss — the analytic ground truth
// coverage is exactly 0.5 by symmetry. Accumulating that hit/miss signal must converge
// toward 0.5 as more real GPU samples are added.
func TestAccumulatorConvergesOverRealTraces(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (accumulator convergence test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer scene.Destroy()

	triangles := []renderer.Triangle{
		{V0: [3]float32{-5, -5, 0}, V1: [3]float32{5, -5, 0}, V2: [3]float32{5, 5, 0}, InstanceID: 1, PrimitiveID: 0},
		{V0: [3]float32{-5, -5, 0}, V1: [3]float32{5, 5, 0}, V2: [3]float32{-5, 5, 0}, InstanceID: 1, PrimitiveID: 1},
	}
	bvh := renderer.BuildBVH(triangles)
	if err := scene.Build(SWBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("Build: %v", err)
	}

	const groundTruth = 0.5
	origin := [3]float32{0, 0, 2}
	rng := rand.New(rand.NewSource(42))
	trace := func() float32 {
		jitter := (rng.Float64()*2 - 1) * 0.05 // uniform in [-0.05, 0.05), symmetric around the edge
		target := [3]float32{5 + float32(jitter), 0, 0}
		dir := normalize(sub(target, origin))
		if scene.Trace(origin, dir, 0, 1e6).Hit {
			return 1
		}
		return 0
	}

	a := renderer.NewAccumulator(1, 1)
	checkpoints := []int{50, 500, 5000}
	var errs []float32
	done := 0
	for _, target := range checkpoints {
		for done < target {
			v := trace()
			a.AddFrame([]float32{v, v, v})
			done++
		}
		r, _, _ := a.At(0, 0)
		errs = append(errs, float32(math.Abs(float64(r-groundTruth))))
	}

	t.Logf("errors at checkpoints %v: %v", checkpoints, errs)
	if last := errs[len(errs)-1]; last > 0.03 {
		t.Errorf("after %d real hardware traces, coverage estimate error = %v, want < 0.03 (ground truth %v)",
			checkpoints[len(checkpoints)-1], last, groundTruth)
	}
	if got := a.SampleCount(); got != checkpoints[len(checkpoints)-1] {
		t.Errorf("SampleCount() = %d, want %d", got, checkpoints[len(checkpoints)-1])
	}

	// A camera move (or scene/material edit) must observably reset the buffer.
	if reset := a.SyncState(renderer.AccumulationState{Camera: 1}); !reset {
		t.Error("SyncState with a new camera fingerprint did not report a reset")
	}
	if got := a.SampleCount(); got != 0 {
		t.Errorf("SampleCount() after a camera-change SyncState = %d, want 0", got)
	}
}

func sub(a, b [3]float32) [3]float32 { return [3]float32{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func normalize(v [3]float32) [3]float32 {
	n := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
	return [3]float32{v[0] / n, v[1] / n, v[2] / n}
}
