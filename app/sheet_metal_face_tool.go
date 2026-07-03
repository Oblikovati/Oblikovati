// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// SheetMetalFaceTool is the interactive Face command (M13-F02): activate it on a sheet-metal
// part, click a closed sketch profile, and OK to thicken it into a wall at the rule's gauge.
// The first Face is the base wall (a new body); a later Face joins the running sheet.
type SheetMetalFaceTool struct {
	profiles []ProfileHandle
	added    *feature.PartFeature
}

// NewSheetMetalFaceTool returns a Face tool awaiting a profile pick.
func NewSheetMetalFaceTool() *SheetMetalFaceTool { return &SheetMetalFaceTool{} }

// Name implements [Tool].
func (t *SheetMetalFaceTool) Name() string { return "Sheet Metal Face" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *SheetMetalFaceTool) Start(*Session) {}

// AcceptedKinds declares the sheet-metal face picks closed sketch regions (profiles).
func (t *SheetMetalFaceTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectProfile} }

// Picks reports the picked regions for the unified highlight.
func (t *SheetMetalFaceTool) Picks() []Selectable { return selectables(t.profiles) }

// Pick captures the clicked profile (a single region; a re-pick replaces it).
func (t *SheetMetalFaceTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profiles = []ProfileHandle{p}
	}
}

// CanCommit reports whether a profile has been picked.
func (t *SheetMetalFaceTool) CanCommit() bool { return len(t.profiles) > 0 }

// Commit thickens the picked profile into a wall at the rule gauge and recomputes. A first
// Face starts a new body; a later one joins the running sheet.
func (t *SheetMetalFaceTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if len(t.profiles) == 0 {
		return errors.New("sheet-metal face: pick a closed sketch profile")
	}
	t.added = t.addFace(part, part.Features())
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Face")
}

// addFace builds the sheet-metal face feature into engine fs — shared by Commit and the
// preview. The operation (new body vs join the running sheet) is decided from the PART's
// current result, not fs, so the preview matches whether a sheet already exists.
func (t *SheetMetalFaceTool) addFace(part *compdef.PartComponentDefinition, fs *feature.PartFeatures) *feature.PartFeature {
	op := ops.NewBody
	if len(part.Features().Result()) > 0 {
		op = ops.Join
	}
	p := t.profiles[0]
	return feature.NewSheetMetalFaceFeatures(fs).Add(&feature.SheetMetalFaceDefinition{
		Sketch: p.Sketch, ProfileIndex: p.ProfileIndex, Operation: op,
	})
}

// Cancel abandons the tool.
func (t *SheetMetalFaceTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SheetMetalFaceTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached sheet-metal face feature the viewport previews.
func (t *SheetMetalFaceTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addFace(part, fs), nil
	})
}
