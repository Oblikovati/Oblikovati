// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/exchange"
)

// cloudProgressSink records the progress ticks a scan reader reports (a named fake per the house
// rules).
type cloudProgressSink struct {
	stages []string
	dones  []int
}

func (c *cloudProgressSink) fn(stage string, done, _ int) bool {
	c.stages = append(c.stages, stage)
	c.dones = append(c.dones, done)
	return false
}

// TestReadScanReportsProgress checks the point-cloud read threads the shared progress seam (#1647):
// ReadScan fires a point-count tick reaching the final record count.
func TestReadScanReportsProgress(t *testing.T) {
	var sink cloudProgressSink
	pts, _, err := ReadScan("scan.xyz", []byte("0 0 0\n1 0 0\n0 1 0\n"), exchange.TranslationOptions{Progress: sink.fn})
	if err != nil {
		t.Fatalf("ReadScan: %v", err)
	}
	if len(sink.dones) == 0 {
		t.Fatal("the scan reader reported no progress; the seam is not wired")
	}
	if last := sink.dones[len(sink.dones)-1]; last != len(pts) {
		t.Errorf("final progress done = %d, want the record count %d", last, len(pts))
	}
}

// TestReadScanCancels checks a first-call cancel aborts the scan read (before decoding) with an
// ErrCancelled-wrapping error.
func TestReadScanCancels(t *testing.T) {
	cancel := func(string, int, int) bool { return true }
	_, _, err := ReadScan("scan.xyz", []byte("0 0 0\n1 0 0\n"), exchange.TranslationOptions{Progress: cancel})
	if !errors.Is(err, exchange.ErrCancelled) {
		t.Fatalf("cancelled scan read error = %v, want it to wrap exchange.ErrCancelled", err)
	}
}
