// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/model/feature"
)

// AngleWorkPlaneTool is the "plane at an angle to a plane, about an axis" interaction: pick the
// axis to rotate about and the plane to measure from, then type the angle and OK. Like the
// offset tool — and unlike the no-value datum tools — it does NOT auto-commit on the picks: an
// angle plane with no angle is just the base plane, so the value is gathered first.
//
// The standard "plane at 30° to this face through that edge" was the most conspicuous of the
// eleven work-plane constructors with no ribbon path (#2044). The angle is held in radians, the
// unit the definition stores; the dialog edits degrees.
type AngleWorkPlaneTool struct {
	axisRef feature.WorkRef
	baseRef feature.WorkRef
	hasAxis bool
	hasBase bool
	angle   float64 // radians; 0 until the user sets it, so OK stays disabled
	added   *feature.WorkPlane
}

// NewAngleWorkPlaneTool returns an angle-plane tool awaiting an axis, a plane and an angle.
func NewAngleWorkPlaneTool() *AngleWorkPlaneTool { return &AngleWorkPlaneTool{} }

// Name implements [Tool].
func (t *AngleWorkPlaneTool) Name() string { return "Angle to Plane" }

// Start seeds the picks from the current selection, so a user who selected the axis and plane
// first only needs to type the angle.
func (t *AngleWorkPlaneTool) Start(s *Session) {
	if axes := s.SelectedWorkAxes(); len(axes) > 0 {
		t.axisRef, t.hasAxis = axes[0].Key(), true
	}
	if wp := s.SelectedWorkPlane(); wp != nil {
		t.baseRef, t.hasBase = wp.Key(), true
	}
	s.Selection().Clear()
}

// AcceptedKinds declares the axis to rotate about and the plane (or planar face) to measure from.
func (t *AngleWorkPlaneTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectWorkAxis, SelectWorkPlane, SelectFace}
}

// Pick records the axis, or the plane/planar face to measure the angle from.
func (t *AngleWorkPlaneTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case WorkAxisHandle:
		t.axisRef, t.hasAxis = h.Axis.Key(), true
	case WorkPlaneHandle:
		t.baseRef, t.hasBase = h.Plane.Key(), true
	case FaceHandle:
		t.baseRef, t.hasBase = feature.FaceRef(h.Face.ReferenceKey()), true
	}
}

// SetAngleDegrees / AngleDegrees hold the angle in degrees, which is how the panel edits it;
// the definition stores radians.
func (t *AngleWorkPlaneTool) SetAngleDegrees(deg float64) { t.angle = deg * stdmath.Pi / 180 }
func (t *AngleWorkPlaneTool) AngleDegrees() float64       { return t.angle * 180 / stdmath.Pi }

// AxisPicked / BasePicked report which inputs are gathered, so the dialog prompts for what is
// still missing.
func (t *AngleWorkPlaneTool) AxisPicked() bool { return t.hasAxis }
func (t *AngleWorkPlaneTool) BasePicked() bool { return t.hasBase }

// CanCommit requires both picks and a non-zero angle — a zero angle would duplicate the base
// plane, so OK stays disabled rather than creating a redundant datum.
func (t *AngleWorkPlaneTool) CanCommit() bool {
	return t.hasAxis && t.hasBase && t.angle != 0
}

// Commit creates the angled work plane and recomputes.
func (t *AngleWorkPlaneTool) Commit(s *Session) error {
	if !t.CanCommit() {
		return errors.New("angle plane: need an axis, a plane and a non-zero angle")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	a, axis, base := t.angle, t.axisRef, t.baseRef
	t.added = finishWorkPlane(part, part.WorkPlanes().AddByLinePlaneAndAngle(axis, base, func() float64 { return a }))
	s.recordEdit(part, labelWorkPlane)
	return nil
}

// Cancel is a no-op; the engine restores the ambient filter.
func (t *AngleWorkPlaneTool) Cancel(*Session) {}

// Prompt guides the user through the three steps.
func (t *AngleWorkPlaneTool) Prompt(*Session) string {
	switch {
	case !t.hasAxis:
		return "Select the axis or edge to rotate about"
	case !t.hasBase:
		return "Select the plane or planar face to measure the angle from"
	default:
		return "Enter the angle, then click OK"
	}
}

// AddedPlane returns the plane created on commit (for inspection/tests).
func (t *AngleWorkPlaneTool) AddedPlane() *feature.WorkPlane { return t.added }

// ClearPicks drops the picked axis and plane — the property panel's selector clear (⊗).
func (t *AngleWorkPlaneTool) ClearPicks() { t.hasAxis, t.hasBase = false, false }
