// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
)

// Drawing-recipe — the VIEWS section (M48 #2226 split of recipe.go). The YAML shape of one drawing
// view and its crop fences (the drawing curves are never stored — they re-project from the referenced
// model on open), plus the snapshot/restore of that section.

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
	// Section options (#1982): the retained-slab depth (model cm; 0 ⇒ full), reverse flag and
	// partial-cut type. SectionLine above carries the cut line.
	SectionDepth   float64 `yaml:"sectionDepth,omitempty"`
	SectionReverse bool    `yaml:"sectionReverse,omitempty"`
	SectionType    string  `yaml:"sectionType,omitempty"`
	// Label overrides (#1983): a caption override and the hide flags / caption position.
	LabelText string  `yaml:"labelText,omitempty"`
	HideLabel bool    `yaml:"hideLabel,omitempty"`
	HideName  bool    `yaml:"hideName,omitempty"`
	HideScale bool    `yaml:"hideScale,omitempty"`
	LabelX    float64 `yaml:"labelXmm,omitempty"`
	LabelY    float64 `yaml:"labelYmm,omitempty"`
	// Crop fences clipping the view (#1987).
	Crops []cropRecipe `yaml:"crops,omitempty"`
	// HideTangentEdges drops smooth tangent edges from the projection (#1984); default shows them.
	HideTangentEdges bool `yaml:"hideTangentEdges,omitempty"`
	// Placement (#1988): rotation about the view centre (degrees), alignment lock to another view,
	// and centring mode.
	RotationDeg   float64 `yaml:"rotationDeg,omitempty"`
	AlignedTo     string  `yaml:"alignedTo,omitempty"`
	Alignment     string  `yaml:"alignment,omitempty"`
	Justification string  `yaml:"justification,omitempty"`
}

// cropRecipe is the YAML shape of one crop fence: a rectangle (X0,Y0)-(X1,Y1) or a circle
// (CircleX,CircleY,Radius), all sheet mm, plus the break-mark boundary spelling (#1987).
type cropRecipe struct {
	Circle    bool    `yaml:"circle,omitempty"`
	X0        float64 `yaml:"x0,omitempty"`
	Y0        float64 `yaml:"y0,omitempty"`
	X1        float64 `yaml:"x1,omitempty"`
	Y1        float64 `yaml:"y1,omitempty"`
	CircleX   float64 `yaml:"circleXmm,omitempty"`
	CircleY   float64 `yaml:"circleYmm,omitempty"`
	Radius    float64 `yaml:"radiusMm,omitempty"`
	BreakMark string  `yaml:"breakMark,omitempty"`
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
		SectionDepth: v.sectionOpts.Depth, SectionReverse: v.sectionOpts.Reverse, SectionType: v.sectionType.String(),
		LabelText: v.labelText, HideLabel: v.hideLabel, HideName: v.hideName, HideScale: v.hideScale,
		LabelX: v.labelX, LabelY: v.labelY, Crops: cropRecipesOf(v.crops),
		HideTangentEdges: v.hideTangentEdges,
		RotationDeg:      v.RotationDeg(), AlignedTo: v.alignedTo, Alignment: alignmentString(v),
		Justification: justificationString(v),
	}
}

// alignmentString persists a view's alignment lock, and only a real lock — "" for a free (in-position)
// view so the common case stays out of the recipe.
func alignmentString(v *DrawingView) string {
	if !v.IsAligned() {
		return ""
	}
	return v.alignment.String()
}

// justificationString persists a view's centring mode, and only a non-default one.
func justificationString(v *DrawingView) string {
	if v.justification == types.CenteredViewJustification {
		return ""
	}
	return v.justification.String()
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
	sectionType, _ := types.ParseSectionViewType(vr.SectionType)
	return &DrawingView{
		name: vr.Name, viewType: vt, projected: vt == types.DrawingViewProjected, baseView: vr.BaseView,
		foldAngle: vr.FoldAngleDeg * math.Pi / 180, section: sectionLine{sl[0], sl[1], sl[2], sl[3]},
		sectionOpts: hlr.SectionOptions{Reverse: vr.SectionReverse, Depth: vr.SectionDepth}, sectionType: sectionType,
		detail: detailBoundary{sheetCX: dt[0], sheetCY: dt[1], sheetR: dt[2]},
		brk:    breakBand{orientation: brkOrient, sheetG0: bg[0], sheetG1: bg[1]},
		draftW: ds[0], draftH: ds[1],
		orientation: orient, direction: dir,
		scale: positiveScale(vr.Scale), style: style, centerX: vr.CenterX, centerY: vr.CenterY,
		labelText: vr.LabelText, hideLabel: vr.HideLabel, hideName: vr.HideName, hideScale: vr.HideScale,
		labelX: vr.LabelX, labelY: vr.LabelY, crops: cropRegionsFrom(vr.Crops),
		hideTangentEdges: vr.HideTangentEdges,
		rotation:         vr.RotationDeg * math.Pi / 180,
		alignedTo:        vr.AlignedTo, alignment: parseAlignment(vr.Alignment), justification: parseJustification(vr.Justification),
	}
}

// parseAlignment resolves a persisted alignment spelling, defaulting to in-position (free).
func parseAlignment(s string) types.DrawingViewAlignment {
	a, _ := types.ParseDrawingViewAlignment(s)
	return a
}

// parseJustification resolves a persisted justification spelling, defaulting to centered.
func parseJustification(s string) types.ViewJustification {
	j, _ := types.ParseViewJustification(s)
	return j
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
