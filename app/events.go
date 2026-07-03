// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// Application UI events on the session bus. Stable TypeIDs (0x05xx = M05 UI).
const (
	tidCommandStarted         event.TypeID = 0x0501
	tidCommandEnded           event.TypeID = 0x0502
	tidSelectionChanged       event.TypeID = 0x0503
	tidTransactionChange      event.TypeID = 0x0504
	tidEditCommitted          event.TypeID = 0x0505
	tidBrowserPaneNode        event.TypeID = 0x0506 // app/browser_panes.go (M05-F03)
	tidDockableWinChanged     event.TypeID = 0x0507 // app/dockwindow_store.go (M05-F03)
	tidProgressCancelled      event.TypeID = 0x0508 // app/progress_ledger.go (M05-F09)
	tidBalloonClicked         event.TypeID = 0x0509 // app/balloon_tips.go (M05-F09)
	tidPromptAnswered         event.TypeID = 0x050a // app/prompt_center.go (M05-F09)
	tidMiniToolbarChanged     event.TypeID = 0x050b // app/minitoolbar_store.go (M05-F07)
	tidMiniToolbarCommitted   event.TypeID = 0x050c // app/minitoolbar_store.go (M05-F07)
	tidFileDialogChosen       event.TypeID = 0x050d // app/dialog_requests.go (M05-F08)
	tidWebDialogChanged       event.TypeID = 0x050e // app/dialog_requests.go (M05-F08)
	tidEnvironmentChanged     event.TypeID = 0x050f // app/ui_shell.go (M05-F12)
	tidTriadSegment           event.TypeID = 0x0510 // app/triad.go (M05-F13)
	tidTriadDrag              event.TypeID = 0x0511 // app/triad.go (M05-F13)
	tidManipulatorDrag        event.TypeID = 0x0512 // app/manipulators.go (M05-F13)
	tidPanelValueChanged      event.TypeID = 0x0513 // app/dockwindow_store.go (M05-F03 editable controls)
	tidPanelReferencesChanged event.TypeID = 0x0514 // app/dockwindow_store.go (M05-F03 referenceList)
	tidTaskPanelClosed        event.TypeID = 0x0515 // app/taskpanel_store.go (M05-F03 task panels)
	tidCameraChanged          event.TypeID = 0x1601 // app/named_views.go (M16-F03 #404)
	tidStyleChanged           event.TypeID = 0x1602 // app/style.go (M16-F02 #403/#408)
	tidBodyColorStyleChanged  event.TypeID = 0x1603 // app/style_assign.go (M16-F02 #403/#408, S5 #1640)
)

// CameraChanged fires (After) when the active view's camera moves — a named-view restore or a
// standard-orientation jump (Inventor's CameraEvents). Document is the affected document's ID;
// collaboration and overlay add-ins re-sync to the new frame.
type CameraChanged struct{ Document doc.ID }

// EventID implements event.Event.
func (CameraChanged) EventID() event.TypeID { return tidCameraChanged }

// StyleChangeKind is which style mutation a [StyleChanged] reports.
type StyleChangeKind uint8

const (
	// StyleAdded is a newly created style.
	StyleAdded StyleChangeKind = iota
	// StyleEdited is an updated style — consumers re-resolve their styling.
	StyleEdited
	// StyleDeleted is a removed style.
	StyleDeleted
)

// StyleChanged fires (After) when a color or lighting style is added, edited, or deleted —
// Inventor's StyleEvents. Name is the affected style; Kind discriminates the change.
type StyleChanged struct {
	Name string
	Kind StyleChangeKind
}

// EventID implements event.Event.
func (StyleChanged) EventID() event.TypeID { return tidStyleChanged }

// CommandStarted fires (Before) when a command begins — Inventor's
// OnExecute(Before). A handler could veto, though most observe.
type CommandStarted struct{ ID string }

// EventID implements event.Event.
func (CommandStarted) EventID() event.TypeID { return tidCommandStarted }

// PanelValueChanged fires (After) when the user edits an editable control of an add-in
// dockable window (M05-F03). It carries the owning window + control ids and the new value, so
// the add-in updates its model. It is the editable-panel counterpart of CommandStarted.
type PanelValueChanged struct {
	WindowID  string
	ControlID string
	Value     string
}

// EventID implements event.Event.
func (PanelValueChanged) EventID() event.TypeID { return tidPanelValueChanged }

// PanelReferencesChanged reports a referenceList control's row set changing (by the user's
// Add-from-selection / per-row Remove, or by an add-in's dockableWindows.setReferences).
// Refs is the full new set; Action is "add"/"remove"/"set" for diagnostics.
type PanelReferencesChanged struct {
	WindowID  string
	ControlID string
	Refs      []string
	Action    string
}

// EventID implements event.Event.
func (PanelReferencesChanged) EventID() event.TypeID { return tidPanelReferencesChanged }

// TaskPanelClosed reports the user accepting (OK) or cancelling a modal add-in task panel.
type TaskPanelClosed struct {
	ID       string
	Accepted bool
}

// EventID implements event.Event.
func (TaskPanelClosed) EventID() event.TypeID { return tidTaskPanelClosed }

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
