// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// StitchTool is the interactive Stitch command (Surface panel): weld the running surface bodies
// into one quilt, promoting a closed quilt to a solid unless "keep as surface" is set. It takes
// no pick (it operates on all surfaces); the property dialog drives the tolerance and that flag.
type StitchTool struct {
	tolerance   float64
	keepSurface bool
	added       *feature.PartFeature
}

// NewStitchTool returns a stitch tool defaulting to an exact (zero-tolerance) weld.
func NewStitchTool() *StitchTool { return &StitchTool{} }

// Name implements [Tool].
func (t *StitchTool) Name() string { return "Stitch" }

// Start has nothing to select — stitch welds every running surface body.
func (t *StitchTool) Start(*Session) {}

// Pick is unused.
func (t *StitchTool) Pick(*Session, Selectable) {}

// The options the dialog drives: weld tolerance and whether to keep a closed quilt as a surface.
func (t *StitchTool) SetTolerance(v float64) { t.tolerance = v }
func (t *StitchTool) Tolerance() float64     { return t.tolerance }
func (t *StitchTool) SetKeepSurface(v bool)  { t.keepSurface = v }
func (t *StitchTool) KeepSurface() bool      { return t.keepSurface }

// Params exposes the tolerance and keep-as-surface flag for the generic property dialog.
func (t *StitchTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Tolerance", Get: t.Tolerance, Set: t.SetTolerance}},
		Bools:  []BoolParam{{Label: "Keep as surface", Get: t.KeepSurface, Set: t.SetKeepSurface}},
	}
}

// CanCommit is always true — stitch acts on whatever surfaces are present (Commit validates).
func (t *StitchTool) CanCommit() bool { return true }

// Commit welds the running surface bodies and recomputes; a failed weld keeps the tool open.
func (t *StitchTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewStitchFeatures(part.Features()).Add(t.tolerance, t.keepSurface)
	part.Recompute()
	s.recordEdit(part, "Stitch")
	if !t.added.Health().OK() {
		return errors.New("stitch: " + t.added.Health().Reason)
	}
	return nil
}

// Cancel abandons the tool with no change.
func (t *StitchTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *StitchTool) AddedFeature() *feature.PartFeature { return t.added }
