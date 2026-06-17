// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
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
	A, B    math.Point2
	Visible bool
	kind    types.DrawingCurveKind
	edgeKey []byte
}

// Start, End and IsVisible expose the curve geometry; EdgeKey is the source model edge key;
// Kind classifies the curve (edge/section/hatch/break) so the head can style it.
func (c DrawingCurve) Start() math.Point2           { return c.A }
func (c DrawingCurve) End() math.Point2             { return c.B }
func (c DrawingCurve) IsVisible() bool              { return c.Visible }
func (c DrawingCurve) Kind() types.DrawingCurveKind { return c.kind }
func (c DrawingCurve) EdgeKey() []byte              { return c.edgeKey }

// DrawingView is one view on a sheet: a base view (standard orientation) or a projected view
// (derived from a base view by a direction), at a scale/style/centre, holding the drawing
// curves the hidden-line engine produced for the referenced model.
type DrawingView struct {
	name        string
	viewType    types.DrawingViewType
	projected   bool
	baseView    string      // the parent view a projected/auxiliary/section view derives from
	foldAngle   float64     // auxiliary fold-line angle on the parent, radians
	section     sectionLine // section-view cut line on the parent (sheet mm)
	orientation types.BaseViewOrientation
	direction   types.ProjectionDirection
	scale       float64
	style       types.DrawingViewStyle
	centerX     float64 // sheet millimetres
	centerY     float64
	curves      []DrawingCurve
}

var (
	_ contract.DrawingView        = (*DrawingView)(nil)
	_ contract.SectionDrawingView = (*DrawingView)(nil)
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
			minX, minY = minF(minX, float64(p.X)), minF(minY, float64(p.Y))
			maxX, maxY = maxFl(maxX, float64(p.X)), maxFl(maxY, float64(p.Y))
		}
	}
	return minX, minY, maxX, maxY, true
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFl(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
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
	segs := hlr.Project(body, basis, ops.DefaultQuality())
	if v.viewType == types.DrawingViewSection {
		segs = hlr.ProjectSection(body, basis, ops.DefaultQuality())
	}
	v.curves = make([]DrawingCurve, 0, len(segs))
	wireframe := v.style == types.WireframeViewStyle
	for _, s := range segs {
		v.curves = append(v.curves, DrawingCurve{
			A: v.place(s.A), B: v.place(s.B), Visible: wireframe || s.Visible,
			kind: curveKind(s.Kind), edgeKey: s.EdgeKey,
		})
	}
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

// place maps a projected 2D point (model centimetres) to the sheet (millimetres) at the view's
// scale and centre.
func (v *DrawingView) place(p math.Point2) math.Point2 {
	s := math.Scalar(cmToMM * v.scale)
	return math.P2(math.Scalar(v.centerX)+p.X*s, math.Scalar(v.centerY)+p.Y*s)
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
		lo = math.P3(minS(lo.X, p.X), minS(lo.Y, p.Y), minS(lo.Z, p.Z))
		hi = math.P3(maxS(hi.X, p.X), maxS(hi.Y, p.Y), maxS(hi.Z, p.Z))
	}
	return math.P3((lo.X+hi.X)/2, (lo.Y+hi.Y)/2, (lo.Z+hi.Z)/2)
}

func minS(a, b math.Scalar) math.Scalar {
	if a < b {
		return a
	}
	return b
}

func maxS(a, b math.Scalar) math.Scalar {
	if a > b {
		return a
	}
	return b
}
