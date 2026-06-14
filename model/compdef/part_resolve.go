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

// ResolveReferences rebinds the part's derive features (derived-assembly and derived-part)
// to their source documents after a reopen (doc.ReferenceResolver): each derive re-resolves
// its source through owner's reference graph, rebinds the live source for associative
// re-derive, and recomputes whether the source is out of date (its recipe revision now
// differs from the one captured when the derive was saved). A source that cannot be opened
// leaves the derive unbound — it contributes no geometry — and is never fatal. owner is the
// part's own document.
func (d *PartComponentDefinition) ResolveReferences(owner *doc.Document) error {
	rebound := false
	for i := 0; i < d.features.Count(); i++ {
		switch derive := d.features.Item(i).Definition().(type) {
		case *feature.DerivedAssemblyComponent:
			if child, rev, ok := openDeriveSource(owner, derive.SourceLink()); ok {
				if source, ok := child.Content().(feature.AssemblyBodySource); ok {
					derive.BindSource(source, rev)
					rebound = true
				}
			}
		case *feature.DerivedPartComponent:
			if child, rev, ok := openDeriveSource(owner, derive.SourceLink()); ok {
				if source, ok := child.Content().(feature.BodySource); ok {
					derive.BindSource(source, rev)
					rebound = true
				}
			}
		}
	}
	if rebound {
		d.Recompute()
	}
	return nil
}

// openDeriveSource opens a derive's source document through owner's reference graph,
// returning it with its current recipe revision. An empty link (a derive built without a
// document, e.g. in tests) or an unopenable source reports false.
func openDeriveSource(owner *doc.Document, link feature.DeriveSourceLink) (*doc.Document, string, bool) {
	if link.Document == "" {
		return nil, "", false
	}
	child, ok := owner.OpenReference(link.Document)
	if !ok {
		return nil, "", false
	}
	return child, child.FileIdentity().DatabaseRevisionID, true
}
