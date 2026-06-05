// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// SculptTool is the interactive Sculpt command (Surface panel): fill the volume bounded by the
// running surface bodies into a solid. It takes no pick (it acts on all bounding surfaces); the
// dialog drives the closing tolerance. The bounding surfaces must enclose a closed cell, else the
// feature goes sick.
type SculptTool struct {
	tolerance float64
	added     *feature.PartFeature
}

// NewSculptTool returns a sculpt tool defaulting to an exact (zero-tolerance) close.
func NewSculptTool() *SculptTool { return &SculptTool{} }

// Name implements [Tool].
func (t *SculptTool) Name() string { return "Sculpt" }

// Start has nothing to select — sculpt acts on every bounding surface.
func (t *SculptTool) Start(*Session) {}

// Pick is unused.
func (t *SculptTool) Pick(*Session, Selectable) {}

// SetTolerance/Tolerance drive the coincidence tolerance for closing the bounding surfaces.
func (t *SculptTool) SetTolerance(v float64) { t.tolerance = v }
func (t *SculptTool) Tolerance() float64     { return t.tolerance }

// Params exposes the closing tolerance for the generic property dialog.
func (t *SculptTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{Label: "Tolerance", Get: t.Tolerance, Set: t.SetTolerance}}}
}

// CanCommit is always true — sculpt acts on whatever bounding surfaces are present (Commit
// validates that they enclose a volume).
func (t *SculptTool) CanCommit() bool { return true }

// Commit fills the bounded volume into a solid and recomputes; open surfaces keep the tool open.
func (t *SculptTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewSculptFeatures(part.Features()).Add(ops.NewBody, t.tolerance)
	part.Recompute()
	s.recordEdit(part, "Sculpt")
	if !t.added.Health().OK() {
		return errors.New("sculpt: " + t.added.Health().Reason)
	}
	return nil
}

// Cancel abandons the tool with no change.
func (t *SculptTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SculptTool) AddedFeature() *feature.PartFeature { return t.added }
