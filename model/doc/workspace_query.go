// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"

	"oblikovati.org/event"
)

// Count returns the number of documents in the workspace, including unopened
// reference stubs.
func (ws *Workspace) Count() int { return len(ws.ordered) }

// LoadedCount returns how many documents have their content paged in (Open).
func (ws *Workspace) LoadedCount() int {
	n := 0
	for _, d := range ws.ordered {
		if d.open {
			n++
		}
	}
	return n
}

// Documents returns a snapshot slice of all documents in insertion order. It
// replaces COM's DocumentsEnumerator; callers range over the slice (core/02).
func (ws *Workspace) Documents() []*Document {
	out := make([]*Document, len(ws.ordered))
	copy(out, ws.ordered)
	return out
}

// VisibleDocuments returns a snapshot of the documents that are both open and
// visible, in insertion order.
func (ws *Workspace) VisibleDocuments() []*Document {
	out := make([]*Document, 0, len(ws.ordered))
	for _, d := range ws.ordered {
		if d.open && d.visible {
			out = append(out, d)
		}
	}
	return out
}

// ByID returns the document with the given session id.
func (ws *Workspace) ByID(id ID) (*Document, bool) {
	d, ok := ws.byID[id]
	return d, ok
}

// ByName returns the document with the given full document name (ItemByName).
func (ws *Workspace) ByName(fullDocumentName string) (*Document, bool) {
	d, ok := ws.byName[fullDocumentName]
	return d, ok
}

// ActiveDocument returns the document the workspace considers active, or nil if
// none are open.
func (ws *Workspace) ActiveDocument() *Document { return ws.active }

// SetActiveDocument makes an already-registered document active. It errors if the
// document is not in this workspace.
func (ws *Workspace) SetActiveDocument(d *Document) error {
	if _, ok := ws.byID[d.id]; !ok {
		return fmt.Errorf("doc: cannot activate %q: not in this workspace", d.fullDocumentName)
	}
	ws.active = d
	event.Emit(ws.bus, event.After, DocumentActivate{Document: d})
	return nil
}

// Close removes a document from the workspace. A Before DocumentClose handler may
// veto the close (e.g. to keep an unsaved document open). A dirty document is saved
// first unless skipSave is set, in which case its changes are discarded.
func (ws *Workspace) Close(d *Document, skipSave bool) error {
	if err := vetoed(ws.bus, "close", DocumentClose{Document: d}); err != nil {
		return err
	}
	if d.dirty && !skipSave {
		if err := ws.Save(d); err != nil {
			return err
		}
	}
	ws.remove(d)
	event.Emit(ws.bus, event.After, DocumentClose{Document: d})
	return nil
}

// Quit shuts down the session: a Before ApplicationQuit handler may veto it;
// otherwise every document is closed (saving dirty ones unless skipSave) and an
// After ApplicationQuit is fired.
func (ws *Workspace) Quit(skipSave bool) error {
	if err := vetoed(ws.bus, "quit", ApplicationQuit{}); err != nil {
		return err
	}
	if err := ws.CloseAll(false, skipSave); err != nil {
		return err
	}
	event.Emit(ws.bus, event.After, ApplicationQuit{})
	return nil
}

// NotifyModelChanged announces a committed batch of model changes on a document:
// a Before ModelChanged handler may veto it, after which the After event drives
// the [ChangeManager] and any other modeling subscribers. Returns a [VetoError] if
// vetoed. Edit code calls this when a transaction commits.
func (ws *Workspace) NotifyModelChanged(d *Document, changes ...ChangeDefinition) error {
	e := ModelChanged{Document: d, Changes: changes}
	if err := vetoed(ws.bus, "model change", e); err != nil {
		return err
	}
	event.Emit(ws.bus, event.After, e)
	return nil
}

// CloseAll closes every document, saving dirty ones unless skipSave is set. When
// unreferencedOnly is set, documents still referenced by another open document
// are left open (the reference graph maintains the count, M03-F04).
func (ws *Workspace) CloseAll(unreferencedOnly, skipSave bool) error {
	for _, d := range ws.Documents() { // snapshot: remove mutates ws.ordered
		if unreferencedOnly && d.Referenced() {
			continue
		}
		if err := ws.Close(d, skipSave); err != nil {
			return err
		}
	}
	return nil
}

// remove deletes a document from all indexes and reassigns the active document.
func (ws *Workspace) remove(d *Document) {
	delete(ws.byID, d.id)
	delete(ws.byName, d.fullDocumentName)
	if guid := d.identity.InternalName; guid != "" && ws.byGUID[guid] == d {
		delete(ws.byGUID, guid)
	}
	for i, other := range ws.ordered {
		if other == d {
			ws.ordered = append(ws.ordered[:i], ws.ordered[i+1:]...)
			break
		}
	}
	if ws.active == d {
		ws.active = nil
		if len(ws.ordered) > 0 {
			ws.active = ws.ordered[0]
		}
	}
}
