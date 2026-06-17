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
