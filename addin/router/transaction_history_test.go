// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strconv"
	"testing"

	"oblikovati.org/api/wire"
)

// TestTransactionHistoryAndJumpOverTheAPI drives the history-browser surface: a feature
// created over the wire is one step in transaction.history, jumpTo moves the cursor to an
// absolute position non-destructively, and an explicit document id targets that document
// without activating it (the multi-document side-by-side case).
func TestTransactionHistoryAndJumpOverTheAPI(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"distance":"5 mm"}}`, nil)

	var h wire.TransactionHistory
	call(t, r, s, "transaction.history", "{}", &h)
	if h.Position != 1 || len(h.Entries) != 1 || h.Entries[0].Label != "Add Feature" {
		t.Fatalf("history = %+v, want position 1 with one Add Feature entry", h)
	}
	if h.Document == 0 || h.Name == "" {
		t.Fatalf("history omitted document identity: %+v", h)
	}

	// Jump back to the baseline; the stream stays intact (non-destructive).
	call(t, r, s, "transaction.jumpTo", `{"position":0}`, &h)
	if h.Position != 0 || len(h.Entries) != 1 {
		t.Fatalf("after jumpTo 0: %+v, want position 0 with the entry still present", h)
	}

	// Targeting by explicit document id returns the same stream (no activation needed).
	id := uint64(s.ActiveDocument().ID())
	var byID wire.TransactionHistory
	call(t, r, s, "transaction.history", `{"document":`+strconv.FormatUint(id, 10)+`}`, &byID)
	if byID.Document != id {
		t.Fatalf("history by id returned document %d, want %d", byID.Document, id)
	}
}

// TestTransactionJumpToRejectsOutOfRange: an out-of-range position is a clean error.
func TestTransactionJumpToRejectsOutOfRange(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "transaction.jumpTo", []byte(`{"position":99}`)); err == nil {
		t.Fatal("jumpTo past the end should error")
	}
}
