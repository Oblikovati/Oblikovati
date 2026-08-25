// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "fmt"

// Accumulator holds a persistent per-pixel running-sum buffer for progressive
// path-trace convergence (M45-F04 PBI-347, ADR-0053): repeated AddFrame calls average
// samples toward the converged image while the viewport is idle, and Reset starts a
// fresh convergence pass. Pure Go, no GPU dependency, per ADR-0014 — the same
// backend-agnostic seam [BLASCache] and [BuildBVH] already use, so the accumulation
// math is unit-testable independent of which intersection backend produced the samples.
type Accumulator struct {
	width, height int
	sum           []float32 // len 3*width*height, row-major running RGB sum
	samples       int

	haveState bool
	lastState AccumulationState
}

// NewAccumulator returns an empty width×height accumulator (SampleCount 0).
func NewAccumulator(width, height int) *Accumulator {
	return &Accumulator{width: width, height: height, sum: make([]float32, 3*width*height)}
}

// Width and Height are the buffer's fixed pixel dimensions.
func (a *Accumulator) Width() int  { return a.width }
func (a *Accumulator) Height() int { return a.height }

// SampleCount is the number of frames accumulated since the last Reset — the raw input
// for the F05 convergence indicator (PBI-350).
func (a *Accumulator) SampleCount() int { return a.samples }

// AddFrame adds one newly traced sample of every pixel into the running sum and
// advances SampleCount by one. frame is row-major RGB, length 3*Width()*Height() —
// one radiance sample per pixel, e.g. one dispatch of swpathtrace.comp or the hardware
// pathtrace pipeline.
func (a *Accumulator) AddFrame(frame []float32) {
	if len(frame) != len(a.sum) {
		panic(fmt.Sprintf("renderer: Accumulator.AddFrame: frame has %d floats, want %d (%dx%d RGB)",
			len(frame), len(a.sum), a.width, a.height))
	}
	for i, v := range frame {
		a.sum[i] += v
	}
	a.samples++
}

// At returns pixel (x,y)'s converged-so-far average color: the running sum divided by
// SampleCount, or black before the first sample.
func (a *Accumulator) At(x, y int) (r, g, b float32) {
	if a.samples == 0 {
		return 0, 0, 0
	}
	if x < 0 || x >= a.width || y < 0 || y >= a.height {
		panic(fmt.Sprintf("renderer: Accumulator.At: pixel (%d,%d) out of bounds for %dx%d buffer", x, y, a.width, a.height))
	}
	i := (y*a.width + x) * 3
	n := float32(a.samples)
	return a.sum[i] / n, a.sum[i+1] / n, a.sum[i+2] / n
}

// Reset clears the buffer and sample count, starting a fresh convergence pass.
func (a *Accumulator) Reset() {
	for i := range a.sum {
		a.sum[i] = 0
	}
	a.samples = 0
}

// AccumulationState is the minimal fingerprint SyncState compares call-to-call to
// decide whether the accumulated image is still valid: a camera move, a scene
// recompute, or a material edit changes one of these fields and invalidates the
// buffer. Computing the actual hashes (camera transform, recompute generation,
// material generation) is the F05 wiring PBI's job (PBI-350, "wire kRealisticRendering
// end to end") — this type only defines the comparison contract those hashes feed.
type AccumulationState struct {
	Camera   uint64
	Scene    uint64
	Material uint64
}

// SyncState resets the accumulator if state differs from the state passed to the
// previous SyncState call (or if this is the first call ever made), and reports
// whether it did. Call once per frame before AddFrame with the current
// camera/scene/material fingerprint.
func (a *Accumulator) SyncState(state AccumulationState) (didReset bool) {
	if a.haveState && state == a.lastState {
		return false
	}
	a.Reset()
	a.lastState = state
	a.haveState = true
	return true
}
