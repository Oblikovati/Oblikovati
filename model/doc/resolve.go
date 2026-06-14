// SPDX-License-Identifier: GPL-2.0-only

package doc

import "path/filepath"

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
	if target, ok := d.graph.resolve(desc); ok {
		return target, true
	}
	// The stored (absolute) name no longer resolves — the project tree may have moved.
	// Re-anchor this reference's owner-relative spelling to the owner's CURRENT directory
	// and retry, repointing the edge so a re-save records the new location (#750).
	relocated, ok := d.relocatedReferenceTarget(fullDocumentName)
	if !ok || relocated == fullDocumentName {
		return nil, false
	}
	d.graph.repointReference(d, fullDocumentName, relocated)
	return d.graph.resolve(desc)
}

// relocatedReferenceTarget re-anchors a moved reference: when the owner holds a persisted
// file-reference record for name carrying an owner-directory-relative spelling, it re-joins
// that spelling to the owner's CURRENT directory, so a project tree moved as a whole still
// resolves on reopen (#750). It reports false when no relative record covers name (e.g. a
// library or absolute reference), leaving absolute resolution as the last resort.
func (d *Document) relocatedReferenceTarget(name string) (string, bool) {
	for _, r := range d.fileReferences {
		if r.fullFileName != name || r.relativeFileName == "" {
			continue
		}
		return filepath.Join(filepath.Dir(d.fullDocumentName), filepath.FromSlash(r.relativeFileName)), true
	}
	return "", false
}
