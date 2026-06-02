// SPDX-License-Identifier: GPL-2.0-only

package doc

// Content is the modeling payload a document holds — the data side of the
// document/content split (parametric-cad §1b). The document owns file identity
// and lifecycle; Content owns the model that recompute rebuilds.
//
// The concrete content types are stubs in M03: the real component definitions,
// sheets and scenes are built in later milestones (Part→M07, Assembly→M11,
// Drawing→M14, Presentation→M16). Defining the interface now lets the document
// and workspace layers be built and tested first (architecture core/05). The
// later concrete types implement this same interface; nothing here is thrown away.
type Content interface {
	// DocumentType reports the kind of document this content belongs to, so a
	// document and its content can never be mismatched.
	DocumentType() DocumentType
}

// RecipeContent is the optional interface real content implements to persist and
// restore its modeling recipe (parameters, sketches, features) as opaque YAML bytes.
// The store serializes the recipe for content that implements it and restores it on
// open; a bare stub (which implements only [Content]) round-trips identity alone.
// The recipe schema is owned by the content's own package (e.g. model/compdef), not
// by doc or persistence — keeping the format layer content-agnostic (ADR-0020).
type RecipeContent interface {
	Content
	// MarshalRecipe renders the content's recipe as YAML bytes.
	MarshalRecipe() ([]byte, error)
	// ApplyRecipe restores the content from recipe YAML bytes (and recomputes).
	ApplyRecipe(model []byte) error
}

// contentFactories holds the constructor for each document kind's REAL content,
// registered by the owning package's init (e.g. model/compdef). It is the seam that
// lets [Restore] rebuild live content — with its recipe machinery — without doc
// importing the heavy model packages (which would cycle). Writes happen at init
// (single-threaded); reads happen later.
var contentFactories = map[DocumentType]func() Content{}

// RegisterContentFactory installs the constructor for a kind's real content. Call it
// from an init() in the package that owns the content type, so any binary that imports
// that package gets live content on open. Without a registered factory, [newContent]
// falls back to the identity-only stub.
func RegisterContentFactory(t DocumentType, factory func() Content) {
	contentFactories[t] = factory
}

// PartComponentDefinition is the stub for a part's modeling content: sketches,
// features and the resulting B-rep body (M07). Modernizes COM
// PartComponentDefinition.
type PartComponentDefinition struct{}

// DocumentType identifies this as part content.
func (*PartComponentDefinition) DocumentType() DocumentType { return Part }

// AssemblyComponentDefinition is the stub for an assembly's modeling content:
// component occurrences, constraints and joints (M11). Modernizes COM
// AssemblyComponentDefinition.
type AssemblyComponentDefinition struct{}

// DocumentType identifies this as assembly content.
func (*AssemblyComponentDefinition) DocumentType() DocumentType { return Assembly }

// DrawingContent is the stub for a drawing's content: the sheets and the views
// they annotate (M14).
type DrawingContent struct{}

// DocumentType identifies this as drawing content.
func (*DrawingContent) DocumentType() DocumentType { return Drawing }

// PresentationContent is the stub for a presentation's content: exploded and
// animated scenes of an assembly (M16).
type PresentationContent struct{}

// DocumentType identifies this as presentation content.
func (*PresentationContent) DocumentType() DocumentType { return Presentation }
