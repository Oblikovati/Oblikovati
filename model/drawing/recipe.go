// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
	"oblikovati.org/yamlcodec"
)

// The real drawing content reaches the document layer through the composition
// root (model/contentset.Default → doc.NewWorkspace), not init()-time
// registration (#1617).
//
// This file is the top-level drawing recipe: the drawingRecipe envelope and the
// build/marshal/apply/restore orchestration. Each recipe section — sheet frame,
// views, dimensions, annotations, sketches — is defined and (de)serialized in its
// own recipe_*.go file (M48 #2226).

// var assertion: a drawing's content is recipe-bearing (doc.RecipeContent), so the
// store persists and restores its sheets on save/open.
var _ doc.RecipeContent = (*Content)(nil)

// drawingRecipe is the YAML shape of a drawing's persisted state: the primary
// referenced model, which sheet is active, and the sheets themselves. The resolved
// title-block values are never stored — they are re-resolved from the referenced model
// on open, so the title block always reflects the current iProperties.
type drawingRecipe struct {
	ModelReference string        `yaml:"modelReference,omitempty"`
	Standard       string        `yaml:"standard,omitempty"` // active drafting standard ("" ⇒ ISO)
	ActiveSheet    int           `yaml:"activeSheet,omitempty"`
	Sheets         []sheetRecipe `yaml:"sheets,omitempty"`
}

// buildRecipe captures the drawing's full persisted state as a [drawingRecipe] value — the shared
// step behind both the YAML save ([MarshalRecipe]) and the fast undo snapshot ([Content.MarshalSnapshot],
// snapshot.go), so a snapshot reuses a faster codec without re-deriving the recipe.
func (c *Content) buildRecipe() drawingRecipe {
	r := drawingRecipe{ModelReference: c.modelRef, Standard: c.styles.active.String(), ActiveSheet: c.sheets.active}
	for _, sh := range c.sheets.items {
		r.Sheets = append(r.Sheets, sheetRecipeOf(sh))
	}
	return r
}

// MarshalRecipe renders the drawing's sheets and referenced model as YAML
// (doc.RecipeContent).
func (c *Content) MarshalRecipe() ([]byte, error) {
	return yamlcodec.Marshal(c.buildRecipe())
}

// ApplyRecipe restores the drawing from recipe YAML (doc.RecipeContent).
func (c *Content) ApplyRecipe(model []byte) error {
	var r drawingRecipe
	if err := yamlcodec.Unmarshal(model, &r); err != nil {
		return fmt.Errorf("drawing: parse recipe: %w", err)
	}
	return c.applyRecipeStruct(r)
}

// applyRecipeStruct restores the drawing from an already-decoded [drawingRecipe] — the shared tail
// of [ApplyRecipe] (YAML) and [Content.RestoreSnapshot] (the fast undo codec, snapshot.go), so both
// decode formats converge on one apply path. It replaces the content's sheets outright (so applying
// onto the factory's default sheet yields exactly the saved set), re-wiring the sheet resolution
// hooks so title-block, body-projection, BOM and dimension-precision lookups target the model again.
// The BOM hook is re-wired here too (NewContent sets it, ApplyRecipe previously did not): an undo
// restore does not re-run the host's SetBOMResolver, so without this a parts list would lose its rows
// after undo.
func (c *Content) applyRecipeStruct(r drawingRecipe) error {
	c.modelRef = r.ModelReference
	if std, ok := types.ParseDraftingStandard(r.Standard); ok {
		c.styles.SetActiveStandard(std)
	}
	c.sheets = newSheets()
	c.sheets.lookup = c.resolveProperty
	c.sheets.bodyResolve = c.resolveBody
	c.sheets.bomResolve = c.resolveBOM
	c.sheets.dimPrecision = c.dimDecimals
	for _, sr := range r.Sheets {
		if err := c.sheets.restore(sr); err != nil {
			return err
		}
	}
	c.sheets.setActiveIndex(r.ActiveSheet)
	return nil
}

// restore rebuilds one sheet from its recipe and appends it. A standard size derives
// its dimensions from size+orientation; a custom size takes the persisted laid-out
// dimensions verbatim.
func (s *Sheets) restore(rec sheetRecipe) error {
	size, ok := types.ParseSheetSize(rec.Size)
	if !ok {
		return fmt.Errorf("drawing: unknown sheet size %q", rec.Size)
	}
	orient, ok := types.ParseSheetOrientation(rec.Orientation)
	if !ok {
		orient = types.SheetPortrait
	}
	w, h, err := s.restoreDims(size, orient, rec)
	if err != nil {
		return err
	}
	sh := &Sheet{name: rec.Name, size: size, orientation: orient, width: w, height: h, views: newDrawingViews(s.bodyResolve), bomResolve: s.bomResolve, modelDims: s.modelDimsResolve, dimPrecision: s.dimPrecision, lookup: s.lookup, revision: rec.Revision}
	if rec.Border {
		sh.border = restoreBorder(rec)
	}
	if rec.TitleBlock != "" {
		sh.titleBlock = newTitleBlock(DefaultTitleBlockDefinition(), s.lookup)
		loc, _ := types.ParseTitleBlockLocation(rec.TitleBlockAt)
		sh.titleBlock.location = loc
	}
	restoreSheetContents(sh, rec)
	s.items = append(s.items, sh)
	return nil
}

// setActiveIndex clamps the restored active-sheet index into range.
func (s *Sheets) setActiveIndex(i int) {
	if i < 0 || i >= len(s.items) {
		i = 0
	}
	s.active = i
}
