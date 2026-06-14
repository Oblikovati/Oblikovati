// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// A part resolves the source documents of its derived-assembly features on reopen (#715):
// the derive recipe carries the source's identity, but the live source assembly — and
// whether it has since changed — is only knowable once the workspace can open it.
var _ doc.ReferenceResolver = (*PartComponentDefinition)(nil)

// ResolveReferences rebinds the part's derived-assembly features to their source documents
// after a reopen (doc.ReferenceResolver): each derive re-resolves its source through
// owner's reference graph, rebinds the live source for associative re-derive, and
// recomputes whether the source is out of date (its recipe revision now differs from the
// one captured when the derive was saved). A source that cannot be opened leaves the
// derive unbound — it contributes no geometry — and is never fatal. owner is the part's
// own document.
func (d *PartComponentDefinition) ResolveReferences(owner *doc.Document) error {
	rebound := false
	for i := 0; i < d.features.Count(); i++ {
		derive, ok := d.features.Item(i).Definition().(*feature.DerivedAssemblyComponent)
		if !ok {
			continue
		}
		if d.bindDeriveSource(owner, derive) {
			rebound = true
		}
	}
	if rebound {
		d.Recompute()
	}
	return nil
}

// bindDeriveSource opens a derive's source document through owner's reference graph and
// rebinds it, reporting whether a live source was bound. An empty link (a derive built
// without a document, e.g. in tests) or an unopenable/ non-assembly source is skipped.
func (d *PartComponentDefinition) bindDeriveSource(owner *doc.Document, derive *feature.DerivedAssemblyComponent) bool {
	link := derive.SourceLink()
	if link.Document == "" {
		return false
	}
	child, ok := owner.OpenReference(link.Document)
	if !ok {
		return false
	}
	source, ok := child.Content().(feature.AssemblyBodySource)
	if !ok {
		return false
	}
	derive.BindSource(source, child.FileIdentity().DatabaseRevisionID)
	return true
}
