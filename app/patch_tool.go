// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/model/feature"
)

// PatchTool is the interactive Patch command (Surface panel): click a closed sketch region to
// fill it with a surface (a boundary patch). It auto-commits on the pick — one click makes the
// patch — matching Inventor's lightweight surfacing flow.
type PatchTool struct {
	profile *ProfileHandle
	added   *feature.PartFeature
}

// NewPatchTool returns a patch tool.
func NewPatchTool() *PatchTool { return &PatchTool{} }

// Name implements [Tool].
func (t *PatchTool) Name() string { return "Patch" }

// Start filters selection to closed regions.
func (t *PatchTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures the clicked closed region.
func (t *PatchTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		pc := p
		t.profile = &pc
	}
}

// AutoCommitOnPick finishes the patch the moment a region is clicked.
func (t *PatchTool) AutoCommitOnPick() bool { return true }

// PickedProfile returns the clicked region (if any) for the unified tool highlight.
func (t *PatchTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

// CanCommit reports whether a region has been picked.
func (t *PatchTool) CanCommit() bool { return t.profile != nil }

// Commit fills the region with a boundary-patch surface and recomputes.
func (t *PatchTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewBoundaryPatchFeatures(part.Features()).
		Add(t.profile.Sketch, t.profile.ProfileIndex, feature.PatchFree)
	part.Recompute()
	s.recordEdit(part, "Patch")
	if !t.added.Health().OK() {
		return errors.New("patch: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel restores the default selection filter.
func (t *PatchTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *PatchTool) AddedFeature() *feature.PartFeature { return t.added }
