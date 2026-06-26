// SPDX-License-Identifier: GPL-2.0-only

package command

import "fmt"

// defaultCheckpointInterval is how often the snapshot log stores a full checkpoint instead of a
// delta. Reconstructing a position replays at most this many deltas from the nearest checkpoint,
// and the retained checkpoints cost ~totalPositions/interval full snapshots — so the constant
// trades reconstruction time against memory. 32 keeps the undo stream's memory ~32× smaller than
// the old whole-recipe-per-edit storage (Oblikovati#1424) while bounding a restore to ≤32 delta
// applies; PR2's incremental recompute removes the per-restore cost that dominates either way.
const defaultCheckpointInterval = 32

// SnapshotLog is an append-only, sparsely-checkpointed delta stream of recipe snapshots indexed
// by position (Oblikovati#1424). Position 0 is the document's open-state baseline; each later
// position is one cursor state of the undo stream. A position is stored either as a full
// checkpoint (every checkpointEvery positions) or as a forward [snapshotDelta] from the previous
// position, so a stream of N edits costs O(N·editSize + N/interval·recipeSize) instead of the
// O(N·recipeSize) two-full-snapshots-per-edit it replaces.
//
// It is the storage behind [RecipeEvent]: the event holds only its before/after positions and
// reconstructs the bytes through [SnapshotLog.At] on undo/redo. The log is content-agnostic
// (part, assembly, drawing recipes are all just bytes), so every recipe store gets the saving.
type SnapshotLog struct {
	checkpointEvery int
	// entries holds the live positions in order; entries[0] is logical position baseOffset.
	// baseOffset advances when the front is trimmed (the session audit's bound), so positions
	// recorded in events stay stable even as old history is reclaimed.
	entries    []logEntry
	baseOffset int
}

// logEntry is one stored position: a full checkpoint, or a forward delta from the previous
// position (exactly one of the two is set).
type logEntry struct {
	checkpoint []byte         // the full snapshot; nil unless this position is a checkpoint
	delta      *snapshotDelta // forward patch from the previous position; nil at a checkpoint
}

// NewSnapshotLog returns an empty log with the default checkpoint spacing.
func NewSnapshotLog() *SnapshotLog { return NewSnapshotLogEvery(defaultCheckpointInterval) }

// NewSnapshotLogEvery returns an empty log that checkpoints every interval positions. An interval
// below 1 is clamped to 1 (every position a checkpoint — the safe, memory-heavy extreme), so a
// caller can never configure a log that fails to reconstruct.
func NewSnapshotLogEvery(interval int) *SnapshotLog {
	if interval < 1 {
		interval = 1
	}
	return &SnapshotLog{checkpointEvery: interval}
}

// IsEmpty reports whether no position has been appended yet (so the caller knows to seed the
// open-state baseline at position 0 before recording its first edit).
func (l *SnapshotLog) IsEmpty() bool { return len(l.entries) == 0 }

// Next returns the position the next [SnapshotLog.Append] will assign.
func (l *SnapshotLog) Next() int { return l.baseOffset + len(l.entries) }

// Append stores snap at the next position and returns that position. It is a checkpoint when the
// position is the log's first live entry or falls on the checkpoint interval; otherwise it is a
// forward delta from the previous position. snap is copied defensively so a caller reusing its
// buffer cannot corrupt the stream.
func (l *SnapshotLog) Append(snap []byte) int {
	pos := l.Next()
	if len(l.entries) == 0 || pos%l.checkpointEvery == 0 {
		l.entries = append(l.entries, logEntry{checkpoint: append([]byte(nil), snap...)})
		return pos
	}
	prev, _ := l.reconstruct(len(l.entries) - 1) // the just-stored previous entry always reconstructs
	d := makeSnapshotDelta(prev, snap)
	l.entries = append(l.entries, logEntry{delta: &d})
	return pos
}

// At reconstructs the snapshot at the given position by copying the nearest checkpoint at or
// before it and replaying the intervening deltas. It errors if the position is outside the live
// range (a stale event reference, or a position trimmed away), surfacing a stream bug instead of
// returning a wrong recipe.
func (l *SnapshotLog) At(pos int) ([]byte, error) {
	i := pos - l.baseOffset
	if i < 0 || i >= len(l.entries) {
		return nil, fmt.Errorf("command: snapshot position %d out of range [%d,%d)", pos, l.baseOffset, l.Next())
	}
	return l.reconstruct(i)
}

// reconstruct rebuilds the snapshot at the given slice index (not logical position).
func (l *SnapshotLog) reconstruct(i int) ([]byte, error) {
	start := i
	for start > 0 && l.entries[start].checkpoint == nil {
		start--
	}
	if l.entries[start].checkpoint == nil {
		return nil, fmt.Errorf("command: snapshot log has no checkpoint at or before index %d (corrupt stream)", i)
	}
	out := append([]byte(nil), l.entries[start].checkpoint...)
	for k := start + 1; k <= i; k++ {
		next, err := l.entries[k].delta.apply(out)
		if err != nil {
			return nil, err
		}
		out = next
	}
	return out, nil
}

// TruncateTo drops every position at or beyond highWater — the redo branch a new edit discards
// (the undo stream truncates in lockstep with the command history's done/undone stacks). A
// highWater at or above the current high-water mark is a no-op.
func (l *SnapshotLog) TruncateTo(highWater int) {
	keep := highWater - l.baseOffset
	if keep < 0 {
		keep = 0
	}
	if keep < len(l.entries) {
		l.entries = l.entries[:keep]
	}
}

// TruncateFront reclaims positions before keepFrom, re-baselining keepFrom as a fresh checkpoint
// so what remains still reconstructs. It is how the session audit honours its memory bound
// without copying whole recipes: positions recorded in surviving events keep their stable index
// (baseOffset advances), only the dropped prefix is freed. A keepFrom at or before the current
// front is a no-op; one beyond the live range is clamped to keep the newest position.
func (l *SnapshotLog) TruncateFront(keepFrom int) error {
	if keepFrom <= l.baseOffset || len(l.entries) == 0 {
		return nil
	}
	if keepFrom >= l.Next() {
		keepFrom = l.Next() - 1
	}
	rebased, err := l.At(keepFrom)
	if err != nil {
		return err
	}
	drop := keepFrom - l.baseOffset
	l.entries = l.entries[drop:]
	l.entries[0] = logEntry{checkpoint: rebased} // the new front must stand alone
	l.baseOffset = keepFrom
	return nil
}

// RetainedBytes is the total snapshot bytes the log holds (checkpoints in full, deltas as their
// differing middles) — the figure the O(N)-growth regression test and the memory benchmark read.
func (l *SnapshotLog) RetainedBytes() int {
	total := 0
	for _, e := range l.entries {
		if e.checkpoint != nil {
			total += len(e.checkpoint)
			continue
		}
		total += e.delta.size()
	}
	return total
}
