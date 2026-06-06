// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati/event"
	"oblikovati/model/doc"
)

// Application UI events on the session bus. Stable TypeIDs (0x05xx = M05 UI).
const (
	tidCommandStarted    event.TypeID = 0x0501
	tidCommandEnded      event.TypeID = 0x0502
	tidSelectionChanged  event.TypeID = 0x0503
	tidTransactionChange event.TypeID = 0x0504
)

// CommandStarted fires (Before) when a command begins — Inventor's
// OnExecute(Before). A handler could veto, though most observe.
type CommandStarted struct{ ID string }

// EventID implements event.Event.
func (CommandStarted) EventID() event.TypeID { return tidCommandStarted }

// CommandEnded fires (After) when a command finishes — OnExecute(After) /
// OnTerminate. Failed reports whether the command returned an error.
type CommandEnded struct {
	ID     string
	Failed bool
}

// EventID implements event.Event.
func (CommandEnded) EventID() event.TypeID { return tidCommandEnded }

// SelectionChanged fires (After) when the selection set changes — Inventor's
// SelectEvents / OnSelect.
type SelectionChanged struct{ Count int }

// EventID implements event.Event.
func (SelectionChanged) EventID() event.TypeID { return tidSelectionChanged }

// TransactionChanged fires (After) once per committed edit, undo, or redo on a
// document's event stream — the coalesced "the model moved" signal (Inventor's
// TransactionEvents). Document is the affected document's ID. The viewport reads model
// state each frame, so this is for observers/add-ins and dirty-state tracking, not the
// renderer.
type TransactionChanged struct{ Document doc.ID }

// EventID implements event.Event.
func (TransactionChanged) EventID() event.TypeID { return tidTransactionChange }
