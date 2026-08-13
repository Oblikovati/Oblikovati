// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strconv"

	"oblikovati.org/event"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/drawing"
	"oblikovati.org/model/sketch"
)

// The drawing environment (M14-F01, #384): creating a drawing document and reaching the
// active drawing's content, with its title-block fields wired to resolve against the
// referenced model's iProperties. New Drawing is the launch action; ActiveDrawing backs
// the Drawing-tab tools and the head's sheet canvas.

// NewDrawing creates a drawing document with a unique "DrawingN" name and makes it
// active (the workspace activates a newly added document), so the ribbon switches to the
// drawing environment. It mirrors [Session.NewPart] / [Session.NewAssembly] and backs the
// New Drawing command on the ZeroDoc ribbon.
//
//	d, err := session.NewDrawing()
func (s *Session) NewDrawing() (*doc.Document, error) {
	ev := FileNew{DocumentType: doc.Drawing}
	if out := event.Emit(s.bus, event.Before, ev); out.Vetoed() {
		s.notice = out.Reason
		return nil, &doc.VetoError{Operation: "new drawing", Reason: out.Reason}
	}
	d, err := s.workspace.Add(doc.Drawing, s.uniqueDocumentName("Drawing"), true)
	if err != nil {
		return nil, err
	}
	wireDrawingResolver(s, d)
	s.documentHistory(d) // open the event stream now so the first sheet edit is undoable
	event.Emit(s.bus, event.After, ev)
	return d, nil
}

// ActiveDrawing returns the active document's drawing content with its title-block
// resolver wired to the workspace, or an error if the active document is not a drawing.
// Both the Drawing-tab tools and the router's drawing.* handlers go through it.
func ActiveDrawing(s *Session) (*drawing.Content, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, fmt.Errorf("drawing: no active document")
	}
	c, ok := d.Content().(*drawing.Content)
	if !ok {
		return nil, fmt.Errorf("drawing: active document %q is not a drawing", d.FullDocumentName())
	}
	c.SetModelProperties(referencedModelProperties{ws: s.workspace, ref: c.ModelReference})
	c.SetBodyResolver(referencedModelBodies{ws: s.workspace})
	c.SetBOMResolver(referencedModelBOM{ws: s.workspace})
	c.SetModelDimensionResolver(referencedModelDimensions{ws: s.workspace})
	return c, nil
}

// referencedModelBOM resolves a drawing's referenced assembly document to its parts-only BOM rows
// for a parts list, looking it up by name in the workspace on each call (so the list tracks the
// assembly). A non-assembly model (or one with no occurrences) yields no rows.
type referencedModelBOM struct {
	ws *doc.Workspace
}

func (r referencedModelBOM) BOMRows(fullDocumentName string) ([]drawing.PartsListRow, bool) {
	d, ok := r.ws.ByName(fullDocumentName)
	if !ok || d.Content() == nil {
		return nil, false
	}
	asm, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return nil, false
	}
	view := bom.New(asm.Occurrences()).PartsOnly()
	if len(view.Rows) == 0 {
		return nil, false
	}
	rows := make([]drawing.PartsListRow, len(view.Rows))
	for i, br := range view.Rows {
		rows[i] = drawing.PartsListRow{Item: br.ItemNumber, PartNumber: br.PartNumber, Description: br.Description, Quantity: br.Quantity}
	}
	return rows, true
}

// hasActiveDrawing reports whether the active document is a drawing — the enable
// predicate for the Drawing ribbon tab (mirrors [hasActivePart]).
func hasActiveDrawing(s *Session) bool {
	c, _ := ActiveDrawing(s)
	return c != nil
}

// wireDrawingResolver injects the workspace-backed iProperty resolver into a drawing's
// content so its title block resolves against the referenced model — for in-proc readers
// (the head sheet canvas) that do not go through ActiveDrawing.
func wireDrawingResolver(s *Session, d *doc.Document) {
	if c, ok := d.Content().(*drawing.Content); ok {
		c.SetModelProperties(referencedModelProperties{ws: s.workspace, ref: c.ModelReference})
		c.SetBodyResolver(referencedModelBodies{ws: s.workspace})
		c.SetBOMResolver(referencedModelBOM{ws: s.workspace})
		c.SetModelDimensionResolver(referencedModelDimensions{ws: s.workspace})
	}
}

