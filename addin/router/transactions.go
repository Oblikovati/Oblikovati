// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// undoTransaction moves the active document's cursor back one transaction event, then
// returns the resulting undo/redo state.
func undoTransaction(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	if err := s.Undo(); err != nil {
		return nil, err
	}
	return marshalUndoState(s)
}

// redoTransaction moves the cursor forward one transaction event, then returns the state.
func redoTransaction(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	if err := s.Redo(); err != nil {
		return nil, err
	}
	return marshalUndoState(s)
}

// transactionState reports what undo/redo can currently do (a read-only query).
func transactionState(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return marshalUndoState(s)
}

// beginTransaction opens a bounded transaction so a batch of edits coalesces into one
// undo step (wire.MethodTransactionBegin).
func beginTransaction(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.TransactionBeginArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.BeginTransaction(a.Label); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// endTransaction closes the innermost open transaction and returns the resulting state
// (wire.MethodTransactionEnd).
func endTransaction(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	if err := s.EndTransaction(); err != nil {
		return nil, err
	}
	return marshalUndoState(s)
}

// abortTransaction discards the open bounded transaction — the model reverts to the
// group's pre-Begin state and no undo step is recorded (wire.MethodTransactionAbort,
// M04-F05): an add-in whose batch failed partway does not leave the document half-edited.
func abortTransaction(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	if err := s.AbortTransaction(); err != nil {
		return nil, err
	}
	return marshalUndoState(s)
}

// transactionHistory reads one open document's whole undo stream for a history browser
// (wire.MethodTransactionHistory) — a read-only query that does not activate the document, so
// several documents' timelines can be shown side by side.
func transactionHistory(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.TransactionHistoryArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	id, err := historyTargetDoc(s, a.Document)
	if err != nil {
		return nil, err
	}
	return marshalHistory(s, id)
}

// jumpTransaction moves one document's undo cursor to an absolute position
// (wire.MethodTransactionJumpTo) and returns its resulting timeline.
func jumpTransaction(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.TransactionJumpToArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	id, err := historyTargetDoc(s, a.Document)
	if err != nil {
		return nil, err
	}
	if err := s.JumpDocumentTo(id, a.Position); err != nil {
		return nil, err
	}
	return marshalHistory(s, id)
}

// historyTargetDoc resolves the document a history request targets: the given id, or the
// active document when id is 0. It errors when 0 is asked for but no document is active.
func historyTargetDoc(s *app.Session, id uint64) (doc.ID, error) {
	if id != 0 {
		return doc.ID(id), nil
	}
	d := s.ActiveDocument()
	if d == nil {
		return 0, app.ErrNoActiveDoc
	}
	return d.ID(), nil
}

// marshalHistory renders one document's timeline into the wire DTO. A save checkpoint at depth
// k flags the entry at index k-1 (the step after which the document was written to disk).
func marshalHistory(s *app.Session, id doc.ID) (json.RawMessage, error) {
	tl, ok := s.DocumentHistoryView(id)
	if !ok {
		return nil, fmt.Errorf("transaction.history: document %d is not an open model document", uint64(id))
	}
	saved := make(map[int]bool, len(tl.SavedDepths))
	for _, d := range tl.SavedDepths {
		saved[d] = true
	}
	entries := make([]wire.TransactionHistoryEntry, len(tl.Labels))
	for i, label := range tl.Labels {
		entries[i] = wire.TransactionHistoryEntry{Label: label, Saved: saved[i+1]}
	}
	return json.Marshal(wire.TransactionHistory{
		Document: uint64(id),
		Name:     tl.Name,
		Position: tl.Position,
		Entries:  entries,
	})
}

// marshalUndoState renders the active document's cursor state into the wire DTO.
func marshalUndoState(s *app.Session) (json.RawMessage, error) {
	return json.Marshal(wire.UndoState{
		CanUndo:  s.CanUndo(),
		CanRedo:  s.CanRedo(),
		NextUndo: s.UndoLabel(),
		NextRedo: s.RedoLabel(),
	})
}
