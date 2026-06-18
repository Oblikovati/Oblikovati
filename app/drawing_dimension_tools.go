// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
)

// The drawing-dimension tool (M14-F03 PBI-141, #388): place a linear dimension across a base
// view's overall extent (horizontal, vertical or the diagonal). The value is the true model size
// and updates with the model. A free two-pick dimension is a follow-up; an overall dimension is
// the common first case and fits the dialog-driven tool flow.

// dimensionTypeLabels indexes the Type dropdown — order matches dimensionPlacement.
var dimensionTypeLabels = []string{"Horizontal", "Vertical", "Aligned"}

// LinearDimensionTool dimensions a chosen base view's overall size in the chosen direction.
type LinearDimensionTool struct {
	derivedViewTool
	dimType int
}

// NewLinearDimensionTool creates the tool; its base-view list is captured on Start.
func NewLinearDimensionTool() *LinearDimensionTool { return &LinearDimensionTool{} }

func (t *LinearDimensionTool) Name() string { return "Linear Dimension" }

// Commit dimensions the selected base view's overall extent in the chosen direction.
func (t *LinearDimensionTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	parent := t.parent()
	if parent == "" {
		return fmt.Errorf("drawing: no base view to dimension — add a base view first")
	}
	minX, minY, maxX, maxY, ok := viewBoundsMM(s, parent)
	if !ok {
		return fmt.Errorf("drawing: base view %q has no geometry to dimension", parent)
	}
	dimType, x1, y1, x2, y2, offset := dimensionPlacement(t.dimType, minX, minY, maxX, maxY)
	if _, err := c.Sheets().Active().Dimensions().AddLinear("", parent, dimType, x1, y1, x2, y2, offset); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view and dimension-type choices for the property dialog.
func (t *LinearDimensionTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		t.baseChoice("Base View"),
		{Label: "Type", Options: dimensionTypeLabels, Get: func() int { return t.dimType }, Set: func(i int) { t.dimType = i }},
	}}
}

// radialDimensionTypes indexes the Type dropdown for the radial tool.
var radialDimensionTypes = []struct {
	label string
	typ   types.DrawingDimensionType
}{
	{"Radius", types.RadiusDimension}, {"Diameter", types.DiameterDimension},
}

// RadialDimensionTool dimensions every circular edge (hole) in a chosen base view as a radius or
// diameter callout — the common "dimension all holes" action.
type RadialDimensionTool struct {
	derivedViewTool
	typeIdx int
}

// NewRadialDimensionTool creates the tool; its base-view list is captured on Start.
func NewRadialDimensionTool() *RadialDimensionTool { return &RadialDimensionTool{} }

func (t *RadialDimensionTool) Name() string { return "Radial Dimension" }

// Commit dimensions every distinct circular edge in the selected base view.
func (t *RadialDimensionTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	parent := t.parent()
	if parent == "" {
		return fmt.Errorf("drawing: no base view to dimension — add a base view first")
	}
	dimType := radialDimensionTypes[clampIndex(t.typeIdx, len(radialDimensionTypes))].typ
	if _, err := c.Sheets().Active().Dimensions().AddRadialForEachCircle(parent, dimType); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view and radius/diameter choices for the property dialog.
func (t *RadialDimensionTool) Params() ToolParams {
	labels := make([]string, len(radialDimensionTypes))
	for i, r := range radialDimensionTypes {
		labels[i] = r.label
	}
	return ToolParams{Choices: []ChoiceParam{
		t.baseChoice("Base View"),
		{Label: "Type", Options: labels, Get: func() int { return t.typeIdx }, Set: func(i int) { t.typeIdx = i }},
	}}
}

// AngularDimensionTool dimensions the corner angle between the first two non-parallel straight
// edges in a chosen base view (e.g. a 90° corner or a bevel angle).
type AngularDimensionTool struct {
	derivedViewTool
}

// NewAngularDimensionTool creates the tool; its base-view list is captured on Start.
func NewAngularDimensionTool() *AngularDimensionTool { return &AngularDimensionTool{} }

func (t *AngularDimensionTool) Name() string { return "Angular Dimension" }

// Commit dimensions the angle between the selected base view's first two non-parallel edges.
func (t *AngularDimensionTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	parent := t.parent()
	if parent == "" {
		return fmt.Errorf("drawing: no base view to dimension — add a base view first")
	}
	if _, err := c.Sheets().Active().Dimensions().AddAngularForFirstCorner(parent); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view choice for the property dialog.
func (t *AngularDimensionTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{t.baseChoice("Base View")}}
}

// dimensionPlacement maps the Type index + a view's bounds (sheet mm) to the two pick points and
// the dimension-line offset: horizontal across the width below the view, vertical down the height
// to its right, or aligned along the diagonal. The pick points snap to the nearest model vertices.
func dimensionPlacement(idx int, minX, minY, maxX, maxY float64) (t types.DrawingDimensionType, x1, y1, x2, y2, offset float64) {
	midX, midY := (minX+maxX)/2, (minY+maxY)/2
	const gap = 12.0
	switch idx {
	case 1: // vertical — measured points up the height at mid-width, dimension line to the right
		return types.VerticalDimension, midX, minY, midX, maxY, -((maxX - midX) + gap)
	case 2: // aligned — the corner-to-corner diagonal
		return types.AlignedDimension, minX, minY, maxX, maxY, 0
	default: // horizontal — measured points across the width at mid-height, dimension line below
		return types.HorizontalDimension, minX, midY, maxX, midY, -((midY - minY) + gap)
	}
}
