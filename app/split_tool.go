// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// SplitTool is the interactive Split command: pick a work plane to cut with, choose whether to
// split the part into two bodies (the default) or trim away one side, then OK to divide the
// active part. The cutting plane may be pre-selected in the browser/3D view before invoking.
type SplitTool struct {
	plane     *feature.WorkPlane
	keep      feature.SplitSide
	facesOnly bool // Split Faces: imprint the plane onto the faces, removing nothing (#330)
	added     *feature.PartFeature
}

// NewSplitTool returns a split tool defaulting to a full split into two bodies.
func NewSplitTool() *SplitTool { return &SplitTool{keep: feature.SplitBoth} }

// Name implements [Tool].
func (t *SplitTool) Name() string { return "Split" }

// Start adopts a pre-selected work plane; the engine installs the filter from AcceptedKinds.
func (t *SplitTool) Start(s *Session) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		t.plane = wp
	}
}

// AcceptedKinds declares split picks a work plane (the cutting plane).
func (t *SplitTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectWorkPlane} }

// Picks reports the picked cutting plane for the unified highlight.
func (t *SplitTool) Picks() []Selectable {
	if t.plane == nil {
		return nil
	}
	return []Selectable{WorkPlaneHandle{Plane: t.plane}}
}

// Pick captures the work plane the user clicked.
func (t *SplitTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(WorkPlaneHandle); ok {
		t.plane = h.Plane
	}
}

// SetKeep/Keep drive which side(s) the split keeps (both = split, one side = trim).
// Choosing a keep side leaves faces-only mode.
func (t *SplitTool) SetKeep(k feature.SplitSide) { t.keep, t.facesOnly = k, false }
func (t *SplitTool) Keep() feature.SplitSide     { return t.keep }

// Keep convenience setters + label, so the head can drive the choice without importing the
// model's enum. Each keep mode clears faces-only (the modes are exclusive).
func (t *SplitTool) SetKeepBoth()     { t.keep, t.facesOnly = feature.SplitBoth, false }
func (t *SplitTool) SetKeepPositive() { t.keep, t.facesOnly = feature.SplitPositive, false }
func (t *SplitTool) SetKeepNegative() { t.keep, t.facesOnly = feature.SplitNegative, false }

// SetSplitFaces/FacesOnly drive the faces-only imprint mode: the plane splits the faces it
// crosses but removes no material (#330).
func (t *SplitTool) SetSplitFaces()  { t.facesOnly = true }
func (t *SplitTool) FacesOnly() bool { return t.facesOnly }

func (t *SplitTool) KeepLabel() string {
	switch {
	case t.facesOnly:
		return "Split faces (imprint only)"
	case t.keep == feature.SplitPositive:
		return "Trim (keep front side)"
	case t.keep == feature.SplitNegative:
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
	t.added = t.addSplit(part.Features())
	part.Recompute()
	s.recordEdit(part, "Split")
	if !t.added.Health().OK() {
		return errors.New("split: " + t.added.Health().Reason)
	}
	return nil
}

// addSplit builds the split feature into engine fs — shared by Commit and the preview.
func (t *SplitTool) addSplit(fs *feature.PartFeatures) *feature.PartFeature {
	mods := feature.NewModifyFeatures(fs)
	if t.facesOnly {
		return mods.AddSplitFaces(t.plane)
	}
	return mods.AddSplitSolid(t.plane, t.keep)
}

// DraftFeature returns the unattached split feature the viewport previews before commit.
func (t *SplitTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addSplit(fs), nil
	})
}

// Cancel restores the default selection filter.
func (t *SplitTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SplitTool) AddedFeature() *feature.PartFeature { return t.added }

// ClearPlane empties the picked cutting plane — the property panel's selector clear
// (⊗) — returning the tool to its pick-a-plane step.
func (t *SplitTool) ClearPlane() { t.plane = nil }
