// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"path/filepath"
	"strings"
	"sync/atomic"
)

// ID is a document's session-stable handle. Like entity ids (core/02), it is a
// cheap in-memory identity for maps, selection and the UI — it is NOT persisted
// and is regenerated each time a document is loaded. Cross-session document
// identity is the full document name and the reference graph (M03-F04).
type ID uint64

// idSeq mints monotonically increasing session ids; the zero id is never handed
// out so callers can treat ID(0) as "no document".
var idSeq atomic.Uint64

func nextID() ID { return ID(idSeq.Add(1)) }

// Document is the common base of every document kind: its identity, dirty flag,
// open/initialized state, and the [Content] it holds. The concrete kinds embed
// it (see [PartDocument] et al.); the workspace handles it generically via the
// [DocumentType] discriminator (architecture core/05).
//
// A document may exist as a lightweight reference stub — known identity, content
// not yet paged in — so an assembly can record what it references without loading
// every part. See [NewReference] and [Document.IsReferenceStub].
type Document struct {
	id               ID
	docType          DocumentType
	displayName      string    // explicit override; empty ⇒ derived from the file name
	fullDocumentName string    // canonical full identity (path today; see FullFileName)
	content          Content   // nil while this is an unopened reference stub
	graph            *RefGraph // the workspace's shared reference graph; nil for a standalone document
	dirty            bool
	open             bool
	visible          bool
	compacted        bool
	subType          string         // add-in flavored document subtype id, "" for plain (M05-F15)
	referencedBy     int            // how many open documents reference this one (maintained by the graph, M03-F04)
	views            *DocumentViews // per-document view collection (cameras); lazily seeded by Views()
}

// newDocument builds a base document. open reflects whether content is paged in;
// a reference stub passes nil content and open=false. A document that is open is
// visible by default; the workspace can hide it (hidden open) via SetVisible.
func newDocument(t DocumentType, fullDocumentName string, content Content, open bool) *Document {
	return &Document{
		id:               nextID(),
		docType:          t,
		fullDocumentName: fullDocumentName,
		content:          content,
		open:             open,
		visible:          open,
	}
}

// NewReference creates an unopened reference stub: a document whose identity is
// known but whose content has not been loaded. This is how the reference graph
// records a dependency without paging in the referenced model (M03-F04).
func NewReference(t DocumentType, fullDocumentName string) *Document {
	return newDocument(t, fullDocumentName, nil, false)
}

// ID returns the session-stable handle (not persisted; see [ID]).
func (d *Document) ID() ID { return d.id }

// DocumentType returns the kind discriminator.
func (d *Document) DocumentType() DocumentType { return d.docType }

// SubType returns the add-in flavored subtype id ("" for a plain document) — the
// DocumentSubType discriminator (M05-F15); persisted in the manifest.
func (d *Document) SubType() string { return d.subType }

// SetSubType stamps the flavored subtype id.
func (d *Document) SetSubType(id string) { d.subType = id }

// Content returns the modeling payload, or nil if this is an unopened reference
// stub. Callers needing a typed view use the specialization accessors
// (e.g. [PartDocument.ComponentDefinition]).
func (d *Document) Content() Content { return d.content }

// SetContent attaches the modeling payload. It is how the real component
// definition (model/compdef, M07) replaces the placeholder stub on a document
// without doc importing compdef — the content satisfies the [Content] interface and
// callers type-assert to the concrete kind.
func (d *Document) SetContent(c Content) { d.content = c }

// FullDocumentName returns the canonical full identity of the document. This is
// the cross-session identity that the reference graph stores.
func (d *Document) FullDocumentName() string { return d.fullDocumentName }

// FullFileName returns the on-disk path of the document's package. It currently
// equals [Document.FullDocumentName]; the two diverge once virtual/library
// documents (which have an identity but no own file) arrive in M03-F04.
func (d *Document) FullFileName() string { return d.fullDocumentName }

// DisplayName returns the human-readable name: the explicit override if set,
// otherwise the file name without its directory or [PackageExtension].
func (d *Document) DisplayName() string {
	if d.displayName != "" {
		return d.displayName
	}
	return derivedDisplayName(d.fullDocumentName)
}

// derivedDisplayName is the human name implied by a full document name — its base
// without directory or extension — used as the fallback when no explicit override
// is set, and by [Restore] to tell a persisted derived name from a custom one.
func derivedDisplayName(fullDocumentName string) string {
	base := filepath.Base(fullDocumentName)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// SetDisplayName overrides the derived display name. Passing "" restores the
// name derived from the file name.
func (d *Document) SetDisplayName(name string) { d.displayName = name }

// Dirty reports whether the document has unsaved changes.
func (d *Document) Dirty() bool { return d.dirty }

// MarkDirty flags the document as having unsaved changes. Editing operations
// call this; saving clears it via [Document.ClearDirty].
func (d *Document) MarkDirty() { d.dirty = true }

// ClearDirty records that the document's on-disk state matches memory, e.g.
// right after a successful save.
func (d *Document) ClearDirty() { d.dirty = false }

// Open reports whether the document's content is paged into memory. A reference
// stub is not open until it is loaded (M03-F02).
func (d *Document) Open() bool { return d.open }

// IsReferenceStub reports whether this is an unopened reference: identity known,
// content not yet loaded.
func (d *Document) IsReferenceStub() bool { return !d.open && d.content == nil }

// Visible reports whether the document is shown in the UI. A hidden-open document
// is loaded and editable but not presented (e.g. a part opened only to be placed
// into an assembly).
func (d *Document) Visible() bool { return d.visible }

// SetVisible shows or hides the document. Hiding does not unload it.
func (d *Document) SetVisible(visible bool) { d.visible = visible }

// Referenced reports whether any open document references this one. An
// unreferenced close (see Workspace.CloseAll) leaves referenced documents open.
// The count is maintained by the [RefGraph] as references are added and removed.
func (d *Document) Referenced() bool { return d.referencedBy > 0 }

// acquireRef/releaseRef adjust the referencing-document count. They are package
// internal because only the reference graph (M03-F04) should call them.
func (d *Document) acquireRef() {
	d.referencedBy++
}

func (d *Document) releaseRef() {
	if d.referencedBy > 0 {
		d.referencedBy--
	}
}

// Compacted reports whether the document's storage has been compacted. The modern
// package format saves atomically (write-temp-then-rename, core/05) and never
// leaves slack to reclaim, so this is always false; it exists for API parity with
// COM's compaction-on-save and may guide migration of legacy files.
func (d *Document) Compacted() bool { return d.compacted }
