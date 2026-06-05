// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/model/feature"
)

// SplitTool is the interactive Split command: pick a work plane to cut with, choose whether to
// split the part into two bodies (the default) or trim away one side, then OK to divide the
// active part. The cutting plane may be pre-selected in the browser/3D view before invoking.
type SplitTool struct {
	plane *feature.WorkPlane
	keep  feature.SplitSide
	added *feature.PartFeature
}

// NewSplitTool returns a split tool defaulting to a full split into two bodies.
func NewSplitTool() *SplitTool { return &SplitTool{keep: feature.SplitBoth} }

// Name implements [Tool].
func (t *SplitTool) Name() string { return "Split" }

// Start adopts a pre-selected work plane and filters selection to work planes.
func (t *SplitTool) Start(s *Session) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		t.plane = wp
	}
	s.Selection().SetFilter(NewSelectionFilter(SelectWorkPlane))
}

// Pick captures the work plane the user clicked.
func (t *SplitTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(WorkPlaneHandle); ok {
		t.plane = h.Plane
	}
}

// SetKeep/Keep drive which side(s) the split keeps (both = split, one side = trim).
func (t *SplitTool) SetKeep(k feature.SplitSide) { t.keep = k }
func (t *SplitTool) Keep() feature.SplitSide     { return t.keep }

// Keep convenience setters + label, so the head can drive the choice without importing the
// model's enum.
func (t *SplitTool) SetKeepBoth()     { t.keep = feature.SplitBoth }
func (t *SplitTool) SetKeepPositive() { t.keep = feature.SplitPositive }
func (t *SplitTool) SetKeepNegative() { t.keep = feature.SplitNegative }
func (t *SplitTool) KeepLabel() string {
	switch t.keep {
	case feature.SplitPositive:
		return "Trim (keep front side)"
	case feature.SplitNegative:
		return "Trim (keep back side)"
	default:
		return "Split into two bodies"
	}
}

// PickedPlane returns the cutting plane (and true), or false when none picked yet.
func (t *SplitTool) PickedPlane() (*feature.WorkPlane, bool) { return t.plane, t.plane != nil }

// CanCommit reports whether a cutting plane has been picked.
func (t *SplitTool) CanCommit() bool { return t.plane != nil }

// Commit splits the active part by the plane and recomputes; a sick feature keeps the tool open.
func (t *SplitTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddSplitSolid(t.plane, t.keep)
	part.Recompute()
	s.recordEdit(part, "Split")
	if !t.added.Health().OK() {
		return errors.New("split: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel restores the default selection filter.
func (t *SplitTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SplitTool) AddedFeature() *feature.PartFeature { return t.added }
