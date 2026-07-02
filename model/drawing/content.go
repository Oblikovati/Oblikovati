// SPDX-License-Identifier: GPL-2.0-only

// (The package comment was promoted to doc.go — #1669, M40 audit D12.)

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

// BOMResolver resolves a referenced assembly document to its parts-only BOM rows for a parts list.
// Like [BodyResolver] it is the host's seam (router/app over the workspace); a nil resolver, an
// unresolved reference, or a non-assembly model means a parts list has no rows.
type BOMResolver interface {
	// BOMRows returns the named document's parts-only BOM rows, and whether it resolved.
	BOMRows(fullDocumentName string) ([]PartsListRow, bool)
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
	bom          BOMResolver     // resolves modelRef's BOM for parts lists; nil ⇒ unresolved
	lastViewBody *topo.Body      // body the views were last projected against (staleness check)
}

// NewContent creates a drawing with one default A3 landscape sheet (bordered, with the
// standard title block) and the ISO style manager — the state a freshly created drawing
// document opens in.
func NewContent() *Content {
	c := &Content{sheets: newSheets(), styles: newStylesManager()}
	c.sheets.lookup = c.resolveProperty
	c.sheets.bodyResolve = c.resolveBody
	c.sheets.bomResolve = c.resolveBOM
	c.sheets.dimPrecision = c.dimDecimals
	c.sheets.addDefault()
	return c
}

// dimDecimals returns the active drafting standard's dimension decimal places, the precision a
// sheet's dimensions format their values to. Drives drawing-dimension precision the way the part
// document's Units settings drive sketch-dimension precision.
func (c *Content) dimDecimals() int {
	return c.styles.ActiveStyle().DimensionStyle().DecimalPlaces()
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

// SetBOMResolver injects the resolver for the referenced assembly's BOM. The host calls it after
// wiring the drawing to its workspace; until then, parts lists have no rows.
func (c *Content) SetBOMResolver(bom BOMResolver) {
	c.bom = bom
	c.sheets.bomResolve = c.resolveBOM
}

// resolveBOM is the parts-list hook handed to the sheets: it resolves the referenced model's
// parts-only BOM rows through the injected resolver, or returns (nil, false) when none is wired.
func (c *Content) resolveBOM() ([]PartsListRow, bool) {
	if c.bom == nil || c.modelRef == "" {
		return nil, false
	}
	return c.bom.BOMRows(c.modelRef)
}

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
