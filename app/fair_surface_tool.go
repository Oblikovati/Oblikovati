// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The Fair Surface tool (M36-F04) smooths curvature wrinkles out of the running surface while holding
// its boundary continuity (G0/G1/G2) — the Class-A "take the wrinkles out" move. Parameter-only (the
// running surface is the input): the hold-continuity, relaxation strength and iteration count.

// fairHoldLabels label the boundary continuity to hold (index = order).
var fairHoldLabels = []string{"Position (G0)", "Tangent (G1)", "Curvature (G2)"}

// FairSurfaceTool fairs the running surface body.
type FairSurfaceTool struct {
	dialogTool
	hold       int // index into fairHoldLabels (order = index)
	strength   float64
	iterations int
	added      *feature.PartFeature
}

// NewFairSurfaceTool returns a fair tool holding G2, strength 0.5, 20 iterations.
func NewFairSurfaceTool() *FairSurfaceTool {
	return &FairSurfaceTool{hold: 2, strength: 0.5, iterations: 20}
}

// Name implements [Tool].
func (t *FairSurfaceTool) Name() string { return "Fair Surface" }

// Prompt guides the input.
func (t *FairSurfaceTool) Prompt(*Session) string {
	return "Fair the running surface: set the held continuity, strength and iterations, then OK."
}

// Params exposes the hold-continuity, strength and iterations.
func (t *FairSurfaceTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Strength", Get: func() float64 { return t.strength }, Set: func(v float64) { t.strength = v }}},
		Ints:   []IntParam{{Label: "Iterations", Get: func() int { return t.iterations }, Set: func(v int) { t.iterations = v }}},
		Choices: []ChoiceParam{
			{Label: "Hold", Options: fairHoldLabels, Get: func() int { return t.hold }, Set: func(v int) { t.hold = v }},
		},
	}
}

// CanCommit reports whether the strength and iterations are positive.
func (t *FairSurfaceTool) CanCommit() bool { return t.strength > 0 && t.iterations > 0 }

// Commit fairs the running surface and recomputes.
func (t *FairSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewFairFeatures(part.Features()).Add(t.hold, t.strength, t.iterations)
	part.Recompute()
	s.recordEdit(part, "Fair Surface")
	if !t.added.Health().OK() {
		return errors.New("fair surface: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FairSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }
