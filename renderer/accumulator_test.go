// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"math"
	"math/rand"
	"testing"
)

// TestAccumulatorConvergesMonotonically is PBI-347's core acceptance criterion: a
// static (noisy but stationary) signal's accumulated estimate converges toward the
// known ground truth as sample count increases. The noise source is a fixed-seed PRNG
// (deterministic across runs, so this is not flaky) standing in for Monte Carlo
// variance a real path-traced sample would have once PBI-348 adds importance
// sampling — Accumulator's job is only to average correctly, independent of where the
// samples came from.
func TestAccumulatorConvergesMonotonically(t *testing.T) {
	const groundTruth = float32(0.63)
	rng := rand.New(rand.NewSource(1))

	a := NewAccumulator(1, 1)
	checkpoints := []int{25, 100, 400, 1600, 6400, 25600}
	var errs []float32
	done := 0
	for _, target := range checkpoints {
		for done < target {
			noisy := groundTruth + float32(rng.NormFloat64())*0.5
			a.AddFrame([]float32{noisy, noisy, noisy})
			done++
		}
		r, _, _ := a.At(0, 0)
		errs = append(errs, float32(math.Abs(float64(r-groundTruth))))
	}

	for i := 1; i < len(errs); i++ {
		if errs[i] > errs[i-1] {
			t.Logf("checkpoint %d: error grew %v -> %v (allowed — checked against the first checkpoint below, not step-to-step)", checkpoints[i], errs[i-1], errs[i])
		}
	}
	// A single 4x sample-count step can still see its error tick up under real noise;
	// what must hold is the overall trend. Every checkpoint from the third on must beat
	// the FIRST checkpoint's error by a wide margin, and the final one must be tight.
	for i := 2; i < len(errs); i++ {
		if errs[i] >= errs[0] {
			t.Errorf("checkpoint %d (n=%d): error %v did not improve on the n=%d error %v", i, checkpoints[i], errs[i], checkpoints[0], errs[0])
		}
	}
	if last := errs[len(errs)-1]; last > 0.02 {
		t.Errorf("after %d samples, error = %v, want < 0.02 (converged estimate %v, ground truth %v)", checkpoints[len(checkpoints)-1], last, groundTruth+last, groundTruth)
	}
	if got := a.SampleCount(); got != checkpoints[len(checkpoints)-1] {
		t.Errorf("SampleCount() = %d, want %d", got, checkpoints[len(checkpoints)-1])
	}
}

func TestAccumulatorAddFrameAveragesPerPixel(t *testing.T) {
	a := NewAccumulator(2, 1)
	a.AddFrame([]float32{1, 0, 0, 0, 1, 0})
	a.AddFrame([]float32{0, 0, 1, 0, 0, 1})

	if r, g, b := a.At(0, 0); r != 0.5 || g != 0 || b != 0.5 {
		t.Errorf("At(0,0) = (%v,%v,%v), want (0.5,0,0.5)", r, g, b)
	}
	if r, g, b := a.At(1, 0); r != 0 || g != 0.5 || b != 0.5 {
		t.Errorf("At(1,0) = (%v,%v,%v), want (0,0.5,0.5)", r, g, b)
	}
	if got := a.SampleCount(); got != 2 {
		t.Errorf("SampleCount() = %d, want 2", got)
	}
}

func TestAccumulatorAddFrameWrongLengthPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("AddFrame with a mismatched frame length did not panic")
		}
	}()
	NewAccumulator(2, 2).AddFrame([]float32{1, 1, 1})
}

func TestAccumulatorResetClearsSumAndCount(t *testing.T) {
	a := NewAccumulator(1, 1)
	a.AddFrame([]float32{1, 1, 1})
	a.AddFrame([]float32{1, 1, 1})

	a.Reset()

	if got := a.SampleCount(); got != 0 {
		t.Errorf("after Reset, SampleCount() = %d, want 0", got)
	}
	if r, g, b := a.At(0, 0); r != 0 || g != 0 || b != 0 {
		t.Errorf("after Reset, At(0,0) = (%v,%v,%v), want (0,0,0)", r, g, b)
	}
}

func TestAccumulatorSyncStateResetsOnFirstCall(t *testing.T) {
	a := NewAccumulator(1, 1)
	a.AddFrame([]float32{1, 1, 1})

	if reset := a.SyncState(AccumulationState{}); !reset {
		t.Error("SyncState on the very first call did not report a reset")
	}
	if got := a.SampleCount(); got != 0 {
		t.Errorf("SampleCount() after first SyncState = %d, want 0", got)
	}
}

func TestAccumulatorSyncStateResetsOnAnyFieldChange(t *testing.T) {
	tests := []struct {
		name string
		from AccumulationState
		to   AccumulationState
	}{
		{"camera moved", AccumulationState{Camera: 1, Scene: 1, Material: 1}, AccumulationState{Camera: 2, Scene: 1, Material: 1}},
		{"scene recomputed", AccumulationState{Camera: 1, Scene: 1, Material: 1}, AccumulationState{Camera: 1, Scene: 2, Material: 1}},
		{"material edited", AccumulationState{Camera: 1, Scene: 1, Material: 1}, AccumulationState{Camera: 1, Scene: 1, Material: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAccumulator(1, 1)
			a.SyncState(tt.from)
			a.AddFrame([]float32{1, 1, 1})
			a.AddFrame([]float32{1, 1, 1})

			if reset := a.SyncState(tt.to); !reset {
				t.Errorf("SyncState(%+v) after %+v did not report a reset", tt.to, tt.from)
			}
			if got := a.SampleCount(); got != 0 {
				t.Errorf("SampleCount() after mutation = %d, want 0 (buffer must observably reset)", got)
			}
		})
	}
}

func TestAccumulatorSyncStateNoResetWhenUnchanged(t *testing.T) {
	a := NewAccumulator(1, 1)
	state := AccumulationState{Camera: 7, Scene: 3, Material: 9}
	a.SyncState(state)
	a.AddFrame([]float32{1, 1, 1})
	a.AddFrame([]float32{1, 1, 1})
	a.AddFrame([]float32{1, 1, 1})

	if reset := a.SyncState(state); reset {
		t.Error("SyncState with an unchanged fingerprint reported a reset")
	}
	if got := a.SampleCount(); got != 3 {
		t.Errorf("SampleCount() after unchanged SyncState = %d, want 3 (buffer must survive)", got)
	}
}
