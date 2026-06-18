// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

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
	Dimensions  []dimensionRecipe  `yaml:"dimensions,omitempty"`
}

// dimensionRecipe is the YAML shape of one drawing dimension. The glyph and measured value are
// not stored — they re-derive on open from the attached vertices (KeyA/KeyB, hex of each vertex's
// reference key), which re-bind to the current model, so a dimension always reflects it.
type dimensionRecipe struct {
	Name     string  `yaml:"name"`
	Type     string  `yaml:"type,omitempty"`
	ViewName string  `yaml:"viewName"`
	KeyA     string  `yaml:"keyA,omitempty"`     // linear: first attached vertex
	KeyB     string  `yaml:"keyB,omitempty"`     // linear: second attached vertex
	EdgeKey  string  `yaml:"edgeKey,omitempty"`  // radial: circular edge; angular: first straight edge
	EdgeKeyB string  `yaml:"edgeKeyB,omitempty"` // angular: second straight edge
	Offset   float64 `yaml:"offsetMm,omitempty"`
	TextDX   float64 `yaml:"textDxMm,omitempty"` // user text nudge (drag-the-text)
	TextDY   float64 `yaml:"textDyMm,omitempty"`
	AxisHorz bool    `yaml:"axisHorizontal,omitempty"` // ordinate: measure the view-X offset, else view-Y
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
	EdgeKey  string  `yaml:"edgeKey,omitempty"` // centre mark: the circular edge it marks
	// feature control frame (GD&T):
	Characteristic string   `yaml:"characteristic,omitempty"`
	Tolerance      string   `yaml:"tolerance,omitempty"`
	Datums         []string `yaml:"datums,omitempty"`
	// surface texture: the material-removal variant (the roughness value reuses Tag).
	MaterialRemoval string `yaml:"materialRemoval,omitempty"`
	// revision table: the user-supplied change-history rows.
	Revisions []revisionRowRecipe `yaml:"revisions,omitempty"`
	// custom table: the column headers and data rows.
	Headers []string   `yaml:"headers,omitempty"`
	Rows    [][]string `yaml:"rows,omitempty"`
}

// revisionRowRecipe is the YAML shape of one revision-table row.
type revisionRowRecipe struct {
	Revision    string `yaml:"revision"`
	Date        string `yaml:"date,omitempty"`
	Description string `yaml:"description,omitempty"`
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
	rec.Annotations = annotationRecipesOf(sh)
	rec.Dimensions = dimensionRecipesOf(sh)
	return rec
}

// annotationRecipesOf snapshots a sheet's annotations for persistence.
func annotationRecipesOf(sh *Sheet) []annotationRecipe {
	if sh.annotations == nil {
		return nil
	}
	out := make([]annotationRecipe, 0, len(sh.annotations.items))
	for _, a := range sh.annotations.items {
		out = append(out, annotationRecipe{
			Name: a.name, Kind: a.kind.String(), ViewName: a.viewName,
			X: a.x, Y: a.y, W: a.w, H: a.h, Tag: a.tag, EdgeKey: hex.EncodeToString(a.edgeKey),
			Characteristic: a.characteristic.String(), Tolerance: a.tolerance, Datums: a.datums,
			MaterialRemoval: a.materialRemoval.String(), Revisions: revisionRowRecipesOf(a.revisions),
			Headers: a.headers, Rows: a.tableRows,
		})
	}
	return out
}

// revisionRowRecipesOf snapshots a revision table's rows for persistence.
func revisionRowRecipesOf(rows []RevisionRow) []revisionRowRecipe {
	if len(rows) == 0 {
		return nil
	}
	out := make([]revisionRowRecipe, len(rows))
	for i, r := range rows {
		out[i] = revisionRowRecipe(r)
	}
	return out
}

// revisionRowsOf rebuilds a revision table's rows from its recipe.
func revisionRowsOf(recs []revisionRowRecipe) []RevisionRow {
	if len(recs) == 0 {
		return nil
	}
	out := make([]RevisionRow, len(recs))
	for i, r := range recs {
		out[i] = RevisionRow(r)
	}
	return out
}

// dimensionRecipesOf snapshots a sheet's dimensions for persistence (the attached vertex keys as
// hex; the glyph and value re-derive on open).
func dimensionRecipesOf(sh *Sheet) []dimensionRecipe {
	if sh.dimensions == nil {
		return nil
	}
	out := make([]dimensionRecipe, 0, len(sh.dimensions.items))
	for _, d := range sh.dimensions.items {
		out = append(out, dimensionRecipe{
			Name: d.name, Type: d.dimType.String(), ViewName: d.viewName,
			KeyA: hex.EncodeToString(d.keyA), KeyB: hex.EncodeToString(d.keyB),
			EdgeKey: hex.EncodeToString(d.edgeKey), EdgeKeyB: hex.EncodeToString(d.edgeKeyB),
			Offset: d.offset, TextDX: d.textDX, TextDY: d.textDY, AxisHorz: d.axisHorizontal,
		})
	}
	return out
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
	sh := &Sheet{name: rec.Name, size: size, orientation: orient, width: w, height: h, views: newDrawingViews(s.bodyResolve), bomResolve: s.bomResolve}
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
	restoreDimensions(sh, rec.Dimensions)
	s.items = append(s.items, sh)
	return nil
}

