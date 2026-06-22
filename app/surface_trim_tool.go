// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
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

// Start adopts a pre-selected work plane; the engine installs the filter from AcceptedKinds.
func (t *SurfaceTrimTool) Start(s *Session) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		t.plane = wp
	}
}

// AcceptedKinds declares surface-trim picks a work plane (the cutting plane).
func (t *SurfaceTrimTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectWorkPlane} }

// Picks reports the picked cutting plane for the unified highlight.
func (t *SurfaceTrimTool) Picks() []Selectable {
	if t.plane == nil {
		return nil
	}
	return []Selectable{WorkPlaneHandle{Plane: t.plane}}
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
	t.added = t.addTrim(part.Features())
	part.Recompute()
	s.recordEdit(part, "Trim")
	if !t.added.Health().OK() {
		return errors.New("trim: " + t.added.Health().Reason)
	}
	return nil
}

// addTrim builds the surface-trim feature into engine fs — shared by Commit and the preview.
func (t *SurfaceTrimTool) addTrim(fs *feature.PartFeatures) *feature.PartFeature {
	sp := t.plane.Plane()
	return feature.NewTrimFeatures(fs).AddByPlane(sp.Origin(), sp.Normal().AsVector(), t.keepPositive)
}

// DraftFeature returns the unattached trim feature the viewport previews before commit.
func (t *SurfaceTrimTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addTrim(fs), nil
	})
}

// Cancel restores the default selection filter.
func (t *SurfaceTrimTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SurfaceTrimTool) AddedFeature() *feature.PartFeature { return t.added }
