// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"

	"oblikovati.org/event"
)

// The core event types fired by the workspace. They are grouped into the COM
// "event sets" by purpose — application lifecycle, per-document lifecycle, and
// modeling changes — but on the typed bus a "set" is just a family of event
// structs, each carrying its own typed payload (architecture core/06). Each is
// emitted in the Before phase (vetoable) and then, if not vetoed, the After phase.
//
// TypeIDs are stable across versions (they cross the gRPC seam); never renumber.
const (
	tidDocumentCreated            event.TypeID = 0x0401
	tidDocumentOpened             event.TypeID = 0x0402
	tidDocumentSave               event.TypeID = 0x0403
	tidDocumentClose              event.TypeID = 0x0404
	tidDocumentActivate           event.TypeID = 0x0405
	tidDocumentIdentityReassigned event.TypeID = 0x0406
	tidApplicationQuit            event.TypeID = 0x0410
	tidModelChanged               event.TypeID = 0x0420
)

// DocumentCreated is fired (After only) when a new document is created from a
// template. ApplicationEvents.OnNewDocument.
type DocumentCreated struct{ Document *Document }

// EventID implements event.Event.
func (DocumentCreated) EventID() event.TypeID { return tidDocumentCreated }

// DocumentOpened is fired around opening a document from storage.
// ApplicationEvents.OnOpenDocument; a Before handler may veto the open.
type DocumentOpened struct{ FullDocumentName string }

// EventID implements event.Event.
func (DocumentOpened) EventID() event.TypeID { return tidDocumentOpened }

// DocumentSave is fired around saving. ApplicationEvents/DocumentEvents.OnSave.
type DocumentSave struct{ Document *Document }

// EventID implements event.Event.
func (DocumentSave) EventID() event.TypeID { return tidDocumentSave }

// DocumentClose is fired around closing a document; a Before handler may veto it
// (e.g. to prompt to save). ApplicationEvents/DocumentEvents.OnClose.
type DocumentClose struct{ Document *Document }

// EventID implements event.Event.
func (DocumentClose) EventID() event.TypeID { return tidDocumentClose }

// DocumentActivate is fired (After) when a document becomes active.
type DocumentActivate struct{ Document *Document }

// EventID implements event.Event.
func (DocumentActivate) EventID() event.TypeID { return tidDocumentActivate }

// DocumentIdentityReassigned is fired (After) when a document is opened whose persisted
// identity GUID collides with an already-open document, so the workspace minted it a fresh
// identity to keep identities unique within the session. The new GUID is in memory only until
// the document's next save persists it. The host surfaces this to the user as a notice.
type DocumentIdentityReassigned struct {
	Document             *Document
	PreviousInternalName string
	NewInternalName      string
}

// EventID implements event.Event.
func (DocumentIdentityReassigned) EventID() event.TypeID { return tidDocumentIdentityReassigned }

// ApplicationQuit is fired around shutting down the session; a Before handler may
// veto it. ApplicationEvents.OnQuit.
type ApplicationQuit struct{}

// EventID implements event.Event.
func (ApplicationQuit) EventID() event.TypeID { return tidApplicationQuit }

// VetoError reports that a Before-event handler cancelled an operation. Callers
// distinguish it from real failures via errors.As.
type VetoError struct {
	Operation string
	Reason    string
}

// Error implements error.
func (e *VetoError) Error() string {
	return fmt.Sprintf("doc: %s vetoed: %s", e.Operation, e.Reason)
}

// vetoed emits a Before event and returns a VetoError if any handler vetoed it.
func vetoed[E event.Event](bus *event.Bus, operation string, e E) error {
	if out := event.Emit(bus, event.Before, e); out.Vetoed() {
		return &VetoError{Operation: operation, Reason: out.Reason}
	}
	return nil
}
