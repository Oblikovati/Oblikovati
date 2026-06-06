// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/kernel/ops"
	"oblikovati/math"
	"oblikovati/model/feature"
	"oblikovati/renderer"
)

// CoilTool is the interactive Coil command: activate it, click a sketch region, choose
// the helix axis, pitch, number of revolutions and operation in the property window,
// and OK to add a coil feature to the active part. It mirrors [RevolveTool].
type CoilTool struct {
	profile     *ProfileHandle
	axis        feature.WorkRef
	pitch       float64
	revolutions float64
	operation   ops.PartFeatureOperation
	added       *feature.PartFeature
}

// NewCoilTool returns a coil tool defaulting to a single-pitch, 3-revolution helix about
// the Y origin axis that creates a new body.
func NewCoilTool() *CoilTool {
	return &CoilTool{axis: feature.OriginYAxis, pitch: 1, revolutions: 3, operation: ops.NewBody}
}

// Name implements [Tool].
func (t *CoilTool) Name() string { return "Coil" }

// Start sets the selection filter to profiles so clicks pick a region.
func (t *CoilTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures the region the user clicked.
func (t *CoilTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		pc := p
		t.profile = &pc
	}
}

// The options the property window drives: the helix axis, pitch, revolutions, operation.
func (t *CoilTool) SetAxis(ref feature.WorkRef)              { t.axis = ref }
func (t *CoilTool) Axis() feature.WorkRef                    { return t.axis }
func (t *CoilTool) SetPitch(p float64)                       { t.pitch = p }
func (t *CoilTool) Pitch() float64                           { return t.pitch }
func (t *CoilTool) SetRevolutions(r float64)                 { t.revolutions = r }
func (t *CoilTool) Revolutions() float64                     { return t.revolutions }
func (t *CoilTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *CoilTool) Operation() ops.PartFeatureOperation      { return t.operation }

// PickedProfile returns the picked region (and true), or false when none picked yet.
func (t *CoilTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

// CanCommit reports whether a region is picked and the revolutions are positive.
func (t *CoilTool) CanCommit() bool { return t.profile != nil && t.revolutions > 0 }

// Commit adds the coil feature to the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *CoilTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	axis, ok := part.WorkGeometry().AxisByRef(t.axis)
	if !ok {
		return errors.New("coil: axis " + string(t.axis) + " not found")
	}
	pitch, revs := t.pitch, t.revolutions
	t.added = feature.NewCoilFeatures(part.Features()).Add(t.profile.Sketch, t.profile.ProfileIndex, axis,
		func() float64 { return pitch }, func() float64 { return revs }, 0, t.operation)
	part.Recompute()
	s.recordEdit(part, "Coil")
	if !t.added.Health().OK() {
		return errors.New("coil: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *CoilTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the coil steps.
func (t *CoilTool) Prompt(*Session) string {
	if t.profile == nil {
		return "Select a region to coil"
	}
	return "Set the axis, pitch and revolutions, then click OK"
}

// Preview returns a transient outline of the region to coil, until a region is picked.
func (t *CoilTool) Preview(*Session) []renderer.DrawItem {
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
func (t *CoilTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