// referencedModelDimensions resolves a drawing's referenced part document to its retrievable
// parametric dimensions (#1991): each of the part's sketch distance-dimensions, resolved to its
// parameter name, measured value and the two world endpoints it spans (via the sketch plane). Other
// dimension kinds (radius/diameter/angle) are not retrieved this increment.
type referencedModelDimensions struct {
	ws *doc.Workspace
}

func (r referencedModelDimensions) ModelDimensions(fullDocumentName string) ([]drawing.ModelDimension, bool) {
	d, ok := r.ws.ByName(fullDocumentName)
	if !ok || d.Content() == nil {
		return nil, false
	}
	src, ok := d.Content().(interface{ Sketches() *sketch.Sketches })
	if !ok {
		return nil, false
	}
	var out []drawing.ModelDimension
	sketches := src.Sketches()
	for i := 0; i < sketches.Count(); i++ {
		out = append(out, sketchDistanceDimensions(sketches.Item(i))...)
	}
	return out, true
}

// sketchDistanceDimensions maps a sketch's distance dimensions to model dimensions, projecting each
// point through the sketch plane to world space (#1991).
func sketchDistanceDimensions(sk *sketch.Sketch) []drawing.ModelDimension {
	plane := sk.Plane()
	var out []drawing.ModelDimension
	for _, dc := range sk.DimensionConstraints().All() {
		if dc.Kind() != sketch.DistanceDim {
			continue
		}
		refs := dc.Refs()
		if len(refs) != 2 {
			continue
		}
		a, aok := refs[0].(*sketch.Point)
		b, bok := refs[1].(*sketch.Point)
		if !aok || !bok {
			continue
		}
		out = append(out, drawing.ModelDimension{
			Name: dc.Parameter().Name(), Value: dc.Measured(),
			A: plane.ToModel(a.Position()), B: plane.ToModel(b.Position()),
		})
	}
	return out
}

// referencedModelBodies resolves a drawing's referenced model document to its body for view
// projection, looking it up by name in the workspace on each call (so it tracks the model
// being opened or edited). It returns the model's first surface body; multi-body parts are a
// follow-up.
type referencedModelBodies struct {
	ws *doc.Workspace
}

func (r referencedModelBodies) Body(fullDocumentName string) (*topo.Body, bool) {
	d, ok := r.ws.ByName(fullDocumentName)
	if !ok || d.Content() == nil {
		return nil, false
	}
	src, ok := d.Content().(interface{ SurfaceBodies() *topo.SurfaceBodies })
	if !ok {
		return nil, false
	}
	bodies := src.SurfaceBodies().All()
	if len(bodies) == 0 {
		return nil, false
	}
	return bodies[0], true
}

// referencedModelProperties resolves a drawing's referenced model iProperties by looking
// the document up by name in the workspace on each read (so it tracks the model being
// opened or edited after the drawing references it). ref re-reads the drawing's current
// reference so a later setModelReference is honoured.
type referencedModelProperties struct {
	ws  *doc.Workspace
	ref func() string
}

func (p referencedModelProperties) Property(set, name string) (string, bool) {
	d, ok := p.ws.ByName(p.ref())
	if !ok || d.Content() == nil {
		return "", false
	}
	props, ok := d.Content().(interface{ Properties() *attr.PropertySets })
	if !ok {
		return "", false
	}
	ps, ok := props.Properties().Lookup(set)
	if !ok {
		return "", false
	}
	prop, ok := ps.Property(name)
	if !ok {
		return "", false
	}
	return propertyText(prop.Value()), true
}

// propertyText renders an iProperty value as the plain text a title block shows.
func propertyText(v attr.Value) string {
	if s, ok := v.Str(); ok {
		return s
	}
	if i, ok := v.Int(); ok {
		return strconv.FormatInt(i, 10)
	}
	if f, ok := v.Float(); ok {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	if b, ok := v.Bool(); ok {
		return strconv.FormatBool(b)
	}
	return ""
}
