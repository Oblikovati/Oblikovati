// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/app"
)

// TestTransactionUndoRedoOverTheAPI drives the transaction.* control surface against a
// live session: an edit makes undo available, transaction.undo moves the cursor back
// (and exposes redo), transaction.redo moves it forward. NewPart opens the stream at the
// empty-part baseline, and AddNumericUserParameter is a recording Session edit.
func TestTransactionUndoRedoOverTheAPI(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}

	var st wire.UndoState
	call(t, r, s, "transaction.state", "{}", &st)
	if st.CanUndo || st.CanRedo {
		t.Fatalf("fresh part state = %+v, want nothing to undo/redo", st)
	}

	if err := s.AddNumericUserParameter("h", "3 cm"); err != nil {
		t.Fatalf("add param: %v", err)
	}
	call(t, r, s, "transaction.state", "{}", &st)
	if !st.CanUndo || st.NextUndo != "Edit Parameters" {
		t.Fatalf("after edit state = %+v, want canUndo with nextUndo=Edit Parameters", st)
	}

	call(t, r, s, "transaction.undo", "{}", &st)
	if st.CanUndo || !st.CanRedo || st.NextRedo != "Edit Parameters" {
		t.Fatalf("after undo state = %+v, want canRedo with nextRedo=Edit Parameters", st)
	}

	call(t, r, s, "transaction.redo", "{}", &st)
	if !st.CanUndo || st.CanRedo {
		t.Fatalf("after redo state = %+v, want canUndo only", st)
	}
}

// TestTransactionUndoNoActiveDocument: undo with no document errors cleanly (not a panic).
func TestTransactionUndoNoActiveDocument(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := r.Handle(s, "transaction.undo", nil); err == nil {
		t.Fatal("expected an error undoing with no active document")
	}
}
