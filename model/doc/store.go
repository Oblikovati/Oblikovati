// SPDX-License-Identifier: GPL-2.0-only

package doc

import "fmt"

// Store is the persistence seam the workspace depends on for save and open. It is
// defined here, on the consumer side, so the document lifecycle (M03-F02) can be
// built and tested against a fake before the real backend exists. The portable
// zip-package implementation lands in M03-F03 (the persistence package), keyed by
// stable TypeIDs (architecture core/05).
//
// A Store deals only in [Document] identity and [Content]; how that maps to bytes
// on disk is entirely the implementation's concern.
type Store interface {
	// Save writes the document's current state, overwriting any prior version at
	// its full document name. It must be atomic from a reader's point of view.
	Save(d *Document) error
	// SaveCopy writes a copy of the document at targetFullFileName without
	// touching the document's binding or dirty state (M03-F09). The copy is a
	// NEW file: the implementation mints it a fresh identity ([CopyIdentity]).
	SaveCopy(d *Document, targetFullFileName string, meta CopyMetadata) error
	// Load reads the document at the given full document name and returns it open
	// (content paged in), building live content through factories (#1617). It
	// returns an error if nothing is stored there.
	Load(fullDocumentName string, factories ContentFactories) (*Document, error)
	// Exists reports whether a document is stored at the given full document name.
	Exists(fullDocumentName string) bool
}

// CopyMetadata customizes the file a SaveCopy mints (the typed "new file"
// descriptor of M03-F09); zero-value fields inherit from the source document.
type CopyMetadata struct {
	// DisplayName overrides the copy's display name.
	DisplayName string
	// SubType re-stamps the copy's flavored subtype id.
	SubType SubTypeID
}

// newContent builds the content object for a document kind. It prefers a real content
// factory from the injected set (assembled at the composition root — see
// [ContentFactories]) so Load reconstructs live, recipe-bearing content; absent a
// factory it returns the identity-only stub. Save/Load and the create-from-template
// path use it so a kind always pairs with matching content.
func newContent(t DocumentType, factories ContentFactories) (Content, error) {
	if factory, ok := factories[t]; ok {
		return factory(), nil
	}
	switch t {
	case Part:
		return &PartComponentDefinition{}, nil
	case Assembly:
		return &AssemblyComponentDefinition{}, nil
	case Drawing:
		return &DrawingContent{}, nil
	case Presentation:
		return &PresentationContent{}, nil
	default:
		return nil, fmt.Errorf("doc: cannot create content for document type %v (want part|assembly|drawing|presentation)", t)
	}
}

// Restore reconstructs an open document from its persisted identity. A [Store]
// implementation calls it after reading a package manifest, forwarding the content
// factories its Load was handed (#1617); the session ID is freshly minted because
// ids are not persisted (core/02). The kind must be a real document type — Unknown
// and out-of-range values are rejected.
func Restore(t DocumentType, fullDocumentName, displayName string, factories ContentFactories) (*Document, error) {
	content, err := newContent(t, factories)
	if err != nil {
		return nil, err
	}
	d := newDocument(t, fullDocumentName, content, true)
	// Only carry an explicit override when the persisted name diverges from what the
	// file path implies. A derived name must stay derived so it follows the file
	// across Save As — otherwise reopening would freeze the old base name (e.g.
	// `save-as a.obk b.obk` would reopen still calling the document "a").
	if displayName != "" && displayName != derivedDisplayName(fullDocumentName) {
		d.displayName = displayName
	}
	return d, nil
}
