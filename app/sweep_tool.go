// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// SweepTool is the interactive Sweep command: activate it, click a sketch region (the
// profile) and a sketch path (the rail, on another plane), set an optional twist, and
// OK to sweep the profile along the path into a solid. The path's sketch plane maps its
// 2D chain to the 3D rail the model sweep consumes.
type SweepTool struct {
	profile   *ProfileHandle
	path      *PathHandle
	twist     float64
	operation ops.PartFeatureOperation
	added     *feature.PartFeature
}

// NewSweepTool returns a sweep tool that creates a new body.
func NewSweepTool() *SweepTool { return &SweepTool{operation: ops.NewBody} }

// Name implements [Tool].
func (t *SweepTool) Name() string { return "Sweep" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *SweepTool) Start(*Session) {}

// AcceptedKinds declares sweep picks a closed region (profile) and a path to sweep it along.
func (t *SweepTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectProfile, SelectPath}
}

// Picks reports the picked profile and path for the unified highlight.
func (t *SweepTool) Picks() []Selectable {
	var picks []Selectable
	if t.profile != nil {
		picks = append(picks, *t.profile)
	}
	if t.path != nil {
		picks = append(picks, *t.path)
	}
	return picks
}

// Pick routes a profile pick to the profile slot and a path pick to the path slot.
func (t *SweepTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case ProfileHandle:
		pc := h
		t.profile = &pc
	case PathHandle:
		pc := h
		t.path = &pc
	}
}

// SetTwist/Twist set the total twist (radians) spread along the path; SetOperation
// chooses the boolean.
func (t *SweepTool) SetTwist(radians float64)                 { t.twist = radians }
func (t *SweepTool) Twist() float64                           { return t.twist }
func (t *SweepTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *SweepTool) Operation() ops.PartFeatureOperation      { return t.operation }

// PickedProfile / PickedPath report what has been gathered (for the UI/tests).
func (t *SweepTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

func (t *SweepTool) PickedPath() (PathHandle, bool) {
	if t.path == nil {
		return PathHandle{}, false
	}
	return *t.path, true
}

// CanCommit reports whether both a profile and a path have been picked.
func (t *SweepTool) CanCommit() bool { return t.profile != nil && t.path != nil }

// ClearProfile / ClearPath empty one pick each — the property panel's selector clear
// (⊗) affordances on the Profiles and Path chips.
func (t *SweepTool) ClearProfile() { t.profile = nil }
func (t *SweepTool) ClearPath()    { t.path = nil }

// SourceSketchName returns the sketch the picked profile comes from, for the property
// panel's breadcrumb; "" until a profile is picked.
func (t *SweepTool) SourceSketchName() string {
	if t.profile == nil {
		return ""
	}
	return t.profile.Sketch.Name()
}

// Commit resolves the path to a 3D rail, adds the sweep feature, and recomputes; a sick
// feature keeps the tool open by returning an error.
func (t *SweepTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.added, err = t.addSweep(part.Features()); err != nil {
		return err
	}
	part.Recompute()
	s.recordEdit(part, "Sweep")
	if !t.added.Health().OK() {
		return errors.New("sweep: " + t.added.Health().Reason)
	}
	return nil
}

// resolveSweepPath turns a picked sketch path into a model-space 3D rail by mapping its
// 2D chain through the path's sketch plane.
func resolveSweepPath(h *PathHandle) (*sketch.Path3D, error) {
	paths := h.Sketch.Paths()
	if h.PathIndex < 0 || h.PathIndex >= len(paths) {
		return nil, fmt.Errorf("sweep: path %d not found (sketch has %d)", h.PathIndex, len(paths))
	}
	p := paths[h.PathIndex]
	plane := h.Sketch.Plane()
	pts := p.Points()
	chain := make([]*sketch.Point3D, len(pts))
	for i, q := range pts {
		chain[i] = sketch.NewPoint3D(plane.ToModel(q))
	}
	return sketch.NewPath3D(chain, p.IsClosed()), nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SweepTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the sweep steps.
func (t *SweepTool) Prompt(*Session) string {
	switch {
	case t.profile == nil:
		return "Select a region to sweep"
	case t.path == nil:
		return "Select a path to sweep along"
	default:
		return "Set the options and click OK"
	}
}

// Preview outlines the picked profile region until the sweep is committed.
// addSweep resolves the path to a 3D rail and builds the sweep feature into engine fs — the
// shared constructor used by both Commit (the part's engine) and DraftFeature (a scratch
// engine), so the preview matches the committed result.
func (t *SweepTool) addSweep(fs *feature.PartFeatures) (*feature.PartFeature, error) {
	path3d, err := resolveSweepPath(t.path)
	if err != nil {
		return nil, err
	}
	twist := t.twist
	return feature.NewSweepFeatures(fs).
		Add(t.profile.Sketch, t.profile.ProfileIndex, path3d, func() float64 { return twist }, t.operation), nil
}

// DraftFeature returns the unattached sweep feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addSweep the commit uses. Empty until both
// a profile and a path are picked.
func (t *SweepTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(t.addSweep)
}

// Cancel restores the default selection filter.
func (t *SweepTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
