// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The Extend Surface tool (M36-F11) lengthens the running NURBS surface past a chosen edge with a
// tangent (G1) or curvature (G2/G3) continuation, by distance. It is parameter-only (the running
// surface is the input). Untrim is a one-shot command (no parameters) that recovers the surface's
// full domain.

// extendContinuationLabels label the continuation orders for the dialog (index+1 = continuation order).
var extendContinuationLabels = []string{"Tangent (G1)", "Curvature (G2)", "G3"}

// ExtendSurfaceTool lengthens the running NURBS surface past an edge.
type ExtendSurfaceTool struct {
	dialogTool
	edge         int // index into matchEdgeOptions
	continuation int // index into extendContinuationLabels (order = index+1)
	distance     float64
	added        *feature.PartFeature
}

// NewExtendSurfaceTool returns an extend tool defaulting to a curvature (G2) extension of the U-max
// edge by 1 unit.
func NewExtendSurfaceTool() *ExtendSurfaceTool {
	return &ExtendSurfaceTool{edge: 1, continuation: 1, distance: 1}
}

// Name implements [Tool].
func (t *ExtendSurfaceTool) Name() string { return "Extend Surface" }

// Prompt guides the input.
func (t *ExtendSurfaceTool) Prompt(*Session) string {
	return "Extend the running surface: pick the edge, continuation and distance, then OK."
}

// Params exposes the edge, continuation and distance for the generic dialog.
func (t *ExtendSurfaceTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Distance", Get: func() float64 { return t.distance }, Set: func(v float64) { t.distance = v }}},
		Choices: []ChoiceParam{
			{Label: "Edge", Options: matchEdgeLabels, Get: func() int { return t.edge }, Set: func(v int) { t.edge = v }},
			{Label: "Continuation", Options: extendContinuationLabels, Get: func() int { return t.continuation }, Set: func(v int) { t.continuation = v }},
		},
	}
}

// CanCommit reports whether the extension distance is positive.
func (t *ExtendSurfaceTool) CanCommit() bool { return t.distance > 0 }

// addExtendSurface builds the extend feature into fs — the shared constructor used by both
// Commit (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *ExtendSurfaceTool) addExtendSurface(fs *feature.PartFeatures) *feature.PartFeature {
	return feature.NewExtendSurfaceFeatures(fs).Add(matchEdgeOptions[t.edge], t.distance, t.continuation+1)
}

// DraftFeature implements [PartFeatureTool] (#1626): the extension it would commit, built into
// a scratch engine so the commit gate and preview can evaluate it without touching the part.
func (t *ExtendSurfaceTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addExtendSurface(fs), nil
	})
}

// Commit extends the running surface and recomputes.
func (t *ExtendSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addExtendSurface(part.Features())
	part.Recompute()
	s.recordEdit(part, "Extend Surface")
	if !t.added.Health().OK() {
		return errors.New("extend surface: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ExtendSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }

// untrimSurface recovers the running NURBS surface's full domain (the Untrim command).
func untrimSurface(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	pf := feature.NewUntrimFeatures(part.Features()).Add()
	part.Recompute()
	s.recordEdit(part, "Untrim Surface")
	if !pf.Health().OK() {
		return errors.New("untrim surface: " + pf.Health().Reason)
	}
	s.SetNotice("Untrimmed surface — recovered the full base surface")
	return nil
}
