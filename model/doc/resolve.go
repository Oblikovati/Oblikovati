// SPDX-License-Identifier: GPL-2.0-only

package doc

// ReferenceResolver is the optional interface content implements when reopening it must
// resolve references to other documents — an assembly binding its occurrences to the
// component documents they instance (#715). The workspace calls ResolveReferences once,
// right after the owning document is registered, so the resolver can reach its sibling
// documents through owner.OpenReference (which hidden-opens them through the graph). A
// content type that does not implement this interface round-trips its own recipe alone.
//
// The owner is passed in rather than held as a back-reference because content has no
// link to its document otherwise, and the resolver needs the graph-connected document
// (which exists only after registration) to open and record references.
type ReferenceResolver interface {
	Content
	// ResolveReferences binds owner's restored references to live documents. A reference
	// that cannot be resolved must be surfaced (e.g. a broken descriptor), never fatal.
	ResolveReferences(owner *Document) error
}

// OpenReference resolves the document named fullDocumentName as a reference of d: an
// already-open document wins, otherwise it is hidden-opened through the workspace store.
// It reuses the reference graph's resolution (already-open short-circuit, lazy load,
// broken-flagging) and records the d→child edge so the dependency is tracked and the
// child stays open while d references it. Returns false when the target cannot be found.
func (d *Document) OpenReference(fullDocumentName string) (*Document, bool) {
	if d.graph == nil {
		return nil, false
	}
	desc := d.graph.addReferenceUnique(d, fullDocumentName)
	return d.graph.resolve(desc)
}
