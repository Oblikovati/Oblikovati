// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The Bridge Surface tool (M36-F09) connects the last two surface bodies with a clean NURBS
// transition meeting each at a chosen continuity (G0/G1/G2 per side) — the Class-A "connect these
// two panels" move. It is parameter-only (the two bounding surfaces are the running input).

// bridgeContinuityLabels label the per-side continuity orders for the dialog (index = order).
var bridgeContinuityLabels = []string{"Position (G0)", "Tangent (G1)", "Curvature (G2)"}

// BridgeSurfaceTool connects the last two surface bodies with a transition surface.
type BridgeSurfaceTool struct {
	dialogTool
	sideA int // index into bridgeContinuityLabels (order = index)
	sideB int
	added *feature.PartFeature
}

// NewBridgeSurfaceTool returns a bridge tool defaulting to curvature (G2) on both sides.
func NewBridgeSurfaceTool() *BridgeSurfaceTool { return &BridgeSurfaceTool{sideA: 2, sideB: 2} }

// SetContinuity sets both sides' continuity orders (for drivers/tests).
func (t *BridgeSurfaceTool) SetContinuity(a, b int) { t.sideA, t.sideB = a, b }

// Name implements [Tool].
func (t *BridgeSurfaceTool) Name() string { return "Bridge Surface" }

// Prompt guides the input.
func (t *BridgeSurfaceTool) Prompt(*Session) string {
	return "Bridge the last two surfaces: pick the continuity for each side, then OK."
}

// Params exposes the per-side continuity for the generic dialog.
func (t *BridgeSurfaceTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{
			{Label: "Side A", Options: bridgeContinuityLabels, Get: func() int { return t.sideA }, Set: func(v int) { t.sideA = v }},
			{Label: "Side B", Options: bridgeContinuityLabels, Get: func() int { return t.sideB }, Set: func(v int) { t.sideB = v }},
		},
	}
}

// CanCommit reports whether both continuity choices are in range.
func (t *BridgeSurfaceTool) CanCommit() bool {
	return t.inRange(t.sideA) && t.inRange(t.sideB)
}

func (t *BridgeSurfaceTool) inRange(i int) bool { return i >= 0 && i < len(bridgeContinuityLabels) }

// Commit bridges the two surfaces and recomputes.
func (t *BridgeSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewBridgeFeatures(part.Features()).Add(t.sideA, t.sideB)
	part.Recompute()
	s.recordEdit(part, "Bridge Surface")
	if !t.added.Health().OK() {
		return errors.New("bridge surface: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *BridgeSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }
