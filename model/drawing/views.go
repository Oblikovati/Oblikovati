// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/internal/collview"
)

// bodyLookup resolves the drawing's referenced model to its B-rep body for projection. It is
// the seam between this package (which knows nothing of the workspace) and the host, which
// finds the referenced model document and returns its body. A nil hook (no resolver wired) or
// an unresolved reference yields (nil, false), so view creation reports a clear error.
type bodyLookup func() (*topo.Body, bool)

// BaseViewSpec describes a base view to add.
type BaseViewSpec struct {
	Name             string
	Orientation      types.BaseViewOrientation
	Style            types.DrawingViewStyle
	Scale            float64
	CenterX, CenterY float64 // sheet millimetres
}

// ProjectedViewSpec describes a view projected from an existing base view.
type ProjectedViewSpec struct {
	Name             string
	BaseView         string
	Direction        types.ProjectionDirection
	CenterX, CenterY float64
}

// DrawingViews is a sheet's ordered, named view collection. It holds the body-resolution hook
// so it can project the referenced model when a view is added or recomputed.
type DrawingViews struct {
	items []*DrawingView
	body  bodyLookup
}

func newDrawingViews(body bodyLookup) *DrawingViews { return &DrawingViews{body: body} }

// AddBase adds a base view of the referenced model at the given orientation, computing its
// hidden-line curves. It errors if the drawing has no resolvable referenced model.
func (vs *DrawingViews) AddBase(spec BaseViewSpec) (*DrawingView, error) {
	body, ok := vs.resolveBody()
	if !ok {
		return nil, fmt.Errorf("drawing: no referenced model to project (set a model reference first)")
	}
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	v := &DrawingView{
		name: name, orientation: spec.Orientation, style: spec.Style,
		scale: positiveScale(spec.Scale), centerX: spec.CenterX, centerY: spec.CenterY,
	}
	origin := bodyCenter(body)
	v.recompute(body, baseBasis(spec.Orientation, origin))
	vs.items = append(vs.items, v)
	return v, nil
}

// AddProjected adds a view projected from baseView in the given direction, inheriting the
// base's scale and style.
func (vs *DrawingViews) AddProjected(spec ProjectedViewSpec) (*DrawingView, error) {
	base, ok := vs.ByName(spec.BaseView)
	if !ok {
		return nil, fmt.Errorf("drawing: no base view %q to project from", spec.BaseView)
	}
	if base.projected {
		return nil, fmt.Errorf("drawing: %q is a projected view; project from a base view", spec.BaseView)
	}
	body, ok := vs.resolveBody()
	if !ok {
		return nil, fmt.Errorf("drawing: no referenced model to project")
	}
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	v := &DrawingView{
		name: name, viewType: types.DrawingViewProjected, projected: true, baseView: spec.BaseView,
		orientation: base.orientation, direction: spec.Direction, style: base.style, scale: base.scale,
		centerX: spec.CenterX, centerY: spec.CenterY,
	}
	origin := bodyCenter(body)
	v.recompute(body, projectedBasis(baseBasis(base.orientation, origin), spec.Direction, origin))
	vs.items = append(vs.items, v)
	return v, nil
}

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

// SectionViewSpec describes a section view cut from a parent view by a section line. The line
// endpoints are in sheet millimetres, drawn on the parent view. Depth (>0, millimetres) limits
// the retained half to a slab that deep; Reverse keeps the opposite half; Type selects a partial
// cut (#1982).
type SectionViewSpec struct {
	Name             string
	ParentView       string
	X1, Y1, X2, Y2   float64 // section line on the parent (sheet mm)
	CenterX, CenterY float64 // where the section view sits on the sheet
	Depth            float64 // >0 ⇒ limited-depth slab (millimetres); 0 ⇒ full through-cut
	Reverse          bool    // keep the far half instead of the near half
	Type             types.SectionViewType
}

