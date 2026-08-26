// SPDX-License-Identifier: GPL-2.0-only

package command

import (
	"fmt"
	"testing"
)

// snap builds a deterministic, mostly-shared snapshot for position n: a large constant body with
// a small varying tail, mimicking how a real edit changes a localised region of the recipe.
func snap(n int) []byte {
	body := make([]byte, 0, 4096)
	for i := range 256 {
		body = append(body, []byte(fmt.Sprintf("feature%d;", i))...)
	}
	return append(body, []byte(fmt.Sprintf("|edit=%d", n))...)
}

func TestSnapshotLogReconstructsEveryPositionAcrossCheckpoints(t *testing.T) {
	l := NewSnapshotLogEvery(4) // force frequent checkpoints + deltas to exercise both paths
	const positions = 21
	for n := range positions {
		if got := l.Append(snap(n)); got != n {
			t.Fatalf("Append returned position %d, want %d", got, n)
		}
	}
	// Every position must reconstruct byte-for-byte — checkpoints and the deltas between them.
	for n := range positions {
		got, err := l.At(n)
		if err != nil {
			t.Fatalf("At(%d): %v", n, err)
		}
		if want := snap(n); string(got) != string(want) {
			t.Fatalf("At(%d) reconstructed the wrong snapshot", n)
		}
	}
}

func TestSnapshotLogAtRejectsOutOfRange(t *testing.T) {
	l := NewSnapshotLog()
	l.Append(snap(0))
	if _, err := l.At(1); err == nil {
		t.Error("At past the high-water mark returned no error")
	}
	if _, err := l.At(-1); err == nil {
		t.Error("At a negative position returned no error")
	}
}

// TestSnapshotLogTruncateToDropsRedoBranch mirrors the undo stream: undo moves the cursor back,
// a new edit truncates the forward positions, and the reused position reconstructs the new state.
func TestSnapshotLogTruncateToDropsRedoBranch(t *testing.T) {
	l := NewSnapshotLogEvery(4)
	for n := range 6 {
		l.Append(snap(n))
	}
	l.TruncateTo(3) // keep positions 0,1,2 (the cursor undid back to position 2)
	if l.Next() != 3 {
		t.Fatalf("after TruncateTo(3) Next = %d, want 3", l.Next())
	}
	if got := l.Append([]byte("new-branch")); got != 3 {
		t.Fatalf("re-appended position = %d, want 3 (reused slot)", got)
	}
	got, err := l.At(3)
	if err != nil || string(got) != "new-branch" {
		t.Fatalf("At(3) = %q, %v; want new-branch", got, err)
	}
	// The discarded position 4 must be gone.
	if _, err := l.At(4); err == nil {
		t.Error("position 4 survived truncation")
	}
}

// TestSnapshotLogTruncateFrontReclaimsAndKeepsPositionsStable is the session-audit bound: old
// positions are freed but surviving event references (by absolute position) still resolve.
func TestSnapshotLogTruncateFrontReclaimsAndKeepsPositionsStable(t *testing.T) {
	l := NewSnapshotLogEvery(4)
	for n := range 10 {
		l.Append(snap(n))
	}
	before := l.RetainedBytes()
	if err := l.TruncateFront(6); err != nil {
		t.Fatalf("TruncateFront(6): %v", err)
	}
	if l.RetainedBytes() >= before {
		t.Errorf("TruncateFront did not reclaim memory: before=%d after=%d", before, l.RetainedBytes())
	}
	// Position 6 onward keep their absolute index and reconstruct unchanged.
	for n := 6; n < 10; n++ {
		got, err := l.At(n)
		if err != nil {
			t.Fatalf("At(%d) after front-trim: %v", n, err)
		}
		if string(got) != string(snap(n)) {
			t.Fatalf("At(%d) wrong after front-trim", n)
		}
	}
	if _, err := l.At(5); err == nil {
		t.Error("position 5 survived front truncation")
	}
}

// TestSnapshotLogMemoryGrowsLinearly is the issue's headline guarantee: storing M localised edits
// of an O(R)-byte recipe costs O(M·editSize + M/interval·R), NOT O(M·R). With localised edits the
// retained bytes must stay far below the M-full-copies the old stream held.
func TestSnapshotLogMemoryGrowsLinearly(t *testing.T) {
	l := NewSnapshotLogEvery(32)
	const edits = 200
	recipeSize := len(snap(0))
	for n := range edits {
		l.Append(snap(n))
	}
	naive := edits * recipeSize // what two-full-snapshots-per-edit storage approximated
	if l.RetainedBytes() >= naive/4 {
		t.Errorf("retained %d bytes for %d edits of %d-byte recipes; not sub-linear vs the %d-byte naive store",
			l.RetainedBytes(), edits, recipeSize, naive)
	}
}

// BenchmarkSnapshotLogMemory documents the #1424 memory win: it records `edits` localised edits
// of an ~R-byte recipe and reports the bytes the delta log retains against the naive
// one-full-snapshot-per-edit storage it replaces. Run: go test ./command/ -run=^$ -bench=Memory.
// Measured (edits=1000, R≈2.7 KB): naive_bytes ≈ 2.71 MB, delta_bytes ≈ 0.10 MB — the stream
// holds ~26× less, and the ratio improves as the recipe grows (the per-edit O(N²) blow-up is
// gone; reconstruction stays bounded to one checkpoint interval).
func BenchmarkSnapshotLogMemory(b *testing.B) {
	const edits = 1000
	for i := 0; i < b.N; i++ {
		l := NewSnapshotLog()
		for n := range edits {
			l.Append(snap(n))
		}
		if i == 0 {
			b.ReportMetric(float64(edits*len(snap(0))), "naive_bytes")
			b.ReportMetric(float64(l.RetainedBytes()), "delta_bytes")
		}
	}
}
