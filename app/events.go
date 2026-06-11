// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// Application UI events on the session bus. Stable TypeIDs (0x05xx = M05 UI).
const (
	tidCommandStarted       event.TypeID = 0x0501
	tidCommandEnded         event.TypeID = 0x0502
	tidSelectionChanged     event.TypeID = 0x0503
	tidTransactionChange    event.TypeID = 0x0504
	tidEditCommitted        event.TypeID = 0x0505
	tidBrowserPaneNode      event.TypeID = 0x0506 // app/browser_panes.go (M05-F03)
	tidDockableWinChanged   event.TypeID = 0x0507 // app/dockwindow_store.go (M05-F03)
	tidProgressCancelled    event.TypeID = 0x0508 // app/progress_ledger.go (M05-F09)
	tidBalloonClicked       event.TypeID = 0x0509 // app/balloon_tips.go (M05-F09)
	tidPromptAnswered       event.TypeID = 0x050a // app/prompt_center.go (M05-F09)
	tidMiniToolbarChanged   event.TypeID = 0x050b // app/minitoolbar_store.go (M05-F07)
	tidMiniToolbarCommitted event.TypeID = 0x050c // app/minitoolbar_store.go (M05-F07)
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

// EditCommitted fires (After) when a document mutation is applied through the method
// router, carrying the very wire request that produced it (Method + raw Args). It is the
// basis for operational replication in a collaboration add-in (oblikovati-meeting
// ADR-0004): a remote peer replays Method/Args through its own client. Distinct from
// TransactionChanged (which only signals "the model moved" with no payload).
//
// v1 scope: only router-path edits are emitted; edits made directly in the UI are not yet
// captured. Args is the raw JSON request bytes (may be nil for a no-arg method).
type EditCommitted struct {
	Document doc.ID
	Method   string
	Args     []byte
}

// EventID implements event.Event.
func (EditCommitted) EventID() event.TypeID { return tidEditCommitted }

// EmitEditCommitted publishes an EditCommitted event for a router-applied mutation,
// tagged with the active document's id (0 when none). The router calls this after a
// mutating method succeeds.
func (s *Session) EmitEditCommitted(method string, args []byte) {
	var id doc.ID
	if d := s.ActiveDocument(); d != nil {
		id = d.ID()
	}
	event.Emit(s.bus, event.After, EditCommitted{Document: id, Method: method, Args: args})
}
