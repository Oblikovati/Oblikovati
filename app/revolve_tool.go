// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/renderer"
)

// RevolveTool is the interactive Revolve command: activate it, click a sketch region,
// choose the axis of revolution and the swept angle in the property window, and OK to
// add a revolve feature to the active part. It mirrors [ExtrudeTool] and is driven
// entirely by session input so a test exercises the full flow with synthetic clicks.
type RevolveTool struct {
	profile   *ProfileHandle
	axis      feature.WorkRef // origin axis (X/Y/Z) or a user work axis
	angle     float64         // swept angle in radians; 0 ⇒ full revolution
	operation ops.PartFeatureOperation
	added     *feature.PartFeature
}

// NewRevolveTool returns a revolve tool defaulting to a full revolution about the Y
// origin axis that creates a new body.
func NewRevolveTool() *RevolveTool {
	return &RevolveTool{axis: feature.OriginYAxis, operation: ops.NewBody}
}

// Name implements [Tool].
func (t *RevolveTool) Name() string { return "Revolve" }

// Start sets the selection filter to profiles so clicks pick a region.
func (t *RevolveTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures the region the user clicked.
func (t *RevolveTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		pc := p
		t.profile = &pc
	}
}

// The options the property window drives: the revolution axis, the swept angle, and the
// boolean operation.
func (t *RevolveTool) SetAxis(ref feature.WorkRef)              { t.axis = ref }
func (t *RevolveTool) Axis() feature.WorkRef                    { return t.axis }
func (t *RevolveTool) SetAngle(radians float64)                 { t.angle = radians }
func (t *RevolveTool) Angle() float64                           { return t.angle }
func (t *RevolveTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *RevolveTool) Operation() ops.PartFeatureOperation      { return t.operation }

// SetFullRevolution sets the angle to a full turn (0, the model's "full" sentinel).
func (t *RevolveTool) SetFullRevolution() { t.angle = 0 }

// IsFullRevolution reports whether the tool will sweep a full turn.
func (t *RevolveTool) IsFullRevolution() bool { return t.angle <= 0 }

// PickedProfile returns the picked region (and true), or false when none picked yet.
func (t *RevolveTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

// CanCommit reports whether a region has been picked (the axis has a default).
func (t *RevolveTool) CanCommit() bool { return t.profile != nil }

// Commit adds the revolve feature to the active part and recomputes; a sick feature
// (open profile, missing axis) keeps the tool open by returning an error.
func (t *RevolveTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	axis, ok := part.WorkGeometry().AxisByRef(t.axis)
	if !ok {
		return errors.New("revolve: axis " + string(t.axis) + " not found")
	}
	angle := t.angle
	t.added = feature.NewRevolveFeatures(part.Features()).
		Add(t.profile.Sketch, t.profile.ProfileIndex, axis, func() float64 { return angle }, t.operation)
	part.Recompute()
	if !t.added.Health().OK() {
		return errors.New("revolve: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *RevolveTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the revolve steps (Inventor's status-bar prompts).
func (t *RevolveTool) Prompt(*Session) string {
	if t.profile == nil {
		return "Select a region to revolve"
	}
	return "Set the axis and angle, then click OK"
}

// Preview returns a transient outline of the region to revolve, so the viewport shows
// what will be swept before OK. Empty until a region is picked.
func (t *RevolveTool) Preview(*Session) []renderer.DrawItem {
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
func (t *RevolveTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// FullTurn is the swept angle of a complete revolution, for the property window's
// "Full" button to set an explicit angle when the user switches to a partial angle.
const FullTurn = 2 * stdmath.Pi
