// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The Fill Surface tool (M36-F07) closes a four-sided opening bounded by the last four surface bodies
// with a single clean NURBS, meeting each neighbour at the chosen continuity. It is parameter-only —
// the four bounding surfaces are the running input — the Class-A boundary-fill move.

// fillContinuityLabels label the continuity orders for the dialog (index = continuity order).
var fillContinuityLabels = []string{"Position (G0)", "Tangent (G1)", "Curvature (G2)"}

// FillSurfaceTool fills the opening bounded by the last Sides surface bodies.
type FillSurfaceTool struct {
	dialogTool
	continuity int // index into fillContinuityLabels (order = index)
	sides      int // number of bounding surfaces (3, 4, 5, …); 4 is the classic four-sided fill
	added      *feature.PartFeature
}

// NewFillSurfaceTool returns a fill tool defaulting to a four-sided curvature (G2) fill.
func NewFillSurfaceTool() *FillSurfaceTool {
	return &FillSurfaceTool{continuity: 2, sides: feature.DefaultFillSides}
}

// SetContinuity sets the continuity order (0=G0, 1=G1, 2=G2).
func (t *FillSurfaceTool) SetContinuity(order int) { t.continuity = order }

// Name implements [Tool].
func (t *FillSurfaceTool) Name() string { return "Fill Surface" }

// Prompt guides the input.
func (t *FillSurfaceTool) Prompt(*Session) string {
	return "Fill the opening bounded by the last N surfaces: set the side count and continuity, then OK."
}

// Params exposes the continuity and the number of bounding sides for the generic dialog.
func (t *FillSurfaceTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{
			{Label: "Sides", Get: func() int { return t.sides }, Set: func(v int) { t.sides = v }},
		},
		Choices: []ChoiceParam{
			{Label: "Continuity", Options: fillContinuityLabels, Get: func() int { return t.continuity }, Set: func(v int) { t.continuity = v }},
		},
	}
}

// CanCommit reports whether the continuity choice is in range and there are at least three sides.
func (t *FillSurfaceTool) CanCommit() bool {
	return t.continuity >= 0 && t.continuity < len(fillContinuityLabels) && t.sides >= 3
}

// Commit fills the opening and recomputes.
func (t *FillSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewFillFeatures(part.Features()).AddSides(t.continuity, t.sides)
	part.Recompute()
	s.recordEdit(part, "Fill Surface")
	if !t.added.Health().OK() {
		return errors.New("fill surface: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FillSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }
