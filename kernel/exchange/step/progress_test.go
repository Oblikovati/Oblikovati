// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/exchange"
)

// stepProgressSink records the progress ticks the STEP reader reports (a named fake per the house
// rules).
type stepProgressSink struct {
	stages []string
	dones  []int
}

func (s *stepProgressSink) fn(stage string, done, _ int) bool {
	s.stages = append(s.stages, stage)
	s.dones = append(s.dones, done)
	return false
}

// TestImportSolidsReportsProgress checks the STEP reader threads the shared progress seam (#1647):
// it fires a per-solid tick with a monotonically non-decreasing done.
func TestImportSolidsReportsProgress(t *testing.T) {
	var sink stepProgressSink
	_, _, err := Reader{}.ImportSolids(readFixture(t, "cube.step"), exchange.TranslationOptions{Progress: sink.fn})
	if err != nil {
		t.Fatalf("ImportSolids: %v", err)
	}
	if len(sink.dones) == 0 {
		t.Fatal("the STEP reader reported no progress; the seam is not wired")
	}
	for i := 1; i < len(sink.dones); i++ {
		if sink.dones[i] < sink.dones[i-1] {
			t.Errorf("progress done went backwards: %v", sink.dones)
		}
	}
}

// TestImportSolidsCancels checks a cancelling ProgressFunc aborts the STEP import with an
// ErrCancelled-wrapping error.
func TestImportSolidsCancels(t *testing.T) {
	cancel := func(string, int, int) bool { return true }
	_, _, err := Reader{}.ImportSolids(readFixture(t, "cube.step"), exchange.TranslationOptions{Progress: cancel})
	if !errors.Is(err, exchange.ErrCancelled) {
		t.Fatalf("cancelled STEP import error = %v, want it to wrap exchange.ErrCancelled", err)
	}
}
