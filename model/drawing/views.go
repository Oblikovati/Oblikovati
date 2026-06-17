// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
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
		name: name, projected: true, baseView: spec.BaseView, orientation: base.orientation,
		direction: spec.Direction, style: base.style, scale: base.scale,
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
		return fmt.Errorf("drawing: no view named %q", name)
	}
	v.orientation, v.style, v.scale, v.centerX, v.centerY = orientation, style, positiveScale(scale), cx, cy
	vs.Recompute()
	return nil
}

// EditProjected updates a projected view's direction/centre and re-projects it.
func (vs *DrawingViews) EditProjected(name string, dir types.ProjectionDirection, cx, cy float64) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view named %q", name)
	}
	v.direction, v.centerX, v.centerY = dir, cx, cy
	vs.Recompute()
	return nil
}

// Recompute re-projects every view against the current referenced model — the associativity
// path after a model edit. With no resolvable model it leaves the views untouched.
func (vs *DrawingViews) Recompute() {
	body, ok := vs.resolveBody()
	if !ok {
		return
	}
	origin := bodyCenter(body)
	for _, v := range vs.items {
		if !v.projected {
			v.recompute(body, baseBasis(v.orientation, origin))
			continue
		}
		if base, ok := vs.ByName(v.baseView); ok {
			v.recompute(body, projectedBasis(baseBasis(base.orientation, origin), v.direction, origin))
		}
	}
}

// Count, Item and ByName read the collection.
func (vs *DrawingViews) Count() int { return len(vs.items) }

func (vs *DrawingViews) Item(i int) *DrawingView {
	if i < 0 || i >= len(vs.items) {
		return nil
	}
	return vs.items[i]
}

func (vs *DrawingViews) ByName(name string) (*DrawingView, bool) {
	for _, v := range vs.items {
		if v.name == name {
			return v, true
		}
	}
	return nil, false
}

// Remove deletes the named view; removing a base view also removes the views projected from
// it (they have no base to derive from).
func (vs *DrawingViews) Remove(name string) error {
	if _, ok := vs.ByName(name); !ok {
		return fmt.Errorf("drawing: no view named %q", name)
	}
	kept := vs.items[:0]
	for _, v := range vs.items {
		if v.name == name || (v.projected && v.baseView == name) {
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
