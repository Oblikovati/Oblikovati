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
	base     *feature.WorkPlane
	distance float64 // offset in model units; 0 until the user sets it (so OK stays disabled)
	added    *feature.WorkPlane
	prev     *SelectionFilter
}

// NewOffsetWorkPlaneTool returns an offset-plane tool awaiting a plane pick and distance.
func NewOffsetWorkPlaneTool() *OffsetWorkPlaneTool { return &OffsetWorkPlaneTool{} }

// Name implements [Tool].
func (t *OffsetWorkPlaneTool) Name() string { return "Offset Plane" }

// Start filters selection to work planes and seeds the base from a pre-selected plane (so
// a user who picked a plane first only needs to enter the distance).
func (t *OffsetWorkPlaneTool) Start(s *Session) {
	t.prev = s.Selection().Filter()
	t.base = s.SelectedWorkPlane()
	s.Selection().Clear()
	s.Selection().SetFilter(NewSelectionFilter(SelectWorkPlane))
}

// Pick records the plane to offset from.
func (t *OffsetWorkPlaneTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(WorkPlaneHandle); ok {
		t.base = h.Plane
	}
}

// SetDistance / Distance hold the offset in model units (set from the dialog field via the
// session bridge, which converts the displayed value from the document's length unit).
func (t *OffsetWorkPlaneTool) SetDistance(d float64) { t.distance = d }
func (t *OffsetWorkPlaneTool) Distance() float64     { return t.distance }

// BasePicked reports whether the plane to offset from has been chosen, so the dialog knows
// to prompt for the pick vs. the distance.
func (t *OffsetWorkPlaneTool) BasePicked() bool { return t.base != nil }

// CanCommit requires a base plane and a non-zero offset (so OK stays disabled until both
// are gathered — the tool never silently creates a plane).
func (t *OffsetWorkPlaneTool) CanCommit() bool { return t.base != nil && t.distance != 0 }

// Commit creates the offset work plane at the entered distance and recomputes.
func (t *OffsetWorkPlaneTool) Commit(s *Session) error {
	s.Selection().SetFilter(t.prev)
	if t.base == nil {
		return errors.New("offset plane: no base plane picked")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d := t.distance
	t.added = finishWorkPlane(part, part.WorkPlanes().AddByPlaneAndOffset(t.base.Key(), func() float64 { return d }))
	return nil
}

// Cancel restores the prior selection filter with no change.
func (t *OffsetWorkPlaneTool) Cancel(s *Session) { s.Selection().SetFilter(t.prev) }

// Prompt guides the user through the two steps (Inventor's status-bar prompts).
func (t *OffsetWorkPlaneTool) Prompt(*Session) string {
	if t.base == nil {
		return "Select a plane or planar face to offset from"
	}
	return "Enter the offset distance, then click OK"
}

// AddedPlane returns the plane created on commit (for inspection/tests).
func (t *OffsetWorkPlaneTool) AddedPlane() *feature.WorkPlane { return t.added }
