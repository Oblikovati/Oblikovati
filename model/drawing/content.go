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

import "oblikovati.org/model/doc"

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

// Content is a drawing document's modeling content. It implements doc.Content (and
// doc.RecipeContent, in recipe.go) so the document/persistence layers treat it like
// any other content kind.
type Content struct {
	sheets   *Sheets
	styles   *StylesManager
	modelRef string          // full document name of the primary referenced model
	props    ModelProperties // resolves modelRef's iProperties; nil ⇒ unresolved
}

// NewContent creates a drawing with one default A3 landscape sheet (bordered, with the
// standard title block) and the ISO style manager — the state a freshly created drawing
// document opens in.
func NewContent() *Content {
	c := &Content{sheets: newSheets(), styles: newStylesManager()}
	c.sheets.lookup = c.resolveProperty
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

// resolveProperty is the title-block resolution hook handed to every sheet: it reads
// the referenced model's iProperties through the injected resolver, or returns
// ("", false) when no model is referenced or no resolver is wired.
func (c *Content) resolveProperty(set, name string) (string, bool) {
	if c.props == nil || c.modelRef == "" {
		return "", false
	}
	return c.props.Property(set, name)
}
