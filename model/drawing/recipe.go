// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence/yamlcodec"
)

// init registers the real drawing content with the document layer so opening a .odd
// reconstructs live sheets (with the recipe machinery), not the identity-only stub
// (see doc.RegisterContentFactory).
func init() {
	doc.RegisterContentFactory(doc.Drawing, func() doc.Content { return NewContent() })
}

// var assertion: a drawing's content is recipe-bearing (doc.RecipeContent), so the
// store persists and restores its sheets on save/open.
var _ doc.RecipeContent = (*Content)(nil)

// drawingRecipe is the YAML shape of a drawing's persisted state: the primary
// referenced model, which sheet is active, and the sheets themselves. The resolved
// title-block values are never stored — they are re-resolved from the referenced model
// on open, so the title block always reflects the current iProperties.
type drawingRecipe struct {
	ModelReference string        `yaml:"modelReference,omitempty"`
	ActiveSheet    int           `yaml:"activeSheet,omitempty"`
	Sheets         []sheetRecipe `yaml:"sheets,omitempty"`
}

// sheetRecipe is the YAML shape of one sheet. WidthMM/HeightMM are written only for a
// custom size (a standard size's dimensions come from the size+orientation on open).
type sheetRecipe struct {
	Name        string  `yaml:"name"`
	Size        string  `yaml:"size"`
	Orientation string  `yaml:"orientation,omitempty"`
	WidthMM     float64 `yaml:"widthMm,omitempty"`
	HeightMM    float64 `yaml:"heightMm,omitempty"`
	Border      bool    `yaml:"border"`
	TitleBlock  string  `yaml:"titleBlock,omitempty"` // definition name; "" ⇒ no title block
}

// MarshalRecipe renders the drawing's sheets and referenced model as YAML
// (doc.RecipeContent).
func (c *Content) MarshalRecipe() ([]byte, error) {
	r := drawingRecipe{ModelReference: c.modelRef, ActiveSheet: c.sheets.active}
	for _, sh := range c.sheets.items {
		r.Sheets = append(r.Sheets, sheetRecipeOf(sh))
	}
	return yamlcodec.Marshal(r)
}

// ApplyRecipe restores the drawing from recipe YAML. It replaces the content's sheets
// outright (so applying onto the factory's default sheet yields exactly the saved set),
// re-wiring the title-block resolution hook so fields resolve against the model again.
func (c *Content) ApplyRecipe(model []byte) error {
	var r drawingRecipe
	if err := yamlcodec.Unmarshal(model, &r); err != nil {
		return fmt.Errorf("drawing: parse recipe: %w", err)
	}
	c.modelRef = r.ModelReference
	c.sheets = newSheets()
	c.sheets.lookup = c.resolveProperty
	for _, sr := range r.Sheets {
		if err := c.sheets.restore(sr); err != nil {
			return err
		}
	}
	c.sheets.setActiveIndex(r.ActiveSheet)
	return nil
}

// sheetRecipeOf snapshots one sheet. A custom size carries its laid-out dimensions
// (already orientation-applied) so restore reproduces them without re-swapping.
func sheetRecipeOf(sh *Sheet) sheetRecipe {
	rec := sheetRecipe{
		Name:        sh.name,
		Size:        sh.size.String(),
		Orientation: sh.orientation.String(),
		Border:      sh.border != nil,
	}
	if sh.size == types.SheetSizeCustom {
		rec.WidthMM, rec.HeightMM = sh.width, sh.height
	}
	if sh.titleBlock != nil {
		rec.TitleBlock = sh.titleBlock.def.name
	}
	return rec
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
	sh := &Sheet{name: rec.Name, size: size, orientation: orient, width: w, height: h}
	if rec.Border {
		sh.border = newBorder(DefaultBorderDefinition())
	}
	if rec.TitleBlock != "" {
		sh.titleBlock = newTitleBlock(DefaultTitleBlockDefinition(), s.lookup)
	}
	s.items = append(s.items, sh)
	return nil
}

// restoreDims resolves a restored sheet's dimensions: the table value for a standard
// size, or the persisted laid-out dimensions for a custom size.
func (s *Sheets) restoreDims(size types.SheetSize, orient types.SheetOrientation, rec sheetRecipe) (float64, float64, error) {
	if size == types.SheetSizeCustom {
		if rec.WidthMM <= 0 || rec.HeightMM <= 0 {
			return 0, 0, fmt.Errorf("drawing: custom sheet %q has non-positive dimensions %g×%g mm", rec.Name, rec.WidthMM, rec.HeightMM)
		}
		return rec.WidthMM, rec.HeightMM, nil
	}
	return laidOutDimsMM(size, orient, 0, 0)
}

// setActiveIndex clamps the restored active-sheet index into range.
func (s *Sheets) setActiveIndex(i int) {
	if i < 0 || i >= len(s.items) {
		i = 0
	}
	s.active = i
}