// AddSection adds a section view: the parent's referenced model cut by the plane through the
// section line (perpendicular to the parent), with the near half removed, the cut outline drawn
// bold and the exposed faces hatched. The parent must be a base view.
func (vs *DrawingViews) AddSection(spec SectionViewSpec) (*DrawingView, error) {
	parent, body, err := vs.parentBaseFor(spec.ParentView)
	if err != nil {
		return nil, err
	}
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	line := sectionLine{spec.X1, spec.Y1, spec.X2, spec.Y2}
	v := &DrawingView{
		name: name, viewType: types.DrawingViewSection, baseView: spec.ParentView, section: line,
		orientation: parent.orientation, style: parent.style, scale: parent.scale,
		centerX: spec.CenterX, centerY: spec.CenterY,
		// The kernel cuts in model centimetres; the request's depth is millimetres (#1982).
		sectionOpts: hlr.SectionOptions{Reverse: spec.Reverse, Depth: spec.Depth / cmToMM},
		sectionType: spec.Type,
	}
	origin := bodyCenter(body)
	v.recompute(body, sectionBasis(baseBasis(parent.orientation, origin), line, parent.scale, parent.centerX, parent.centerY, origin))
	vs.items = append(vs.items, v)
	return v, nil
}

// PreviewSection returns the origin-placed curves a section view through the given line on
// parentView would produce, without adding it. ok is false when the parent or model is missing.
func (vs *DrawingViews) PreviewSection(parentView string, x1, y1, x2, y2 float64) ([]DrawingCurve, bool) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, false
	}
	line := sectionLine{x1, y1, x2, y2}
	v := &DrawingView{viewType: types.DrawingViewSection, style: parent.style, scale: parent.scale}
	origin := bodyCenter(body)
	v.recompute(body, sectionBasis(baseBasis(parent.orientation, origin), line, parent.scale, parent.centerX, parent.centerY, origin))
	return v.curves, true
}

// DetailViewSpec describes a detail view: a magnified circular region of a parent view. The
// boundary centre/radius are in sheet millimetres on the parent; Scale is the (larger) detail
// scale.
type DetailViewSpec struct {
	Name             string
	ParentView       string
	BoundaryX        float64 // circle centre on the parent (sheet mm)
	BoundaryY        float64
	RadiusMM         float64 // circle radius on the parent (sheet mm)
	Scale            float64
	CenterX, CenterY float64 // where the detail view sits on the sheet
}

// AddDetail adds a detail view: the parent's projection clipped to a circular boundary and
// re-placed at a larger scale. The parent must be a base view.
func (vs *DrawingViews) AddDetail(spec DetailViewSpec) (*DrawingView, error) {
	parent, body, err := vs.parentBaseFor(spec.ParentView)
	if err != nil {
		return nil, err
	}
	name, err := vs.uniqueName(spec.Name)
	if err != nil {
		return nil, err
	}
	v := &DrawingView{
		name: name, viewType: types.DrawingViewDetail, baseView: spec.ParentView,
		detail:      detailBoundaryOf(parent, spec.BoundaryX, spec.BoundaryY, spec.RadiusMM),
		orientation: parent.orientation, style: parent.style, scale: positiveScale(spec.Scale),
		centerX: spec.CenterX, centerY: spec.CenterY,
	}
	v.recompute(body, baseBasis(parent.orientation, bodyCenter(body)))
	vs.items = append(vs.items, v)
	return v, nil
}

