// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
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

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *PatchTool) Start(*Session) {}

// AcceptedKinds declares patch picks a closed sketch region (profile).
func (t *PatchTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectProfile} }

// Picks reports the picked region for the unified highlight.
func (t *PatchTool) Picks() []Selectable { return singlePick(t.profile) }

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
	t.added = t.addPatch(part.Features())
	part.Recompute()
	s.recordEdit(part, "Patch")
	if !t.added.Health().OK() {
		return errors.New("patch: " + t.added.Health().Reason)
	}
	return nil
}

// addPatch builds the boundary-patch feature into engine fs — shared by Commit and the preview.
func (t *PatchTool) addPatch(fs *feature.PartFeatures) *feature.PartFeature {
	return feature.NewBoundaryPatchFeatures(fs).Add(t.profile.Sketch, t.profile.ProfileIndex, feature.PatchFree)
}

// DraftFeature returns the unattached patch feature the viewport previews before commit.
func (t *PatchTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addPatch(fs), nil
	})
}

// Cancel restores the default selection filter.
func (t *PatchTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *PatchTool) AddedFeature() *feature.PartFeature { return t.added }
