// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Drawing-recipe — the SHEET / FORMAT section (M48 #2226 split of recipe.go). The YAML shape of one
// sheet (size, orientation, border/zones, title block, revision) and the snapshot/restore of the
// sheet frame; the sheet's views/dimensions/annotations/sketches are delegated to their own recipe_*
// sections. A standard size derives its dimensions from size+orientation on open; a custom size takes
// the persisted laid-out dimensions verbatim.

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
	// Sheet authoring (#1989): revision string, zoned-border grid + label modes, title-block corner.
	Revision     string             `yaml:"revision,omitempty"`
	BorderHZones int                `yaml:"borderHZones,omitempty"`
	BorderVZones int                `yaml:"borderVZones,omitempty"`
	BorderHLabel string             `yaml:"borderHLabel,omitempty"`
	BorderVLabel string             `yaml:"borderVLabel,omitempty"`
	TitleBlockAt string             `yaml:"titleBlockAt,omitempty"`
	Views        []viewRecipe       `yaml:"views,omitempty"`
	Annotations  []annotationRecipe `yaml:"annotations,omitempty"`
	Dimensions   []dimensionRecipe  `yaml:"dimensions,omitempty"`
	Sketches     []sketchRecipeItem `yaml:"sketches,omitempty"`
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
		rec.TitleBlockAt = sh.titleBlock.location.String()
	}
	rec.Revision = sh.revision
	if sh.border != nil && sh.border.def.hZones > 0 {
		rec.BorderHZones, rec.BorderVZones = sh.border.def.hZones, sh.border.def.vZones
		rec.BorderHLabel, rec.BorderVLabel = sh.border.def.hLabelMode.String(), sh.border.def.vLabelMode.String()
	}
	for _, v := range sh.views.items {
		rec.Views = append(rec.Views, viewRecipeOf(v))
	}
	rec.Annotations = annotationRecipesOf(sh)
	rec.Dimensions = dimensionRecipesOf(sh)
	rec.Sketches = sketchRecipesOf(sh)
	return rec
}

// restoreSheetContents rebuilds a restored sheet's views, annotations, dimensions and sketches.
func restoreSheetContents(sh *Sheet, rec sheetRecipe) {
	for _, vr := range rec.Views {
		sh.views.items = append(sh.views.items, restoreView(vr))
	}
	restoreAnnotations(sh, rec.Annotations)
	restoreDimensions(sh, rec.Dimensions)
	restoreSketches(sh, rec.Sketches)
}

// restoreDims resolves a restored sheet's dimensions: the table value for a standard
// size, or the persisted laid-out dimensions for a custom size.
// restoreBorder rebuilds a sheet's border from its recipe — a zoned border when the recipe carries a
// zone grid, otherwise the plain default (#1989).
func restoreBorder(rec sheetRecipe) *Border {
	if rec.BorderHZones <= 0 || rec.BorderVZones <= 0 {
		return newBorder(DefaultBorderDefinition())
	}
	hMode, _ := types.ParseBorderLabelMode(rec.BorderHLabel)
	vMode, _ := types.ParseBorderLabelMode(rec.BorderVLabel)
	return newBorder(ZonedBorderDefinition(rec.BorderHZones, rec.BorderVZones, hMode, vMode))
}

func (s *Sheets) restoreDims(size types.SheetSize, orient types.SheetOrientation, rec sheetRecipe) (float64, float64, error) {
	if size == types.SheetSizeCustom {
		if rec.WidthMM <= 0 || rec.HeightMM <= 0 {
			return 0, 0, fmt.Errorf("drawing: custom sheet %q has non-positive dimensions %g×%g mm", rec.Name, rec.WidthMM, rec.HeightMM)
		}
		return rec.WidthMM, rec.HeightMM, nil
	}
	return laidOutDimsMM(size, orient, 0, 0)
}