// PreviewDetail returns the origin-placed curves a detail view of the given region on parentView
// at the given scale would produce, without adding it.
func (vs *DrawingViews) PreviewDetail(parentView string, boundaryX, boundaryY, radiusMM, scale float64) ([]DrawingCurve, bool) {
	parent, body, err := vs.parentBaseFor(parentView)
	if err != nil {
		return nil, false
	}
	v := &DrawingView{
		viewType: types.DrawingViewDetail, style: parent.style, scale: positiveScale(scale),
		detail: detailBoundaryOf(parent, boundaryX, boundaryY, radiusMM),
	}
	v.recompute(body, baseBasis(parent.orientation, bodyCenter(body)))
	return v.curves, true
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

// parentBaseFor resolves a derived view's parent (which must be a base view) and the referenced
// model body — the common precondition for auxiliary/section/detail views.
func (vs *DrawingViews) parentBaseFor(parentView string) (*DrawingView, *topo.Body, error) {
	parent, ok := vs.ByName(parentView)
	if !ok {
		return nil, nil, fmt.Errorf("drawing: no parent view %q to derive from", parentView)
	}
	if parent.viewType != types.DrawingViewBase {
		return nil, nil, fmt.Errorf("drawing: %q is not a base view; derive from a base view", parentView)
	}
	body, ok := vs.resolveBody()
	if !ok {
		return nil, nil, fmt.Errorf("drawing: no referenced model to project")
	}
	return parent, body, nil
}

// PreviewBase returns the drawing curves a base view of the given orientation/style/scale
// would produce, centred at the origin (0,0), WITHOUT adding a view — the geometry a
// placement preview follows under the cursor. ok is false when no model resolves.
func (vs *DrawingViews) PreviewBase(orientation types.BaseViewOrientation, style types.DrawingViewStyle, scale float64) ([]DrawingCurve, bool) {
	body, ok := vs.resolveBody()
	if !ok {
		return nil, false
	}
	v := &DrawingView{orientation: orientation, style: style, scale: positiveScale(scale)}
	v.recompute(body, baseBasis(orientation, bodyCenter(body)))
	return v.curves, true
}

// PreviewProjected returns the origin-centred curves a view projected from baseName in dir
// would produce, without adding it. ok is false when the base view or model is missing.
func (vs *DrawingViews) PreviewProjected(baseName string, dir types.ProjectionDirection) ([]DrawingCurve, bool) {
	base, ok := vs.ByName(baseName)
	if !ok || base.projected {
		return nil, false
	}
	body, ok := vs.resolveBody()
	if !ok {
		return nil, false
	}
	origin := bodyCenter(body)
	v := &DrawingView{projected: true, baseView: baseName, orientation: base.orientation, direction: dir, style: base.style, scale: base.scale}
	v.recompute(body, projectedBasis(baseBasis(base.orientation, origin), dir, origin))
	return v.curves, true
}

// EditBase updates a base view's orientation/style/scale/centre and re-projects it.
func (vs *DrawingViews) EditBase(name string, orientation types.BaseViewOrientation, style types.DrawingViewStyle, scale, cx, cy float64) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf(errNoViewNamed, name)
	}
	v.orientation, v.style, v.scale, v.centerX, v.centerY = orientation, style, positiveScale(scale), cx, cy
	vs.Recompute()
	return nil
}

// EditProjected updates a projected view's direction/centre and re-projects it.
func (vs *DrawingViews) EditProjected(name string, dir types.ProjectionDirection, cx, cy float64) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf(errNoViewNamed, name)
	}
	v.direction, v.centerX, v.centerY = dir, cx, cy
	vs.Recompute()
	return nil
}

// Recompute re-projects every view against the current referenced model — the associativity
// path after a model edit. Draft views (no model) refresh their frame regardless; the
// model-backed views are left untouched when no model resolves.
func (vs *DrawingViews) Recompute() {
	vs.resolveEffectiveStyles()
	body, ok := vs.resolveBody()
	for _, v := range vs.items {
		if v.viewType == types.DrawingViewDraft {
			v.recomputeDraftFrame()
			continue
		}
		if !ok {
			continue
		}
		if basis, ok := vs.basisFor(v, bodyCenter(body)); ok {
			v.recompute(body, basis)
		}
	}
}

// ViewLabelStyle carries the optional view-label overrides to apply (#1983); a nil field is left
// unchanged. XMM and YMM are applied together (both must be set to reposition the caption).
type ViewLabelStyle struct {
	Text      *string
	ShowLabel *bool
	ShowName  *bool
	ShowScale *bool
	XMM       *float64
	YMM       *float64
}

// SetLabel applies the given label overrides to the named view (#1983), erroring when no view
// carries that name.
func (vs *DrawingViews) SetLabel(name string, style ViewLabelStyle) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view named %q", name)
	}
	if style.Text != nil {
		v.SetLabelText(*style.Text)
	}
	if style.ShowLabel != nil {
		v.SetShowLabel(*style.ShowLabel)
	}
	if style.ShowName != nil {
		v.SetShowName(*style.ShowName)
	}
	if style.ShowScale != nil {
		v.SetShowScale(*style.ShowScale)
	}
	if style.XMM != nil && style.YMM != nil {
		v.SetLabelPositionMM(*style.XMM, *style.YMM)
	}
	return nil
}

