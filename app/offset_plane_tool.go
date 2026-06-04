// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/model/feature"
)

// OffsetWorkPlaneTool is Inventor's Plane-and-Offset work-plane interaction: pick the
// plane (or planar face) to offset from, then type the offset distance and OK. Unlike the
// no-value datum tools it does NOT auto-commit on the pick — an offset with no distance is
// meaningless, so the tool gathers the value first (the fix for Offset dropping a plane at
// a fixed distance without asking). The distance is held in model (database) units; the
// session bridge converts to/from the document's display unit for the dialog field.
type OffsetWorkPlaneTool struct {
	baseRef  feature.WorkRef // the plane or planar-face reference to offset from
	hasBase  bool
	distance float64 // offset in model units; 0 until the user sets it (so OK stays disabled)
	added    *feature.WorkPlane
	prev     *SelectionFilter
}

// NewOffsetWorkPlaneTool returns an offset-plane tool awaiting a plane pick and distance.
func NewOffsetWorkPlaneTool() *OffsetWorkPlaneTool { return &OffsetWorkPlaneTool{} }

// Name implements [Tool].
func (t *OffsetWorkPlaneTool) Name() string { return "Offset Plane" }

// Start filters selection to work planes and planar faces and seeds the base from a
// pre-selected plane (so a user who picked a plane first only needs to enter the distance).
func (t *OffsetWorkPlaneTool) Start(s *Session) {
	t.prev = s.Selection().Filter()
	if wp := s.SelectedWorkPlane(); wp != nil {
		t.baseRef, t.hasBase = wp.Key(), true
	}
	s.Selection().Clear()
	s.Selection().SetFilter(NewSelectionFilter(SelectWorkPlane, SelectFace))
}

// Pick records the plane or planar face to offset from.
func (t *OffsetWorkPlaneTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case WorkPlaneHandle:
		t.baseRef, t.hasBase = h.Plane.Key(), true
	case FaceHandle:
		t.baseRef, t.hasBase = feature.FaceRef(h.Face.ReferenceKey()), true
	}
}

// SetDistance / Distance hold the offset in model units (set from the dialog field via the
// session bridge, which converts the displayed value from the document's length unit).
func (t *OffsetWorkPlaneTool) SetDistance(d float64) { t.distance = d }
func (t *OffsetWorkPlaneTool) Distance() float64     { return t.distance }

// BasePicked reports whether the plane/face to offset from has been chosen, so the dialog
// knows to prompt for the pick vs. the distance.
func (t *OffsetWorkPlaneTool) BasePicked() bool { return t.hasBase }

// CanCommit requires a base and a non-zero offset (so OK stays disabled until both are
// gathered — the tool never silently creates a plane).
func (t *OffsetWorkPlaneTool) CanCommit() bool { return t.hasBase && t.distance != 0 }

// Commit creates the offset work plane at the entered distance and recomputes.
func (t *OffsetWorkPlaneTool) Commit(s *Session) error {
	s.Selection().SetFilter(t.prev)
	if !t.hasBase {
		return errors.New("offset plane: no base plane or face picked")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d, ref := t.distance, t.baseRef
	t.added = finishWorkPlane(part, part.WorkPlanes().AddByPlaneAndOffset(ref, func() float64 { return d }))
	s.recordEdit(part, "Offset Work Plane")
	return nil
}

// Cancel restores the prior selection filter with no change.
func (t *OffsetWorkPlaneTool) Cancel(s *Session) { s.Selection().SetFilter(t.prev) }

// Prompt guides the user through the two steps (Inventor's status-bar prompts).
func (t *OffsetWorkPlaneTool) Prompt(*Session) string {
	if !t.hasBase {
		return "Select a plane or planar face to offset from"
	}
	return "Enter the offset distance, then click OK"
}

// AddedPlane returns the plane created on commit (for inspection/tests).
func (t *OffsetWorkPlaneTool) AddedPlane() *feature.WorkPlane { return t.added }
