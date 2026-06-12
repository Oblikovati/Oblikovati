// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/event"
)

// newPartSession is a session with one fresh part document — the smallest
// vehicle for transaction-stream events (a parameter edit is one undo step).
func newPartSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	return s
}

// TestCommitEmitsTransactionCommitted pins the happy path: one edit fires
// Before then After with the step's label and document id (M04-F05).
func TestCommitEmitsTransactionCommitted(t *testing.T) {
	s := newPartSession(t)
	var phases []event.Phase
	var got TransactionCommitted
	event.Subscribe(s.Events(), event.Before, func(c event.Context, e TransactionCommitted) event.Outcome {
		phases = append(phases, c.Phase)
		return event.Continue()
	})
	event.Subscribe(s.Events(), event.After, func(c event.Context, e TransactionCommitted) event.Outcome {
		phases = append(phases, c.Phase)
		got = e
		return event.Continue()
	})

	if err := s.AddNumericUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter: %v", err)
	}
	if len(phases) != 2 || phases[0] != event.Before || phases[1] != event.After {
		t.Fatalf("phases = %v, want [Before After]", phases)
	}
	if got.Label != "Edit Parameters" || got.Document != s.ActiveDocument().ID() {
		t.Errorf("committed event = %+v, want the Edit Parameters step on the active document", got)
	}
}

// TestVetoedCommitRevertsEditAndReportsAbort: a Before handler vetoing the
// commit rolls the model back to the pre-edit snapshot, records no undo step,
// and announces a TransactionAborted instead.
func TestVetoedCommitRevertsEditAndReportsAbort(t *testing.T) {
	s := newPartSession(t)
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e TransactionCommitted) event.Outcome {
		return event.Veto("policy: parameters are locked")
	})
	var aborted []TransactionAborted
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TransactionAborted) event.Outcome {
		aborted = append(aborted, e)
		return event.Continue()
	})

	if err := s.AddNumericUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter itself must not fail: %v", err)
	}
	if _, ok := partOf(t, s).Parameters().ByName("w"); ok {
		t.Error("the vetoed edit must be reverted, but parameter w survived")
	}
	if s.CanUndo() {
		t.Error("a vetoed commit must not record an undo step")
	}
	if len(aborted) != 1 || aborted[0].Label != "Edit Parameters" {
		t.Errorf("aborted events = %+v, want one for the Edit Parameters step", aborted)
	}
	if s.Notice() != "policy: parameters are locked" {
		t.Errorf("notice = %q, want the veto reason surfaced", s.Notice())
	}
}

// TestUndoRedoEmitEventsAndHonorVeto covers the navigation events: undo/redo
// fire with the moved step's label, and a Before veto blocks the move.
func TestUndoRedoEmitEventsAndHonorVeto(t *testing.T) {
	s := newPartSession(t)
	if err := s.AddNumericUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter: %v", err)
	}
	var undone []TransactionUndone
	var redone []TransactionRedone
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TransactionUndone) event.Outcome {
		undone = append(undone, e)
		return event.Continue()
	})
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TransactionRedone) event.Outcome {
		redone = append(redone, e)
		return event.Continue()
	})

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if err := s.Redo(); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if len(undone) != 1 || undone[0].Label != "Edit Parameters" {
		t.Errorf("undone events = %+v, want one Edit Parameters", undone)
	}
	if len(redone) != 1 || redone[0].Label != "Edit Parameters" {
		t.Errorf("redone events = %+v, want one Edit Parameters", redone)
	}

	// An external observer that cannot roll back vetoes the next undo: the
	// cursor must not move and the parameter must survive.
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e TransactionUndone) event.Outcome {
		return event.Veto("exported state cannot roll back")
	})
	if err := s.Undo(); err == nil {
		t.Fatal("a vetoed undo must return an error")
	}
	if _, ok := partOf(t, s).Parameters().ByName("w"); !ok {
		t.Error("a vetoed undo must not move the cursor (parameter w vanished)")
	}
	if len(undone) != 1 {
		t.Errorf("undone events after veto = %d, want still 1 (no After for a vetoed move)", len(undone))
	}
}

// TestAbortTransactionDiscardsTheGroup: an aborted bounded transaction reverts
// the whole batch, records nothing, and announces the abort with the group's
// label — the seam behind wire transaction.abort.
func TestAbortTransactionDiscardsTheGroup(t *testing.T) {
	s := newPartSession(t)
	if err := s.AbortTransaction(); err != ErrNoOpenTransaction {
		t.Fatalf("abort with nothing open = %v, want ErrNoOpenTransaction", err)
	}

	var aborted []TransactionAborted
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TransactionAborted) event.Outcome {
		aborted = append(aborted, e)
		return event.Continue()
	})
	if err := s.BeginTransaction("remote batch"); err != nil {
		t.Fatalf("BeginTransaction: %v", err)
	}
	if err := s.AddNumericUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter: %v", err)
	}
	if err := s.AbortTransaction(); err != nil {
		t.Fatalf("AbortTransaction: %v", err)
	}
	if _, ok := partOf(t, s).Parameters().ByName("w"); ok {
		t.Error("the aborted batch must be reverted, but parameter w survived")
	}
	if s.InTransaction() || s.CanUndo() {
		t.Errorf("after abort: InTransaction=%v CanUndo=%v, want both false", s.InTransaction(), s.CanUndo())
	}
	if len(aborted) != 1 || aborted[0].Label != "remote batch" {
		t.Errorf("aborted events = %+v, want one for the remote batch group", aborted)
	}
}

// TestDocumentCloseDeletesTransactionStream: closing a document discards its
// stream (no history leak) and announces TransactionDeleted.
func TestDocumentCloseDeletesTransactionStream(t *testing.T) {
	s := newPartSession(t)
	if err := s.AddNumericUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("AddNumericUserParameter: %v", err)
	}
	var deleted []TransactionDeleted
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TransactionDeleted) event.Outcome {
		deleted = append(deleted, e)
		return event.Continue()
	})

	d := s.ActiveDocument()
	if err := s.Workspace().Close(d, true); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(deleted) != 1 || deleted[0].Document != d.ID() {
		t.Fatalf("deleted events = %+v, want one for document %d", deleted, d.ID())
	}
	if _, ok := s.histories[d.ID()]; ok {
		t.Error("the closed document's history must be dropped from the session")
	}
}