// resolveEffectiveStyles resolves each view's FromBase style to its base view's style before the
// projection pass, so a derived view renders with its parent's style associatively (#1985).
func (vs *DrawingViews) resolveEffectiveStyles() {
	lookup := func(name string) (types.DrawingViewStyle, bool) {
		if base, ok := vs.ByName(name); ok {
			return base.style, true
		}
		return 0, false
	}
	for _, v := range vs.items {
		v.resolveEffectiveStyle(lookup)
	}
}

// basisFor returns the projection frame a view uses, dispatching on its type. A base view
// projects its standard orientation; a derived view derives from its parent's frame (and yields
// ok=false if that parent is missing, so it is left as-is rather than mis-projected).
func (vs *DrawingViews) basisFor(v *DrawingView, origin math.Point3) (hlr.View, bool) {
	if v.viewType == types.DrawingViewBase {
		return baseBasis(v.orientation, origin), true
	}
	base, ok := vs.ByName(v.baseView)
	if !ok {
		return hlr.View{}, false
	}
	parent := baseBasis(base.orientation, origin)
	switch v.viewType {
	case types.DrawingViewProjected:
		return projectedBasis(parent, v.direction, origin), true
	case types.DrawingViewAuxiliary:
		return auxiliaryBasis(parent, v.foldAngle, origin), true
	case types.DrawingViewSection, types.DrawingViewSlice:
		return sectionBasis(parent, v.section, base.scale, base.centerX, base.centerY, origin), true
	default: // detail/breakout/break reuse the parent projection (recompute clips/breaks); re-map
		// their sheet-mm region against the parent's current placement so they track a rescale/move.
		v.refreshParentRegion(base)
		return parent, true
	}
}

// refreshParentRegion re-derives a detail/break view's region (clip circle or break band) from
// its persisted sheet-mm definition against the parent's current placement — the associativity
// path when the parent is rescaled or moved.
func (v *DrawingView) refreshParentRegion(parent *DrawingView) {
	switch v.viewType {
	case types.DrawingViewDetail, types.DrawingViewBreakout:
		v.detail = detailBoundaryOf(parent, v.detail.sheetCX, v.detail.sheetCY, v.detail.sheetR)
	case types.DrawingViewBreak:
		v.brk = breakBandOf(parent, v.brk.orientation, v.brk.sheetG0, v.brk.sheetG1)
	}
}

// Count, Item and ByName read the collection.
func (vs *DrawingViews) Count() int { return len(vs.items) }

func (vs *DrawingViews) Item(i int) *DrawingView { return collview.At(vs.items, i) }

func (vs *DrawingViews) ByName(name string) (*DrawingView, bool) {
	for _, v := range vs.items {
		if v.name == name {
			return v, true
		}
	}
	return nil, false
}

// Remove deletes the named view; removing a base view also removes the views derived from it
// (projected/auxiliary/…), which have no parent left to derive from.
func (vs *DrawingViews) Remove(name string) error {
	if _, ok := vs.ByName(name); !ok {
		return fmt.Errorf(errNoViewNamed, name)
	}
	kept := vs.items[:0]
	for _, v := range vs.items {
		if v.name == name || v.baseView == name {
			continue
		}
		kept = append(kept, v)
	}
	vs.items = kept
	return nil
}

func (vs *DrawingViews) resolveBody() (*topo.Body, bool) {
	if vs.body == nil {
		return nil, false
	}
	return vs.body()
}

// uniqueName returns the requested name (erroring if taken) or the next auto "VIEW:N".
func (vs *DrawingViews) uniqueName(requested string) (string, error) {
	if requested == "" {
		return vs.nextName(), nil
	}
	if _, exists := vs.ByName(requested); exists {
		return "", fmt.Errorf("drawing: view %q already exists", requested)
	}
	return requested, nil
}

func (vs *DrawingViews) nextName() string {
	for n := len(vs.items) + 1; ; n++ {
		name := fmt.Sprintf("VIEW:%d", n)
		if _, exists := vs.ByName(name); !exists {
			return name
		}
	}
}

func positiveScale(s float64) float64 {
	if s <= 0 {
		return 1
	}
	return s
}
