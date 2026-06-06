// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/model/feature"
)

// ExtendTool is the interactive Extend command (Surface panel): click a surface's boundary edge
// and set a distance to grow the face outward along that edge. It exposes the distance through
// ParameterizedTool for the generic dialog.
type ExtendTool struct {
	edge     *EdgeHandle
	distance float64
	added    *feature.PartFeature
}

// NewExtendTool returns an extend tool defaulting to a 1-unit extension.
func NewExtendTool() *ExtendTool { return &ExtendTool{distance: 1} }

// Name implements [Tool].
func (t *ExtendTool) Name() string { return "Extend" }

// Start filters selection to edges so clicks pick a boundary edge.
func (t *ExtendTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectEdge)) }

// Pick captures the boundary edge to extend.
func (t *ExtendTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok {
		ec := e
		t.edge = &ec
	}
}

// Edges returns the picked boundary edge (one or none) for the unified tool highlight.
func (t *ExtendTool) Edges() []EdgeHandle {
	if t.edge == nil {
		return nil
	}
	return []EdgeHandle{*t.edge}
}

// SetDistance/Distance drive how far the edge extends.
func (t *ExtendTool) SetDistance(d float64) { t.distance = d }
func (t *ExtendTool) Distance() float64     { return t.distance }

// Params exposes the extension distance for the generic property dialog.
func (t *ExtendTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{Label: "Distance", Get: t.Distance, Set: t.SetDistance}}}
}

// CanCommit reports whether an edge is picked and the distance is positive.
func (t *ExtendTool) CanCommit() bool { return t.edge != nil && t.distance > 0 }

// Commit extends the surface along the picked edge and recomputes.
func (t *ExtendTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d := t.distance
	t.added = feature.NewExtendFeatures(part.Features()).Add(t.edge.Edge.ReferenceKey(), func() float64 { return d })
	part.Recompute()
	s.recordEdit(part, "Extend")
	if !t.added.Health().OK() {
		return errors.New("extend: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel restores the default selection filter.
func (t *ExtendTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ExtendTool) AddedFeature() *feature.PartFeature { return t.added }
