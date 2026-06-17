// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

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
	Standard       string        `yaml:"standard,omitempty"` // active drafting standard ("" ⇒ ISO)
	ActiveSheet    int           `yaml:"activeSheet,omitempty"`
	Sheets         []sheetRecipe `yaml:"sheets,omitempty"`
}

// sheetRecipe is the YAML shape of one sheet. WidthMM/HeightMM are written only for a
// custom size (a standard size's dimensions come from the size+orientation on open).
type sheetRecipe struct {
	Name        string             `yaml:"name"`
	Size        string             `yaml:"size"`
	Orientation string             `yaml:"orientation,omitempty"`
	WidthMM     float64            `yaml:"widthMm,omitempty"`
	HeightMM    float64            `yaml:"heightMm,omitempty"`
	Border      bool               `yaml:"border"`
	TitleBlock  string             `yaml:"titleBlock,omitempty"` // definition name; "" ⇒ no title block
	Views       []viewRecipe       `yaml:"views,omitempty"`
	Annotations []annotationRecipe `yaml:"annotations,omitempty"`
}

// annotationRecipe is the YAML shape of one drawing annotation. A CoG marker's glyph re-derives
// from its view's centroid on open; a revision cloud's scallops re-derive from its rectangle.
type annotationRecipe struct {
	Name     string  `yaml:"name"`
	Kind     string  `yaml:"kind"`
	ViewName string  `yaml:"viewName,omitempty"`
	X        float64 `yaml:"xmm,omitempty"`
	Y        float64 `yaml:"ymm,omitempty"`
	W        float64 `yaml:"widthMm,omitempty"`
	H        float64 `yaml:"heightMm,omitempty"`
	Tag      string  `yaml:"tag,omitempty"`
}

// viewRecipe is the YAML shape of one drawing view. The drawing curves are not stored — they
// are re-projected from the referenced model on open, so a view always reflects the current
// model.
type viewRecipe struct {
	Name         string     `yaml:"name"`
	Type         string     `yaml:"type,omitempty"` // DrawingViewType ("" ⇒ base, or projected via Projected)
	Projected    bool       `yaml:"projected,omitempty"`
	BaseView     string     `yaml:"baseView,omitempty"`
	Orientation  string     `yaml:"orientation,omitempty"`
	Direction    string     `yaml:"direction,omitempty"`
	FoldAngleDeg float64    `yaml:"foldAngleDeg,omitempty"` // auxiliary fold-line angle on the parent
	SectionLine  [4]float64 `yaml:"sectionLine,omitempty"`  // section cut line on the parent (sheet mm)
	Detail       [3]float64 `yaml:"detail,omitempty"`       // detail boundary on the parent: centreX, centreY, radius (sheet mm)
	BreakOrient  string     `yaml:"breakOrient,omitempty"`  // break orientation (horizontal/vertical)
	BreakGap     [2]float64 `yaml:"breakGap,omitempty"`     // break band on the parent: start, end (sheet mm)
	DraftSize    [2]float64 `yaml:"draftSize,omitempty"`    // draft frame: width, height (sheet mm)
	Scale        float64    `yaml:"scale,omitempty"`
	Style        string     `yaml:"style,omitempty"`
	CenterX      float64    `yaml:"centerXmm,omitempty"`
	CenterY      float64    `yaml:"centerYmm,omitempty"`
}

// MarshalRecipe renders the drawing's sheets and referenced model as YAML
// (doc.RecipeContent).
func (c *Content) MarshalRecipe() ([]byte, error) {
	r := drawingRecipe{ModelReference: c.modelRef, Standard: c.styles.active.String(), ActiveSheet: c.sheets.active}
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
	if std, ok := types.ParseDraftingStandard(r.Standard); ok {
		c.styles.SetActiveStandard(std)
	}
	c.sheets = newSheets()
	c.sheets.lookup = c.resolveProperty
	c.sheets.bodyResolve = c.resolveBody
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
	for _, v := range sh.views.items {
		rec.Views = append(rec.Views, viewRecipeOf(v))
	}
	if sh.annotations != nil {
		for _, a := range sh.annotations.items {
			rec.Annotations = append(rec.Annotations, annotationRecipe{
				Name: a.name, Kind: a.kind.String(), ViewName: a.viewName,
				X: a.x, Y: a.y, W: a.w, H: a.h, Tag: a.tag,
			})
		}
	}
	return rec
}

