// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
)

// Drawing view generation — SECTION views (M48 #2227 split of views.go). A section view cuts the
// parent's referenced model by the plane through a section line (perpendicular to the parent), removes
// the near half, draws the cut outline bold and hatches the exposed faces. This file holds its spec,
// adder and cursor preview.

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
