// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/model/feature"
)

// SurfaceTrimTool is the interactive Trim command (Surface panel): pick a work plane to cut the
// running surface body with and keep one side. The cutting plane may be pre-selected before
// invoking. It exposes "keep positive side" through ParameterizedTool for the generic dialog.
type SurfaceTrimTool struct {
	plane        *feature.WorkPlane
	keepPositive bool
	added        *feature.PartFeature
}

// NewSurfaceTrimTool returns a trim tool defaulting to keeping the +normal side.
func NewSurfaceTrimTool() *SurfaceTrimTool { return &SurfaceTrimTool{keepPositive: true} }

// Name implements [Tool].
func (t *SurfaceTrimTool) Name() string { return "Trim" }

// Start adopts a pre-selected work plane and filters selection to work planes.
func (t *SurfaceTrimTool) Start(s *Session) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		t.plane = wp
	}
	s.Selection().SetFilter(NewSelectionFilter(SelectWorkPlane))
}

// Pick captures the cutting work plane.
func (t *SurfaceTrimTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(WorkPlaneHandle); ok {
		t.plane = h.Plane
	}
}

// SetKeepPositive/KeepPositive choose which side of the plane survives.
func (t *SurfaceTrimTool) SetKeepPositive(v bool) { t.keepPositive = v }
func (t *SurfaceTrimTool) KeepPositive() bool     { return t.keepPositive }

// Params exposes the keep-side flag for the generic property dialog.
func (t *SurfaceTrimTool) Params() ToolParams {
	return ToolParams{Bools: []BoolParam{{Label: "Keep positive side", Get: t.KeepPositive, Set: t.SetKeepPositive}}}
}

// PickedPlane returns the cutting plane (and true), or false when none picked yet.
func (t *SurfaceTrimTool) PickedPlane() (*feature.WorkPlane, bool) { return t.plane, t.plane != nil }

// CanCommit reports whether a cutting plane has been picked.
func (t *SurfaceTrimTool) CanCommit() bool { return t.plane != nil }

// Commit trims the running surface body by the plane and recomputes.
func (t *SurfaceTrimTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	sp := t.plane.Plane()
	t.added = feature.NewTrimFeatures(part.Features()).AddByPlane(sp.Origin(), sp.Normal().AsVector(), t.keepPositive)
	part.Recompute()
	s.recordEdit(part, "Trim")
	if !t.added.Health().OK() {
		return errors.New("trim: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel restores the default selection filter.
func (t *SurfaceTrimTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SurfaceTrimTool) AddedFeature() *feature.PartFeature { return t.added }
