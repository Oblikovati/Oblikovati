// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
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

// Start accepts both profile and path picks.
func (t *SweepTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile, SelectPath))
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

// Commit resolves the path to a 3D rail, adds the sweep feature, and recomputes; a sick
// feature keeps the tool open by returning an error.
func (t *SweepTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	path3d, err := resolveSweepPath(t.path)
	if err != nil {
		return err
	}
	twist := t.twist
	t.added = feature.NewSweepFeatures(part.Features()).
		Add(t.profile.Sketch, t.profile.ProfileIndex, path3d, func() float64 { return twist }, t.operation)
	part.Recompute()
	s.recordEdit(part, "Sweep")
	if !t.added.Health().OK() {
		return errors.New("sweep: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
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
func (t *SweepTool) Preview(*Session) []renderer.DrawItem {
	if t.profile == nil || t.profile.ProfileIndex >= t.profile.Sketch.Profiles().Count() {
		return nil
	}
	poly := t.profile.Sketch.Profiles().Item(t.profile.ProfileIndex).OuterLoop().Polygon()
	plane := t.profile.Sketch.Plane()
	pts := make([]math.Point3, len(poly))
	idx := make([]int, 0, 2*len(poly))
	for i, p := range poly {
		pts[i] = plane.ToModel(p)
		idx = append(idx, i, (i+1)%len(poly))
	}
	return []renderer.DrawItem{{Primitive: renderer.Lines, Positions: pts, Indices: idx, Color: [4]float32{1, 0.6, 0, 1}}}
}

// Cancel restores the default selection filter.
func (t *SweepTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
