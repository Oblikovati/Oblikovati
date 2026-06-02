// SPDX-License-Identifier: GPL-2.0-only

package doc

// The four document specializations embed the common [Document] base and add a
// typed accessor for their content. Type-specific behavior (features, sheets,
// scenes) lands in later milestones; here each simply exposes its content object
// so the kind-specific surface exists and type discrimination is verifiable
// (architecture core/05, PBI-034).

// AsPartDocument returns a typed part view of d, or false if d is not a part
// document. The workspace stores the [Document] base; callers that retrieve a
// document and need its kind-specific surface convert through these helpers.
func AsPartDocument(d *Document) (*PartDocument, bool) {
	def, ok := d.content.(*PartComponentDefinition)
	if !ok {
		return nil, false
	}
	return &PartDocument{Document: d, def: def}, true
}

// AsAssemblyDocument returns a typed assembly view of d, or false if d is not an
// assembly document.
func AsAssemblyDocument(d *Document) (*AssemblyDocument, bool) {
	def, ok := d.content.(*AssemblyComponentDefinition)
	if !ok {
		return nil, false
	}
	return &AssemblyDocument{Document: d, def: def}, true
}

// AsDrawingDocument returns a typed drawing view of d, or false if d is not a
// drawing document.
func AsDrawingDocument(d *Document) (*DrawingDocument, bool) {
	drawing, ok := d.content.(*DrawingContent)
	if !ok {
		return nil, false
	}
	return &DrawingDocument{Document: d, drawing: drawing}, true
}

// AsPresentationDocument returns a typed presentation view of d, or false if d is
// not a presentation document.
func AsPresentationDocument(d *Document) (*PresentationDocument, bool) {
	presentation, ok := d.content.(*PresentationContent)
	if !ok {
		return nil, false
	}
	return &PresentationDocument{Document: d, presentation: presentation}, true
}

// PartDocument is a document holding a single modeled part.
type PartDocument struct {
	*Document
	def *PartComponentDefinition
}

// NewPartDocument creates an open part document with empty content.
func NewPartDocument(fullDocumentName string) *PartDocument {
	def := &PartComponentDefinition{}
	return &PartDocument{Document: newDocument(Part, fullDocumentName, def, true), def: def}
}

// ComponentDefinition returns the part's modeling content (sketches, features,
// body — populated from M07).
func (d *PartDocument) ComponentDefinition() *PartComponentDefinition { return d.def }

// AssemblyDocument is a document holding component occurrences and constraints.
type AssemblyDocument struct {
	*Document
	def *AssemblyComponentDefinition
}

// NewAssemblyDocument creates an open assembly document with empty content.
func NewAssemblyDocument(fullDocumentName string) *AssemblyDocument {
	def := &AssemblyComponentDefinition{}
	return &AssemblyDocument{Document: newDocument(Assembly, fullDocumentName, def, true), def: def}
}

// ComponentDefinition returns the assembly's modeling content (occurrences,
// constraints, joints — populated from M11).
func (d *AssemblyDocument) ComponentDefinition() *AssemblyComponentDefinition { return d.def }

// DrawingDocument is a document holding annotated sheets and views.
type DrawingDocument struct {
	*Document
	drawing *DrawingContent
}

// NewDrawingDocument creates an open drawing document with empty content.
func NewDrawingDocument(fullDocumentName string) *DrawingDocument {
	drawing := &DrawingContent{}
	return &DrawingDocument{Document: newDocument(Drawing, fullDocumentName, drawing, true), drawing: drawing}
}

// DrawingContent returns the drawing's content (sheets and views — populated from M14).
func (d *DrawingDocument) DrawingContent() *DrawingContent { return d.drawing }

// PresentationDocument is a document holding exploded/animated scenes of an assembly.
type PresentationDocument struct {
	*Document
	presentation *PresentationContent
}

// NewPresentationDocument creates an open presentation document with empty content.
func NewPresentationDocument(fullDocumentName string) *PresentationDocument {
	presentation := &PresentationContent{}
	return &PresentationDocument{Document: newDocument(Presentation, fullDocumentName, presentation, true), presentation: presentation}
}

// PresentationContent returns the presentation's content (scenes/explosions —
// populated from M16).
func (d *PresentationDocument) PresentationContent() *PresentationContent { return d.presentation }
