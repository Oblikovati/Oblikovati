// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/types"
)

// Drawing view generation — DETAIL views (M48 #2227 split of views.go). A detail view magnifies a
// circular region of a parent view: the parent's projection is clipped to the boundary circle and
// re-placed at a larger scale. This file holds its spec, adder and cursor preview.

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
