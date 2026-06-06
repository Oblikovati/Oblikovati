// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati/api/wire"
	"oblikovati/app"
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

// marshalUndoState renders the active document's cursor state into the wire DTO.
func marshalUndoState(s *app.Session) (json.RawMessage, error) {
	return json.Marshal(wire.UndoState{
		CanUndo:  s.CanUndo(),
		CanRedo:  s.CanRedo(),
		NextUndo: s.UndoLabel(),
		NextRedo: s.RedoLabel(),
	})
}
