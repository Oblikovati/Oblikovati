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
