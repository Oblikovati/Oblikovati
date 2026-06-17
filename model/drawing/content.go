// SPDX-License-Identifier: GPL-2.0-only

// Package drawing is the modeling content of a drawing document (M14-F01, #384): an
// ordered set of sheets (sizes/orientation, borders, title blocks) plus the primary
// referenced model the drawing documents. Title-block fields resolve against that
// model's iProperties, so a title block shows live part/assembly metadata.
//
// This is the GPL implementation of the api/contract drawing surface (ADR-0018); it
// registers itself as the real content for doc.Drawing so opening a .odd reconstructs
// live sheets rather than the identity-only stub.
package drawing

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/doc"
)

// ModelProperties resolves a referenced model document's iProperty values for
// title-block field resolution. It is the seam between this package (which knows
// nothing of the workspace) and the host, which finds the referenced model and reads
// its property sets. The host injects it via [Content.SetModelProperties]; a nil
// resolver (an unwired drawing, or no reference set) leaves property fields blank.
type ModelProperties interface {
	// Property returns the value of the named property in the named iProperty set
	// (e.g. "Design Tracking Properties", "Part Number"), and whether it exists.
	Property(set, name string) (value string, ok bool)
}

// BodyResolver resolves a referenced model document to its B-rep body for drawing-view
// projection. Like [ModelProperties] it is the host's seam (router/app over the workspace);
// a nil resolver or an unresolved reference means views cannot be projected yet.
type BodyResolver interface {
	// Body returns the body of the named model document, and whether it resolved.
	Body(fullDocumentName string) (*topo.Body, bool)
}

// Content is a drawing document's modeling content. It implements doc.Content (and
// doc.RecipeContent, in recipe.go) so the document/persistence layers treat it like
// any other content kind.
type Content struct {
	sheets       *Sheets
	styles       *StylesManager
	modelRef     string          // full document name of the primary referenced model
	props        ModelProperties // resolves modelRef's iProperties; nil ⇒ unresolved
	bodies       BodyResolver    // resolves modelRef's body for view projection; nil ⇒ unresolved
	lastViewBody *topo.Body      // body the views were last projected against (staleness check)
}

// NewContent creates a drawing with one default A3 landscape sheet (bordered, with the
// standard title block) and the ISO style manager — the state a freshly created drawing
// document opens in.
func NewContent() *Content {
	c := &Content{sheets: newSheets(), styles: newStylesManager()}
	c.sheets.lookup = c.resolveProperty
	c.sheets.bodyResolve = c.resolveBody
	c.sheets.addDefault()
	return c
}

// DocumentType identifies this as drawing content.
func (c *Content) DocumentType() doc.DocumentType { return doc.Drawing }

// Sheets returns the drawing's sheets.
func (c *Content) Sheets() *Sheets { return c.sheets }

// Styles returns the drawing's style system (active drafting standard + presets).
func (c *Content) Styles() *StylesManager { return c.styles }

// ModelReference returns the full document name of the primary referenced model, or ""
// if none is set.
func (c *Content) ModelReference() string { return c.modelRef }

// SetModelReference points the drawing at the model it documents; its title-block
// fields resolve against that model's iProperties. An empty name clears the reference.
func (c *Content) SetModelReference(fullDocumentName string) { c.modelRef = fullDocumentName }

// SetModelProperties injects the resolver for the referenced model's iProperties. The
// host (router/app) calls it after wiring the drawing to its workspace; until then,
// property-bound title-block fields resolve to "".
func (c *Content) SetModelProperties(props ModelProperties) { c.props = props }

// SetBodyResolver injects the resolver for the referenced model's body. The host calls it
// after wiring the drawing to its workspace; until then, views cannot be projected.
func (c *Content) SetBodyResolver(bodies BodyResolver) { c.bodies = bodies }

// resolveBody is the view-projection hook handed to the sheets: it resolves the referenced
// model's body through the injected resolver, or returns (nil, false) when none is wired.
func (c *Content) resolveBody() (*topo.Body, bool) {
	if c.bodies == nil || c.modelRef == "" {
		return nil, false
	}
	return c.bodies.Body(c.modelRef)
}

// RecomputeViews re-projects every sheet's views against the current referenced model — the
// associativity path the host runs after the model changes.
func (c *Content) RecomputeViews() {
	for i := 0; i < c.sheets.Count(); i++ {
		sh := c.sheets.Item(i)
		sh.views.Recompute()
		if sh.annotations != nil {
			sh.annotations.Recompute() // CoG markers track the model centroid
		}
		if sh.dimensions != nil {
			sh.dimensions.Recompute() // dimensions re-measure against the recomputed model
		}
	}
}

// SyncViews re-projects the views only if the referenced model's body changed since the last
// projection (a part recompute rebuilds the body, so its pointer changes). The head calls it
// every frame an open drawing renders, so the views track edits made to the part in another
// tab or window — cheaply, since an unchanged model skips the re-projection.
func (c *Content) SyncViews() {
	body, ok := c.resolveBody()
	if !ok {
		return
	}
	if body == c.lastViewBody {
		return
	}
	c.lastViewBody = body
	c.RecomputeViews()
}

// resolveProperty is the title-block resolution hook handed to every sheet: it reads
// the referenced model's iProperties through the injected resolver, or returns
// ("", false) when no model is referenced or no resolver is wired.
func (c *Content) resolveProperty(set, name string) (string, bool) {
	if c.props == nil || c.modelRef == "" {
		return "", false
	}
	return c.props.Property(set, name)
}
