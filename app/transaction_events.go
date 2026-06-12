// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// The transaction lifecycle event set (M04-F05, Oblikovati#613): every move of a
// document's transaction stream, on the session bus. Commit/undo/redo fire in the
// Before phase (vetoable) and then, if not vetoed, the After phase; abort and
// delete are observations only, matching the reference event set. Distinct from
// the coalesced [TransactionChanged] (no payload, fires once per cursor move):
// these carry which step moved and how, so an add-in can keep external state
// (caches, overlays, exported data) consistent with undo/redo.
//
// Stable TypeIDs continue the session-bus block; never renumber.
const (
	tidTransactionCommitted event.TypeID = 0x0513
	tidTransactionUndone    event.TypeID = 0x0514
	tidTransactionRedone    event.TypeID = 0x0515
	tidTransactionAborted   event.TypeID = 0x0516
	tidTransactionDeleted   event.TypeID = 0x0517
)

// TransactionCommitted fires around one edit being recorded as an undo step.
// A Before handler may veto, which reverts the model to the pre-edit snapshot
// and turns the commit into an abort.
type TransactionCommitted struct {
	Document doc.ID
	Label    string
}

// EventID implements event.Event.
func (TransactionCommitted) EventID() event.TypeID { return tidTransactionCommitted }

// TransactionUndone fires around undo. A Before handler may veto the move
// (e.g. external state cannot roll back); Label names the step undone.
type TransactionUndone struct {
	Document doc.ID
	Label    string
}

// EventID implements event.Event.
func (TransactionUndone) EventID() event.TypeID { return tidTransactionUndone }

// TransactionRedone fires around redo; a Before handler may veto the move.
type TransactionRedone struct {
	Document doc.ID
	Label    string
}

// EventID implements event.Event.
func (TransactionRedone) EventID() event.TypeID { return tidTransactionRedone }

// TransactionAborted fires (After) when a transaction is discarded instead of
// committed: an explicit [Session.AbortTransaction], or a commit a Before
// handler vetoed. The model has already reverted to the pre-transaction state.
type TransactionAborted struct {
	Document doc.ID
	Label    string
}

// EventID implements event.Event.
func (TransactionAborted) EventID() event.TypeID { return tidTransactionAborted }

// TransactionDeleted fires (After) when a document's whole transaction stream is
// discarded — the document closed and its undo steps are gone.
type TransactionDeleted struct{ Document doc.ID }

// EventID implements event.Event.
func (TransactionDeleted) EventID() event.TypeID { return tidTransactionDeleted }
