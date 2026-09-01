// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestBoundedTransactionCoalescesEdits checks transaction.begin/end fold several recording
// edits into a single, named undo step — the team-shared-undo primitive of ADR-0005.
func TestBoundedTransactionCoalescesEdits(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}

	call(t, r, s, "transaction.begin", `{"label":"batch"}`, nil)
	if err := s.AddNumericUserParameter("h", "3 cm"); err != nil {
		t.Fatalf("add h: %v", err)
	}
	if err := s.AddNumericUserParameter("w", "4 cm"); err != nil {
		t.Fatalf("add w: %v", err)
	}

	// Inside the open transaction, nothing is undoable yet (recording is deferred).
	var st wire.UndoState
	call(t, r, s, "transaction.state", "{}", &st)
	if st.CanUndo {
		t.Fatalf("inside open transaction nothing should be undoable, got %+v", st)
	}

	var end wire.UndoState
	call(t, r, s, "transaction.end", "{}", &end)
	if !end.CanUndo || end.NextUndo != "batch" {
		t.Fatalf("after end want one undoable step 'batch', got %+v", end)
	}

	// A single undo reverts the whole batch.
	call(t, r, s, "transaction.undo", "{}", &st)
	if st.CanUndo {
		t.Fatalf("one undo should revert the whole batch, got %+v", st)
	}
}

// TestEndTransactionWithoutBeginErrors guards the unbalanced case.
func TestEndTransactionWithoutBeginErrors(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if _, err := r.Handle(s, "transaction.end", nil); err == nil {
		t.Fatal("expected an error ending with no open transaction")
	}
}

// TestAbortTransactionDiscardsBatchOverWire: transaction.abort reverts the open
// batch instead of committing it — the failed-batch escape hatch (M04-F05).
func TestAbortTransactionDiscardsBatchOverWire(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}

	call(t, r, s, "transaction.begin", `{"label":"batch"}`, nil)
	if err := s.AddNumericUserParameter("h", "3 cm"); err != nil {
		t.Fatalf("add h: %v", err)
	}

	var st wire.UndoState
	call(t, r, s, "transaction.abort", "{}", &st)
	if st.CanUndo {
		t.Fatalf("an aborted batch must record nothing, got %+v", st)
	}
	if err := s.AddNumericUserParameter("h", "5 cm"); err != nil {
		t.Fatalf("re-adding h after abort must work (the batch was reverted): %v", err)
	}

	if _, err := r.Handle(s, "transaction.abort", nil); err == nil {
		t.Fatal("expected an error aborting with no open transaction")
	}
}
