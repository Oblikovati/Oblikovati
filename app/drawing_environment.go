// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strconv"

	"oblikovati.org/event"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/drawing"
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
	return c, nil
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
	}
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
