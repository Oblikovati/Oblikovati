// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"testing"

	"oblikovati.org/command"
	"oblikovati.org/model/doc"
)

// activeLog returns the active document's undo snapshot log — the delta stream #1424 records
// edits into, the observable the memory tests read.
func activeLog(t *testing.T, s *Session) *command.SnapshotLog {
	t.Helper()
	dh, ok := s.histories[s.ActiveDocument().ID()]
	if !ok {
		t.Fatal("active document has no transaction stream")
	}
	return dh.log
}

// TestUndoStreamStoresDeltasNotFullSnapshots is the issue's headline memory guarantee: recording
// many edits costs far less than the two-full-snapshots-per-edit storage it replaces. Each added
// parameter changes a localised region of the recipe, so the stream must retain deltas — well
// under the naive O(edits × recipeSize) the old RecipeEvent before+after held.
func TestUndoStreamStoresDeltasNotFullSnapshots(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	trackFromHere(s)
	def := partOf(t, s)

	const edits = 96
	for i := 0; i < edits; i++ {
		if err := s.AddNumericUserParameter(fmt.Sprintf("d%d", i), "10 mm"); err != nil {
			t.Fatalf("add param %d: %v", i, err)
		}
	}

	final, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	naive := edits * len(final) // the old stream held ~one full snapshot per edit (before==prev after)
	retained := activeLog(t, s).RetainedBytes()
	if retained >= naive/4 {
		t.Errorf("undo stream retained %d bytes for %d edits of ~%d-byte recipes; not sub-linear vs the %d-byte naive store",
			retained, edits, len(final), naive)
	}
}

// TestUndoRedoParityThroughDeltaLog: undoing every edit back to the baseline then redoing every
// edit restores the identical model state, proving the delta-log reconstruction is lossless at
// every cursor position (the parity acceptance criterion).
func TestUndoRedoParityThroughDeltaLog(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	trackFromHere(s)
	def := partOf(t, s)
	base := def.Parameters().Count()

	const edits = 40 // spans multiple checkpoint intervals so reconstruction replays real deltas
	for i := 0; i < edits; i++ {
		if err := s.AddNumericUserParameter(fmt.Sprintf("p%d", i), fmt.Sprintf("%d mm", i+1)); err != nil {
			t.Fatalf("add param %d: %v", i, err)
		}
	}
	if def.Parameters().Count() != base+edits {
		t.Fatalf("after edits: %d params, want %d", def.Parameters().Count(), base+edits)
	}

	for i := 0; i < edits; i++ {
		if err := s.Undo(); err != nil {
			t.Fatalf("undo %d: %v", i, err)
		}
	}
	if def.Parameters().Count() != base {
		t.Errorf("after undoing all edits: %d params, want the baseline %d", def.Parameters().Count(), base)
	}

	for i := 0; i < edits; i++ {
		if err := s.Redo(); err != nil {
			t.Fatalf("redo %d: %v", i, err)
		}
	}
	if def.Parameters().Count() != base+edits {
		t.Errorf("after redoing all edits: %d params, want %d", def.Parameters().Count(), base+edits)
	}
	for i := 0; i < edits; i++ {
		if _, ok := def.Parameters().ByName(fmt.Sprintf("p%d", i)); !ok {
			t.Errorf("redo lost parameter p%d", i)
		}
	}
}

// TestReclaimAuditLogsTrimsAndDiscards pins the session-audit bound's bookkeeping (#1424): a
// document with surviving events keeps its log trimmed to the earliest live position; a document
// whose events all aged out has its log discarded. Tested directly so it needs no 2000-event run.
func TestReclaimAuditLogsTrimsAndDiscards(t *testing.T) {
	keptDoc, goneDoc := doc.ID(1), doc.ID(2)
	audit := map[doc.ID]*command.SnapshotLog{
		keptDoc: command.NewSnapshotLogEvery(2),
		goneDoc: command.NewSnapshotLogEvery(2),
	}
	for n := 0; n < 8; n++ {
		audit[keptDoc].Append([]byte(fmt.Sprintf("kept-%d", n)))
	}
	for n := 0; n < 4; n++ {
		audit[goneDoc].Append([]byte(fmt.Sprintf("gone-%d", n)))
	}
	// Surviving events reference only keptDoc, earliest position 5; goneDoc has no live event.
	live := []sessionTxEvent{
		{docID: keptDoc, pos: 5},
		{docID: keptDoc, pos: 7},
		{docID: keptDoc, pos: -1}, // a no-recipe step must not affect trimming
	}

	reclaimAuditLogs(live, audit)

	if _, ok := audit[goneDoc]; ok {
		t.Error("a document with no surviving events kept its audit log")
	}
	kept, ok := audit[keptDoc]
	if !ok {
		t.Fatal("a referenced document lost its audit log")
	}
	if _, err := kept.At(5); err != nil {
		t.Errorf("earliest live position 5 no longer reconstructs after trim: %v", err)
	}
	got, err := kept.At(7)
	if err != nil || string(got) != "kept-7" {
		t.Errorf("At(7) after trim = %q, %v; want kept-7", got, err)
	}
	if _, err := kept.At(4); err == nil {
		t.Error("position 4 should have been reclaimed (before the earliest live position)")
	}
}
