// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Drawing view generation — BASE and PROJECTED views (M48 #2227 split of views.go). A base view
// projects the referenced model at a standard orientation; a projected view folds off a base view in
// one of the orthographic directions, inheriting its scale and style. This file holds their specs,
// adders, cursor previews and in-place edits.

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
