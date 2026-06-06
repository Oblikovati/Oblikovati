// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"

	"oblikovati/event"
)

// Workspace is the in-memory collection of open documents and the active one — it
// replaces COM's Documents collection plus the Application's document ownership
// (architecture core/02, core/05). It owns the create-from-template, open, save
// and close lifecycle. Document content is persisted through an injected [Store]
// so the lifecycle stays independent of the on-disk format (M03-F03).
//
// Construction goes through the workspace, not the document constructors, because
// this is the seam that enforces identity and (later) the reference graph and
// transactions (parametric-cad §13).
type Workspace struct {
	store   Store
	ordered []*Document          // insertion order, for stable enumeration
	byID    map[ID]*Document     // session-id lookup
	byName  map[string]*Document // full-document-name lookup (ItemByName)
	graph   *RefGraph            // shared document reference graph (M03-F04)
	bus     *event.Bus           // application/document/modeling events (M04-F04)
	active  *Document
}

// NewWorkspace creates an empty workspace backed by store. A nil store is allowed
// for purely in-memory sessions; Save/Open then return an error.
func NewWorkspace(store Store) *Workspace {
	ws := &Workspace{
		store:  store,
		byID:   map[ID]*Document{},
		byName: map[string]*Document{},
		bus:    event.NewBus(),
	}
	ws.graph = newRefGraph(ws)
	return ws
}

// References returns the workspace's document reference graph.
func (ws *Workspace) References() *RefGraph { return ws.graph }

// Events returns the workspace event bus. Subscribe to application/document/
// modeling events through it (e.g. event.Subscribe(ws.Events(), event.Before, …)).
func (ws *Workspace) Events() *event.Bus { return ws.bus }

// OpenOptions is the typed options bag for OpenWithOptions, replacing COM's
// stringly-typed NameValueMap (core/05: typed option structs in-proc).
type OpenOptions struct {
	// Visible controls whether the opened document is shown (hidden-open when false).
	Visible bool
	// DeferContent opens the document as a reference stub — identity registered,
	// content not paged in — for callers that only need to record a dependency.
	DeferContent bool
}

// Add creates a new document of kind t from the built-in template and registers
// it as the active document. The full template-file resolution via the project
// search path arrives in M03-F04; here the template is the empty default content.
// A freshly created document is dirty until first saved.
func (ws *Workspace) Add(t DocumentType, fullDocumentName string, visible bool) (*Document, error) {
	if _, taken := ws.byName[fullDocumentName]; taken {
		return nil, fmt.Errorf("doc: a document named %q is already open", fullDocumentName)
	}
	content, err := newContent(t)
	if err != nil {
		return nil, err
	}
	d := newDocument(t, fullDocumentName, content, true)
	d.visible = visible
	d.dirty = true
	ws.register(d)
	event.Emit(ws.bus, event.After, DocumentCreated{Document: d})
	return d, nil
}

// Open loads the document at fullDocumentName (or returns the already-open one),
// makes it active, and returns it. It is OpenWithOptions with default options.
func (ws *Workspace) Open(fullDocumentName string, visible bool) (*Document, error) {
	return ws.OpenWithOptions(fullDocumentName, OpenOptions{Visible: visible})
}

// OpenWithOptions loads a document honoring opts. An already-open document is
// returned as-is (its visibility updated). DeferContent yields a reference stub
// without touching the store.
func (ws *Workspace) OpenWithOptions(fullDocumentName string, opts OpenOptions) (*Document, error) {
	if existing, ok := ws.byName[fullDocumentName]; ok {
		existing.SetVisible(opts.Visible)
		ws.active = existing
		return existing, nil
	}
	if opts.DeferContent {
		return ws.openStub(fullDocumentName)
	}
	if ws.store == nil {
		return nil, fmt.Errorf("doc: cannot open %q: no store configured", fullDocumentName)
	}
	if err := vetoed(ws.bus, "open", DocumentOpened{FullDocumentName: fullDocumentName}); err != nil {
		return nil, err
	}
	d, err := ws.store.Load(fullDocumentName)
	if err != nil {
		return nil, fmt.Errorf("doc: open %q: %w", fullDocumentName, err)
	}
	d.visible = opts.Visible
	ws.register(d)
	event.Emit(ws.bus, event.After, DocumentOpened{FullDocumentName: fullDocumentName})
	return d, nil
}

// openStub registers an unopened reference stub. The kind is unknown until the
// package manifest is read (M03-F03/F04), so it carries DocumentType Unknown.
func (ws *Workspace) openStub(fullDocumentName string) (*Document, error) {
	d := NewReference(Unknown, fullDocumentName)
	ws.register(d)
	return d, nil
}

// Save writes the document through the store and clears its dirty flag.
func (ws *Workspace) Save(d *Document) error {
	if ws.store == nil {
		return fmt.Errorf("doc: cannot save %q: no store configured", d.fullDocumentName)
	}
	if err := vetoed(ws.bus, "save", DocumentSave{Document: d}); err != nil {
		return err
	}
	if err := ws.store.Save(d); err != nil {
		return fmt.Errorf("doc: save %q: %w", d.fullDocumentName, err)
	}
	d.ClearDirty()
	event.Emit(ws.bus, event.After, DocumentSave{Document: d})
	return nil
}

// SaveAs writes the document under a new full document name, which becomes its
// identity. The new name must not collide with another open document.
func (ws *Workspace) SaveAs(d *Document, newFullDocumentName string) error {
	if other, ok := ws.byName[newFullDocumentName]; ok && other != d {
		return fmt.Errorf("doc: cannot save as %q: name already open", newFullDocumentName)
	}
	delete(ws.byName, d.fullDocumentName)
	d.fullDocumentName = newFullDocumentName
	ws.byName[newFullDocumentName] = d
	return ws.Save(d)
}

// register inserts d into all indexes, links it to the reference graph, and makes
// it the active document.
func (ws *Workspace) register(d *Document) {
	d.graph = ws.graph
	ws.ordered = append(ws.ordered, d)
	ws.byID[d.id] = d
	ws.byName[d.fullDocumentName] = d
	ws.active = d
}
