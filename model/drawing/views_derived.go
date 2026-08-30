// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/types"
)

// Drawing view generation — the other DERIVED views (M48 #2227 split of views.go): auxiliary (folded
// off a parent), slice (zero-thickness cut), breakout (interior revealed inside a region), break
// (compressed by removing a band) and the model-less draft frame. addDerived is the shared adder for
// the region-based kinds; this file holds each kind's spec, adder and cursor preview.

// AuxiliaryViewSpec describes an auxiliary view folded off a parent view.
type AuxiliaryViewSpec struct {
	Name             string
	ParentView       string
	FoldAngleRad     float64
	CenterX, CenterY float64
}

// AddAuxiliary adds a view projected perpendicular to a fold line on parentView, inheriting the
// parent's scale and style. The parent must be a base view (auxiliaries fold off an orthographic
// base, like projected views).
func (vs *DrawingViews) AddAuxiliary(spec AuxiliaryViewSpec) (*DrawingView, error) {
	parent, body, err := vs.parentBaseFor(spec.ParentView)
	if err != nil {
		return nil, err
	}
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	v := &DrawingView{
		name: name, viewType: types.DrawingViewAuxiliary, baseView: spec.ParentView,
		foldAngle: spec.FoldAngleRad, orientation: parent.orientation, style: parent.style,
		scale: parent.scale, centerX: spec.CenterX, centerY: spec.CenterY,
	}
	origin := bodyCenter(body)
	v.recompute(body, auxiliaryBasis(baseBasis(parent.orientation, origin), spec.FoldAngleRad, origin))
	vs.items = append(vs.items, v)
	return v, nil
}

// BreakViewSpec describes a break view: the parent compressed by removing a band along an axis.
// GapStart/GapEnd bound the removed band on the parent (sheet mm) along the break axis.
type BreakViewSpec struct {
	Name             string
	ParentView       string
	Orientation      types.BreakOrientation
	GapStart, GapEnd float64 // on the parent (sheet mm)
	CenterX, CenterY float64
}

// SliceViewSpec describes a slice view: the zero-thickness cut at a section line on the parent.
type SliceViewSpec struct {
	Name             string
	ParentView       string
	X1, Y1, X2, Y2   float64 // section line on the parent (sheet mm)
	CenterX, CenterY float64
}

// BreakoutViewSpec describes a breakout view: a parent copy with the interior revealed inside a
// circular region (sheet mm) on the parent.
type BreakoutViewSpec struct {
	Name             string
	ParentView       string
	BoundaryX        float64
	BoundaryY        float64
	RadiusMM         float64
	CenterX, CenterY float64
}

// DraftViewSpec describes a model-less framed draft view (sheet mm).
type DraftViewSpec struct {
	Name              string
	WidthMM, HeightMM float64
	CenterX, CenterY  float64
}

// addDerived builds a parent-derived view: it resolves the parent base view + model, assigns a
// unique name, configures the view via setup (which sets the type and its payload), projects it
// and adds it. Shared by the slice/breakout adders (mirroring section/detail/auxiliary).
func (vs *DrawingViews) addDerived(parentView, name string, setup func(parent *DrawingView) *DrawingView) (*DrawingView, error) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, err
	}
	n, err := vs.uniqueName(name)
	if err != nil {
		return nil, err
	}
	v := setup(parent)
	v.name, v.baseView = n, parentView
	if basis, ok := vs.basisFor(v, bodyCenter(body)); ok {
		v.recompute(body, basis)
	}
	vs.items = append(vs.items, v)
	return v, nil
}

// AddSlice adds a slice view: the zero-thickness cut outline at a section line on the parent.
func (vs *DrawingViews) AddSlice(spec SliceViewSpec) (*DrawingView, error) {
	return vs.addDerived(spec.ParentView, spec.Name, func(parent *DrawingView) *DrawingView {
		return &DrawingView{
			viewType: types.DrawingViewSlice, section: sectionLine{spec.X1, spec.Y1, spec.X2, spec.Y2},
			orientation: parent.orientation, style: parent.style, scale: parent.scale,
			centerX: spec.CenterX, centerY: spec.CenterY,
		}
	})
}

