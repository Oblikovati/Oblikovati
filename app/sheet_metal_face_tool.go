// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
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

// Start filters selection to closed sketch profiles.
func (t *SheetMetalFaceTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile))
}

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
	op := ops.NewBody
	if len(part.Features().Result()) > 0 {
		op = ops.Join
	}
	p := t.profiles[0]
	t.added = feature.NewSheetMetalFaceFeatures(part.Features()).Add(&feature.SheetMetalFaceDefinition{
		Sketch: p.Sketch, ProfileIndex: p.ProfileIndex, Operation: op,
	})
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Face")
}

// Cancel abandons the tool.
func (t *SheetMetalFaceTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SheetMetalFaceTool) AddedFeature() *feature.PartFeature { return t.added }