// restoreDimensions rebuilds a sheet's dimensions from its recipe; each re-binds its attached
// vertices and re-measures on the next RecomputeViews (once the referenced model resolves).
func restoreDimensions(sh *Sheet, recs []dimensionRecipe) {
	if len(recs) == 0 {
		return
	}
	ds := sh.Dimensions()
	for _, dr := range recs {
		dimType, _ := types.ParseDrawingDimensionType(dr.Type)
		keyA, _ := hex.DecodeString(dr.KeyA)
		keyB, _ := hex.DecodeString(dr.KeyB)
		edgeKey, _ := hex.DecodeString(dr.EdgeKey)
		edgeKeyB, _ := hex.DecodeString(dr.EdgeKeyB)
		d := &DrawingDimension{
			name: dr.Name, dimType: dimType, viewName: dr.ViewName,
			keyA: keyA, keyB: keyB, edgeKey: edgeKey, edgeKeyB: edgeKeyB,
			offset: dr.Offset, textDX: dr.TextDX, textDY: dr.TextDY, axisHorizontal: dr.AxisHorz,
		}
		ds.recompute(d)
		ds.items = append(ds.items, d)
	}
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
		edgeKey, _ := hex.DecodeString(ar.EdgeKey)
		characteristic, _ := types.ParseGeometricCharacteristic(ar.Characteristic)
		a := &DrawingAnnotation{name: ar.Name, kind: kind, viewName: ar.ViewName, x: ar.X, y: ar.Y, w: ar.W, h: ar.H, tag: ar.Tag, edgeKey: edgeKey,
			characteristic: characteristic, tolerance: ar.Tolerance, datums: ar.Datums, revisions: revisionRowsOf(ar.Revisions),
			headers: ar.Headers, tableRows: ar.Rows}
		restoreAnnotationGeometry(a, ar)
		as.items = append(as.items, a)
	}
}

// restoreAnnotationGeometry rebuilds a restored annotation's curves and labels from its recipe (the
// kinds whose glyph is a pure function of persisted fields; the model-associative ones re-derive on
// the next RecomputeViews instead).
func restoreAnnotationGeometry(a *DrawingAnnotation, ar annotationRecipe) {
	switch a.kind {
	case types.RevisionCloudAnnotation:
		a.curves = revisionCloudCurves(ar.X, ar.Y, ar.W, ar.H)
	case types.FeatureControlFrameAnnotation:
		a.curves, a.labels = featureControlFrameGeometry(ar.X, ar.Y, a.characteristic, ar.Tolerance, ar.Datums)
	case types.DatumFeatureAnnotation:
		a.curves, a.labels = datumFeatureGeometry(ar.X, ar.Y, ar.Tag)
	case types.SurfaceTextureAnnotation:
		variant, _ := types.ParseMaterialRemoval(ar.MaterialRemoval)
		a.materialRemoval = variant
		a.curves, a.labels = surfaceTextureGeometry(ar.X, ar.Y, ar.Tag, variant)
	case types.BalloonAnnotation:
		item, _ := strconv.Atoi(ar.Tag)
		a.curves, a.labels = balloonGeometry(ar.X, ar.Y, item, ar.W, ar.H)
	case types.RevisionTagAnnotation:
		a.curves, a.labels = revisionTagGeometry(ar.X, ar.Y, ar.Tag)
	case types.DrawingNoteAnnotation:
		a.curves, a.labels = noteGeometry(ar.X, ar.Y, ar.Tag, ar.W, ar.H)
	default:
		restoreAnnotationTableGeometry(a, ar)
	}
}

// restoreAnnotationTableGeometry rebuilds the user-supplied table annotations (revision and custom)
// from their persisted rows — split out of restoreAnnotationGeometry to keep each focused.
func restoreAnnotationTableGeometry(a *DrawingAnnotation, ar annotationRecipe) {
	switch a.kind {
	case types.RevisionTableAnnotation:
		a.rowCount = len(a.revisions)
		a.curves, a.labels = revisionTableGeometry(ar.X, ar.Y, a.revisions)
	case types.CustomTableAnnotation:
		a.rowCount = len(a.tableRows)
		a.curves, a.labels = customTableGeometry(ar.X, ar.Y, a.headers, a.tableRows)
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
