// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"
	"strconv"
	"strings"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Drawing views (M14-F02 PBI-139, #386): a view projects the drawing's referenced model onto
// the sheet. A base view projects from a standard orientation; a projected view derives its
// orientation from a base view plus a direction. The hidden-line engine (kernel/hlr) produces
// the drawing curves — visible (solid) and hidden (dashed) edge segments — which are placed on
// the sheet at the view's scale and centre. The kernel works in centimetres; the sheet in
// millimetres, so a 1:1 view scales model centimetres up by 10.
const cmToMM = 10.0

// DrawingCurve is one projected edge segment on the sheet (millimetres), classified visible or
// hidden, carrying the source model edge's reference key so the view stays associative across
// model recompute.
type DrawingCurve struct {
	A, B     math.Point2
	Visible  bool
	kind     types.DrawingCurveKind
	edgeKey  []byte
	edgeType types.DrawingEdgeType // model-edge role (tangent/thread/…); zero ⇒ ordinary sharp edge
}

// Start, End and IsVisible expose the curve geometry; EdgeKey is the source model edge key;
// Kind classifies the curve (edge/section/hatch/break) so the head can style it.
func (c DrawingCurve) Start() math.Point2           { return c.A }
func (c DrawingCurve) End() math.Point2             { return c.B }
func (c DrawingCurve) IsVisible() bool              { return c.Visible }
func (c DrawingCurve) Kind() types.DrawingCurveKind { return c.kind }
func (c DrawingCurve) EdgeKey() []byte              { return c.edgeKey }

// EdgeType is the model-edge role the curve came from (tangent/thread/bend…); zero ⇒ ordinary sharp
// edge (#1984).
func (c DrawingCurve) EdgeType() types.DrawingEdgeType { return c.edgeType }

// DrawingView is one view on a sheet: a base view (standard orientation) or a projected view
// (derived from a base view by a direction), at a scale/style/centre, holding the drawing
// curves the hidden-line engine produced for the referenced model.
type DrawingView struct {
	name        string
	viewType    types.DrawingViewType
	projected   bool
	baseView    string                // the parent view a projected/auxiliary/section view derives from
	foldAngle   float64               // auxiliary fold-line angle on the parent, radians
	section     sectionLine           // section-view cut line on the parent (sheet mm)
	sectionOpts hlr.SectionOptions    // section reverse / limited-depth (#1982)
	sectionType types.SectionViewType // section partial-cut kind (#1982)
	detail      detailBoundary        // detail-view circular boundary (parent model-2D)
	brk         breakBand             // break-view removed band (parent model-2D, along the break axis)
	draftW      float64               // draft-view frame width (sheet mm)
	draftH      float64               // draft-view frame height (sheet mm)
	orientation types.BaseViewOrientation
	direction   types.ProjectionDirection
	scale       float64
	style       types.DrawingViewStyle // authored style; FromBase resolves to the base at recompute
	effStyle    types.DrawingViewStyle // the style actually rendered this recompute (FromBase resolved, #1985)
	centerX     float64                // sheet millimetres
	centerY     float64
	// Label (#1983). labelText overrides the default caption; the hide* flags (zero ⇒ shown) drop
	// the label, its name, or its scale note; labelX/Y position the caption (sheet mm, 0 ⇒ below the
	// view center).
	labelText string
	hideLabel bool
	hideName  bool
	hideScale bool
	labelX    float64
	labelY    float64
	crops     []cropRegion // crop fences clipping this view (#1987)
	// hideTangentEdges drops smooth tangent edges (fillet/blend transitions) from the projection when
	// set; the zero value shows them (the default), so no view constructor needs to opt in (#1984).
	hideTangentEdges bool
	// Placement (#1988). rotation turns the view's curves about its centre (radians, CCW). alignedTo
	// locks the view to another view on alignment's axis (horizontal ⇒ shared Y, vertical ⇒ shared X);
	// justification records the centring mode. The zero values are unrotated / free / centred.
	rotation      float64
	alignedTo     string
	alignment     types.DrawingViewAlignment
	justification types.ViewJustification
	curves        []DrawingCurve
}

var (
	_ contract.DrawingView        = (*DrawingView)(nil)
	_ contract.SectionDrawingView = (*DrawingView)(nil)
	_ contract.DetailDrawingView  = (*DrawingView)(nil)
)

// Name, Type, ParentView, IsProjected, Orientation, Scale, Style, CenterMM and CurveCount
// satisfy contract.DrawingView.
func (v *DrawingView) Name() string                           { return v.name }
func (v *DrawingView) Type() types.DrawingViewType            { return v.viewType }
func (v *DrawingView) ParentView() string                     { return v.baseView }
func (v *DrawingView) IsProjected() bool                      { return v.projected }
func (v *DrawingView) Orientation() types.BaseViewOrientation { return v.orientation }
func (v *DrawingView) Scale() float64                         { return v.scale }
func (v *DrawingView) Style() types.DrawingViewStyle          { return v.style }
func (v *DrawingView) CenterMM() (x, y float64)               { return v.centerX, v.centerY }
func (v *DrawingView) CurveCount() int                        { return len(v.curves) }

// Direction is the projection direction (for a projected view).
func (v *DrawingView) Direction() types.ProjectionDirection { return v.direction }

// BaseViewName is the parent view a projected/auxiliary view derives from ("" for a base view).
func (v *DrawingView) BaseViewName() string { return v.baseView }

// FoldAngle is the auxiliary view's fold-line angle on its parent, in radians.
func (v *DrawingView) FoldAngle() float64 { return v.foldAngle }

// SectionLineMM returns the section view's cut line on its parent (sheet millimetres).
func (v *DrawingView) SectionLineMM() (x1, y1, x2, y2 float64) {
	return v.section.x1, v.section.y1, v.section.x2, v.section.y2
}

// SectionDepthMM is the retained-slab depth in millimetres, or 0 for a full through-cut. The
// kernel stores it in model centimetres, so it converts back on the way out (#1982).
func (v *DrawingView) SectionDepthMM() float64 { return v.sectionOpts.Depth * cmToMM }

// SectionReverse reports whether the far half is kept instead of the near half (#1982).
func (v *DrawingView) SectionReverse() bool { return v.sectionOpts.Reverse }

// SectionType is the partial-cut kind (none/quarter/half/threeQuarter, #1982).
func (v *DrawingView) SectionType() types.SectionViewType { return v.sectionType }

// DetailBoundaryMM returns the detail view's circular boundary on its parent (sheet millimetres):
// centre and radius, converted back from the parent's projection space.
func (v *DrawingView) DetailBoundaryMM() (cx, cy, r float64) {
	return v.detail.sheetCX, v.detail.sheetCY, v.detail.sheetR
}

// BreakOrientation returns the axis a break view compresses along.
func (v *DrawingView) BreakOrientation() types.BreakOrientation { return v.brk.orientation }

// BreakGapMM returns the removed band's start/end on the parent (sheet millimetres).
func (v *DrawingView) BreakGapMM() (start, end float64) { return v.brk.sheetG0, v.brk.sheetG1 }

// Curves returns the view's computed drawing curves.
func (v *DrawingView) Curves() []DrawingCurve { return v.curves }

// BoundsMM returns the view's 2D bounding box on the sheet (millimetres) from its curves, and
// false if it has none — the hit-test rectangle for selecting/right-clicking the view.
func (v *DrawingView) BoundsMM() (minX, minY, maxX, maxY float64, ok bool) {
	if len(v.curves) == 0 {
		return 0, 0, 0, 0, false
	}
	first := v.curves[0].A
	minX, minY = float64(first.X), float64(first.Y)
	maxX, maxY = minX, minY
	for _, c := range v.curves {
		for _, p := range [2]math.Point2{c.A, c.B} {
			minX, minY = min(minX, float64(p.X)), min(minY, float64(p.Y))
			maxX, maxY = max(maxX, float64(p.X)), max(maxY, float64(p.Y))
		}
	}
	return minX, minY, maxX, maxY, true
}

// VisibleHidden counts the view's visible and hidden curves.
func (v *DrawingView) VisibleHidden() (visible, hidden int) {
	for _, c := range v.curves {
		if c.Visible {
			visible++
		} else {
			hidden++
		}
	}
	return
}

// recompute runs hidden-line projection of body through basis and rebuilds the view's curves,
// placing each segment on the sheet at the view's scale and centre. A section view runs the
// clipped cut-away projection (cut outline + hatch); other views run plain HLR. In wireframe
// style every edge is drawn visible (no hidden-line removal).
func (v *DrawingView) recompute(body *topo.Body, basis hlr.View) {
	// A concrete style sets the effective style here so it holds even when a view is recomputed on
	// its own (e.g. right after AddBase); a FromBase style keeps whatever the collection pre-pass
	// resolved it to (#1985).
	if v.style != types.FromBaseViewStyle {
		v.effStyle = v.style
	}
	segs := v.project(body, basis)
	v.curves = make([]DrawingCurve, 0, len(segs))
	v.buildCurves(segs, tangentEdgeKeys(body))
	v.applyCrops() // clip the projected curves to any crop fences (#1987)
}

// buildCurves fills the view's curves from its projected segments, dispatched by view type: the
// break/slice/breakout views run their own compression/clip logic; every other type runs the
// plain per-segment placement (clipped to a detail boundary where present).
func (v *DrawingView) buildCurves(segs []hlr.Segment, tangent map[string]bool) {
	switch v.viewType {
	case types.DrawingViewBreak:
		v.recomputeBreak(segs)
	case types.DrawingViewSlice:
		v.recomputeSlice(segs)
	case types.DrawingViewBreakout:
		v.recomputeBreakout(segs)
	default:
		wireframe := v.style == types.WireframeViewStyle
		for _, s := range segs {
			edgeType := segEdgeType(s, tangent)
			if edgeType == types.TangentDrawingEdge && v.hideTangentEdges {
				continue // tangent display suppressed for this view (#1984)
			}
			a, b, ok := v.clip(s.A, s.B)
			if !ok {
				continue
			}
			v.appendEdgeCurve(a, b, wireframe || s.Visible, curveKind(s.Kind), s.EdgeKey, edgeType)
		}
	}
}

// segEdgeType classifies a projected segment by the role of its source model edge — currently
// tangent (a smooth fillet/blend transition) or ordinary; other roles are unknown for now (#1984).
func segEdgeType(s hlr.Segment, tangent map[string]bool) types.DrawingEdgeType {
	if len(s.EdgeKey) > 0 && tangent[string(s.EdgeKey)] {
		return types.TangentDrawingEdge
	}
	return types.UnknownDrawingEdge
}

// DraftSizeMM returns a draft view's frame size (sheet millimetres).
func (v *DrawingView) DraftSizeMM() (w, h float64) { return v.draftW, v.draftH }

// recomputeDraftFrame rebuilds a model-less draft view's rectangular frame directly in sheet
// millimetres (no projection); a placement preview/selection then has bounds to work with.
func (v *DrawingView) recomputeDraftFrame() {
	w, h := v.draftW/2, v.draftH/2
	x0, y0, x1, y1 := v.centerX-w, v.centerY-h, v.centerX+w, v.centerY+h
	corners := [4]math.Point2{
		math.P2(math.Scalar(x0), math.Scalar(y0)), math.P2(math.Scalar(x1), math.Scalar(y0)),
		math.P2(math.Scalar(x1), math.Scalar(y1)), math.P2(math.Scalar(x0), math.Scalar(y1)),
	}
	v.curves = v.curves[:0]
	for i := range 4 {
		v.curves = append(v.curves, DrawingCurve{A: corners[i], B: corners[(i+1)%4], Visible: true, kind: types.DrawingEdgeCurve})
	}
}

// appendCurve places a model-2D segment on the sheet and adds it to the view's curves.
func (v *DrawingView) appendCurve(a, b math.Point2, visible bool, kind types.DrawingCurveKind, edgeKey []byte) {
	v.appendEdgeCurve(a, b, visible, kind, edgeKey, types.UnknownDrawingEdge)
}

// appendEdgeCurve is appendCurve carrying the source edge's role (#1984).
func (v *DrawingView) appendEdgeCurve(a, b math.Point2, visible bool, kind types.DrawingCurveKind, edgeKey []byte, edgeType types.DrawingEdgeType) {
	// The hidden-line-removed style keeps only visible edges, so a hidden model edge produces no
	// curve at all (#1985). Non-edge curves (section fills, break glyphs) are always kept.
	if !visible && kind == types.DrawingEdgeCurve && v.effStyle.RemovesHiddenEdges() {
		return
	}
	v.curves = append(v.curves, DrawingCurve{A: v.place(a), B: v.place(b), Visible: visible, kind: kind, edgeKey: edgeKey, edgeType: edgeType})
}

// resolveEffectiveStyle sets the view's effective render style for this recompute, following a
// FromBase authored style to the named base view's style so a derived view tracks its parent
// associatively (#1985) without overwriting the authored style. A base view (or a missing parent)
// with FromBase falls back to the hidden-line default. lookup returns a view's authored style by name.
func (v *DrawingView) resolveEffectiveStyle(lookup func(name string) (types.DrawingViewStyle, bool)) {
	v.effStyle = v.style
	if v.style != types.FromBaseViewStyle {
		return
	}
	if s, ok := lookup(v.baseView); ok && s != types.FromBaseViewStyle {
		v.effStyle = s
		return
	}
	v.effStyle = types.HiddenLineViewStyle
}

// EffectiveStyle returns the style the view actually renders with (FromBase resolved), for reads.
func (v *DrawingView) EffectiveStyle() types.DrawingViewStyle { return v.effStyle }

// Label is the view's caption (#1983): the explicit override when set, else a default built from the
// view name and a scale note, each dropped by its show flag. An empty string means no label is drawn.
func (v *DrawingView) Label() string {
	if v.hideLabel {
		return ""
	}
	if v.labelText != "" {
		return v.labelText
	}
	parts := make([]string, 0, 2)
	if !v.hideName {
		parts = append(parts, v.name)
	}
	if !v.hideScale {
		parts = append(parts, "SCALE "+scaleNote(v.scale))
	}
	return strings.Join(parts, "  ")
}

// ShowLabel / ShowName / ShowScale report whether the label, its name, and its scale note are drawn.
func (v *DrawingView) ShowLabel() bool { return !v.hideLabel }
func (v *DrawingView) ShowName() bool  { return !v.hideName }
func (v *DrawingView) ShowScale() bool { return !v.hideScale }

// SetShowLabel / SetShowName / SetShowScale toggle the label and its parts.
func (v *DrawingView) SetShowLabel(show bool) { v.hideLabel = !show }
func (v *DrawingView) SetShowName(show bool)  { v.hideName = !show }
func (v *DrawingView) SetShowScale(show bool) { v.hideScale = !show }

// SetLabelText overrides the default caption ("" restores the default).
func (v *DrawingView) SetLabelText(text string) { v.labelText = text }

// LabelPositionMM is where the caption is placed (sheet mm); (0,0) means below the view center.
func (v *DrawingView) LabelPositionMM() (x, y float64) { return v.labelX, v.labelY }

// SetLabelPositionMM places the caption (sheet mm).
func (v *DrawingView) SetLabelPositionMM(x, y float64) { v.labelX, v.labelY = x, y }

// scaleNote formats a view scale as a drafting ratio: 1 → "1:1", 0.5 → "1:2", 2 → "2:1".
func scaleNote(scale float64) string {
	if scale <= 0 {
		return "1:1"
	}
	if scale >= 1 {
		return strconv.FormatFloat(scale, 'g', -1, 64) + ":1"
	}
	return "1:" + strconv.FormatFloat(1/scale, 'g', -1, 64)
}

// project runs the projection a view's type calls for: a section view's clipped cut-away, or
// plain hidden-line projection for every other kind.
func (v *DrawingView) project(body *topo.Body, basis hlr.View) []hlr.Segment {
	if v.viewType == types.DrawingViewSection || v.viewType == types.DrawingViewSlice {
		return hlr.ProjectSectionOpts(body, basis, ops.DefaultQuality(), v.sectionOpts)
	}
	return hlr.Project(body, basis, ops.DefaultQuality())
}

// clip restricts a projected segment (in the parent's model-2D) to a detail view's circular
// boundary; every other view kind passes the segment through unchanged.
func (v *DrawingView) clip(a, b math.Point2) (math.Point2, math.Point2, bool) {
	if v.viewType != types.DrawingViewDetail {
		return a, b, true
	}
	return clipToCircle(a, b, v.detail.cx, v.detail.cy, v.detail.r)
}

// curveKind maps a projected segment's kind to its drawing-curve classification.
func curveKind(k hlr.SegmentKind) types.DrawingCurveKind {
	switch k {
	case hlr.KindCut:
		return types.DrawingSectionCurve
	case hlr.KindHatch:
		return types.DrawingHatchCurve
	default:
		return types.DrawingEdgeCurve
	}
}

// SheetPointOfModelMM projects a 3D model point through the view's base orientation and places
// it on the sheet (millimetres) — used to position annotations (e.g. a centre-of-gravity marker)
// on the view. origin is the model's projection centre (its bounding-box centre).
func (v *DrawingView) SheetPointOfModelMM(p, origin math.Point3) math.Point2 {
	return v.place(hlr.ProjectPoint(baseBasis(v.orientation, origin), p))
}

// place maps a projected 2D point (model centimetres) to the sheet (millimetres) at the view's
// scale and centre.
func (v *DrawingView) place(p math.Point2) math.Point2 {
	s := math.Scalar(cmToMM * v.scale)
	rp := rotatePoint2(p, v.rotation) // view rotation turns the curves about the model-2D origin (#1988)
	return math.P2(math.Scalar(v.centerX)+rp.X*s, math.Scalar(v.centerY)+rp.Y*s)
}

// rotatePoint2 rotates a model-2D point about the origin by angle (radians, CCW). The projection is
// centred on the model-2D origin, which maps to the view centre, so this rotates the view's curves
// about that centre.
func rotatePoint2(p math.Point2, angle float64) math.Point2 {
	if angle == 0 {
		return p
	}
	sin, cos := stdmath.Sincos(angle)
	x, y := float64(p.X), float64(p.Y)
	return math.P2(math.Scalar(x*cos-y*sin), math.Scalar(x*sin+y*cos))
}

// baseBasis is the projection frame for a base view's standard orientation, centred on origin.
func baseBasis(orientation types.BaseViewOrientation, origin math.Point3) hlr.View {
	switch orientation {
	case types.BaseViewBack:
		return hlr.NewView(origin, math.V3(0, -1, 0), math.V3(0, 0, 1))
	case types.BaseViewTop:
		return hlr.NewView(origin, math.V3(0, 0, -1), math.V3(0, 1, 0))
	case types.BaseViewBottom:
		return hlr.NewView(origin, math.V3(0, 0, 1), math.V3(0, 1, 0))
	case types.BaseViewRight:
		return hlr.NewView(origin, math.V3(-1, 0, 0), math.V3(0, 0, 1))
	case types.BaseViewLeft:
		return hlr.NewView(origin, math.V3(1, 0, 0), math.V3(0, 0, 1))
	case types.BaseViewIso:
		return hlr.NewView(origin, math.V3(-1, 1, -1), math.V3(0, 0, 1))
	default: // BaseViewFront
		return hlr.NewView(origin, math.V3(0, 1, 0), math.V3(0, 0, 1))
	}
}

// projectedBasis derives a projected view's frame from its base view's frame and the
// projection direction (re-using the base's axes so shared edges align between the views).
func projectedBasis(base hlr.View, dir types.ProjectionDirection, origin math.Point3) hlr.View {
	switch dir {
	case types.ProjectLeft:
		return hlr.NewView(origin, base.XAxis.Negate(), base.YAxis)
	case types.ProjectUp:
		return hlr.NewView(origin, base.YAxis.Negate(), base.ViewDir)
	case types.ProjectDown:
		return hlr.NewView(origin, base.YAxis, base.ViewDir.Negate())
	default: // ProjectRight
		return hlr.NewView(origin, base.XAxis, base.YAxis)
	}
}

// bodyCenter is the centre of a body's bounding box — the projection origin, so the view is
// centred on the model.
func bodyCenter(body *topo.Body) math.Point3 {
	verts := body.Vertices()
	if len(verts) == 0 {
		return math.P3(0, 0, 0)
	}
	lo, hi := verts[0].Point(), verts[0].Point()
	for _, vt := range verts {
		p := vt.Point()
		lo = math.P3(min(lo.X, p.X), min(lo.Y, p.Y), min(lo.Z, p.Z))
		hi = math.P3(max(hi.X, p.X), max(hi.Y, p.Y), max(hi.Z, p.Z))
	}
	return math.P3((lo.X+hi.X)/2, (lo.Y+hi.Y)/2, (lo.Z+hi.Z)/2)
}