// AddBreakout adds a breakout view: the parent projection with the interior revealed inside the
// circular region.
func (vs *DrawingViews) AddBreakout(spec BreakoutViewSpec) (*DrawingView, error) {
	return vs.addDerived(spec.ParentView, spec.Name, func(parent *DrawingView) *DrawingView {
		return &DrawingView{
			viewType: types.DrawingViewBreakout, detail: detailBoundaryOf(parent, spec.BoundaryX, spec.BoundaryY, spec.RadiusMM),
			orientation: parent.orientation, style: parent.style, scale: parent.scale,
			centerX: spec.CenterX, centerY: spec.CenterY,
		}
	})
}

// AddDraft adds a model-less framed draft view (a container for manual 2D geometry).
func (vs *DrawingViews) AddDraft(spec DraftViewSpec) (*DrawingView, error) {
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	v := &DrawingView{
		name: name, viewType: types.DrawingViewDraft, scale: 1, centerX: spec.CenterX, centerY: spec.CenterY,
		draftW: positiveScale(spec.WidthMM), draftH: positiveScale(spec.HeightMM),
	}
	v.recomputeDraftFrame()
	vs.items = append(vs.items, v)
	return v, nil
}

// PreviewSlice returns the origin-placed curves a slice view at the given line on parentView
// would produce, without adding it.
func (vs *DrawingViews) PreviewSlice(parentView string, x1, y1, x2, y2 float64) ([]DrawingCurve, bool) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, false
	}
	v := &DrawingView{viewType: types.DrawingViewSlice, section: sectionLine{x1, y1, x2, y2}, style: parent.style, scale: parent.scale}
	origin := bodyCenter(body)
	v.recompute(body, sectionBasis(baseBasis(parent.orientation, origin), v.section, parent.scale, parent.centerX, parent.centerY, origin))
	return v.curves, true
}

// PreviewBreakout returns the origin-placed curves a breakout view of the given region on
// parentView would produce, without adding it.
func (vs *DrawingViews) PreviewBreakout(parentView string, boundaryX, boundaryY, radiusMM float64) ([]DrawingCurve, bool) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, false
	}
	v := &DrawingView{
		viewType: types.DrawingViewBreakout, detail: detailBoundaryOf(parent, boundaryX, boundaryY, radiusMM),
		style: parent.style, scale: parent.scale,
	}
	v.recompute(body, baseBasis(parent.orientation, bodyCenter(body)))
	return v.curves, true
}

// AddBreak adds a break view: the parent's projection with the band removed and the two sides
// butted together (a zigzag break line at the join). The parent must be a base view.
func (vs *DrawingViews) AddBreak(spec BreakViewSpec) (*DrawingView, error) {
	parent, body, err := vs.parentBaseFor(spec.ParentView)
	if err != nil {
		return nil, err
	}
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	v := &DrawingView{
		name: name, viewType: types.DrawingViewBreak, baseView: spec.ParentView,
		brk:         breakBandOf(parent, spec.Orientation, spec.GapStart, spec.GapEnd),
		orientation: parent.orientation, style: parent.style, scale: parent.scale,
		centerX: spec.CenterX, centerY: spec.CenterY,
	}
	v.recompute(body, baseBasis(parent.orientation, bodyCenter(body)))
	vs.items = append(vs.items, v)
	return v, nil
}

// PreviewBreak returns the origin-placed curves a break view of parentView (band removed) would
// produce, without adding it.
func (vs *DrawingViews) PreviewBreak(parentView string, orientation types.BreakOrientation, gapStart, gapEnd float64) ([]DrawingCurve, bool) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, false
	}
	v := &DrawingView{
		viewType: types.DrawingViewBreak, style: parent.style, scale: parent.scale,
		brk: breakBandOf(parent, orientation, gapStart, gapEnd),
	}
	v.recompute(body, baseBasis(parent.orientation, bodyCenter(body)))
	return v.curves, true
}

// PreviewAuxiliary returns the origin-centred curves an auxiliary view folded off parentView at
// foldAngleRad would produce, without adding it. ok is false when the parent or model is missing.
func (vs *DrawingViews) PreviewAuxiliary(parentView string, foldAngleRad float64) ([]DrawingCurve, bool) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, false
	}
	origin := bodyCenter(body)
	v := &DrawingView{viewType: types.DrawingViewAuxiliary, style: parent.style, scale: parent.scale}
	v.recompute(body, auxiliaryBasis(baseBasis(parent.orientation, origin), foldAngleRad, origin))
	return v.curves, true
}