// viewRecipeOf snapshots one view's definition (its curves are re-projected on open).
func viewRecipeOf(v *DrawingView) viewRecipe {
	return viewRecipe{
		Name: v.name, Type: v.viewType.String(), Projected: v.projected, BaseView: v.baseView,
		Orientation: v.orientation.String(), Direction: v.direction.String(),
		FoldAngleDeg: v.foldAngle * 180 / math.Pi,
		SectionLine:  [4]float64{v.section.x1, v.section.y1, v.section.x2, v.section.y2},
		Detail:       [3]float64{v.detail.sheetCX, v.detail.sheetCY, v.detail.sheetR},
		BreakOrient:  v.brk.orientation.String(),
		BreakGap:     [2]float64{v.brk.sheetG0, v.brk.sheetG1},
		DraftSize:    [2]float64{v.draftW, v.draftH},
		Scale:        v.scale, Style: v.style.String(), CenterX: v.centerX, CenterY: v.centerY,
	}
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
	sh := &Sheet{name: rec.Name, size: size, orientation: orient, width: w, height: h, views: newDrawingViews(s.bodyResolve)}
	if rec.Border {
		sh.border = newBorder(DefaultBorderDefinition())
	}
	if rec.TitleBlock != "" {
		sh.titleBlock = newTitleBlock(DefaultTitleBlockDefinition(), s.lookup)
	}
	for _, vr := range rec.Views {
		sh.views.items = append(sh.views.items, restoreView(vr))
	}
	restoreAnnotations(sh, rec.Annotations)
	s.items = append(s.items, sh)
	return nil
}

// restoreAnnotations rebuilds a sheet's annotations from its recipe; CoG glyphs re-derive on the
// next RecomputeViews, revision-cloud scallops re-derive now from the rectangle.
func restoreAnnotations(sh *Sheet, recs []annotationRecipe) {
	if len(recs) == 0 {
		return
	}
	as := sh.Annotations()
	for _, ar := range recs {
		kind, _ := types.ParseDrawingAnnotationKind(ar.Kind)
		a := &DrawingAnnotation{name: ar.Name, kind: kind, viewName: ar.ViewName, x: ar.X, y: ar.Y, w: ar.W, h: ar.H, tag: ar.Tag}
		if kind == types.RevisionCloudAnnotation {
			a.curves = revisionCloudCurves(ar.X, ar.Y, ar.W, ar.H)
		}
		as.items = append(as.items, a)
	}
}

// restoreView rebuilds a view's definition from its recipe; its curves are re-projected by
// the next RecomputeViews (once the referenced model resolves).
func restoreView(vr viewRecipe) *DrawingView {
	orient, _ := types.ParseBaseViewOrientation(vr.Orientation)
	dir, _ := types.ParseProjectionDirection(vr.Direction)
	style, _ := types.ParseDrawingViewStyle(vr.Style)
	vt := restoredViewType(vr)
	sl, dt, bg, ds := vr.SectionLine, vr.Detail, vr.BreakGap, vr.DraftSize
	brkOrient, _ := types.ParseBreakOrientation(vr.BreakOrient)
	return &DrawingView{
		name: vr.Name, viewType: vt, projected: vt == types.DrawingViewProjected, baseView: vr.BaseView,
		foldAngle: vr.FoldAngleDeg * math.Pi / 180, section: sectionLine{sl[0], sl[1], sl[2], sl[3]},
		detail: detailBoundary{sheetCX: dt[0], sheetCY: dt[1], sheetR: dt[2]},
		brk:    breakBand{orientation: brkOrient, sheetG0: bg[0], sheetG1: bg[1]},
		draftW: ds[0], draftH: ds[1],
		orientation: orient, direction: dir,
		scale: positiveScale(vr.Scale), style: style, centerX: vr.CenterX, centerY: vr.CenterY,
	}
}

// restoredViewType resolves a recipe's view type, falling back to the Projected flag for
// recipes written before the type discriminator existed.
func restoredViewType(vr viewRecipe) types.DrawingViewType {
	if vt, ok := types.ParseDrawingViewType(vr.Type); ok {
		return vt
	}
	if vr.Projected {
		return types.DrawingViewProjected
	}
	return types.DrawingViewBase
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
