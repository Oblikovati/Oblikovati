// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
)

// Editing a placed work plane — Inventor's double-click / browser "Edit" on a datum plane.
// Only the plane-and-offset kind carries a single scalar to edit (its offset distance); the
// other plane kinds (three-point, tangent, …) are defined purely by references and have no
// value to type, so they are not opened for edit here (a follow-up redefine flow). Each
// change recomputes live; OK keeps it, Cancel restores the distance captured when editing
// opened. See issue #132.

// WorkPlaneEditTool edits one offset work plane's distance in place.
type WorkPlaneEditTool struct {
	plane    *feature.WorkPlane
	distance float64 // current edited offset, model units
	original float64 // offset captured at open, for Cancel
}

// newWorkPlaneEditTool captures the plane's current offset as the edit's starting value.
func newWorkPlaneEditTool(wp *feature.WorkPlane) *WorkPlaneEditTool {
	d, _ := wp.OffsetDistance()
	return &WorkPlaneEditTool{plane: wp, distance: d, original: d}
}

// Name implements [Tool].
func (t *WorkPlaneEditTool) Name() string { return "Edit " + t.plane.Name() }

// Start is a no-op: the plane is already chosen, the dialog only edits its distance.
func (t *WorkPlaneEditTool) Start(*Session) {}

// Pick ignores viewport picks: an offset edit takes only a typed distance, no geometry.
func (t *WorkPlaneEditTool) Pick(*Session, Selectable) {}

// SetDistance / Distance hold the offset in model units (the head's dialog reads/sets these
// through the session bridge, converting to/from the document's length unit).
func (t *WorkPlaneEditTool) SetDistance(d float64) { t.distance = d }
func (t *WorkPlaneEditTool) Distance() float64     { return t.distance }

// CanCommit allows committing any distance (including 0 — a coincident plane is valid).
func (t *WorkPlaneEditTool) CanCommit() bool { return true }

// Commit writes the edited distance onto the plane, recomputes, and records the edit.
func (t *WorkPlaneEditTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if !t.plane.SetOffsetDistance(t.distance) {
		return errors.New("work plane edit: plane is not an editable offset plane")
	}
	s.endEditScope()
	part.Recompute()
	s.recordEdit(part, "Edit "+t.plane.Name())
	return nil
}

// Cancel restores the offset captured at open and recomputes.
func (t *WorkPlaneEditTool) Cancel(s *Session) {
	t.plane.SetOffsetDistance(t.original)
	s.endEditScope()
	if part, err := activePart(s); err == nil {
		part.Recompute()
	}
}

// BeginEditWorkPlane opens an offset work plane for distance editing (browser double-click /
// Edit menu). Origin coordinate-system planes and non-offset plane kinds have nothing to
// edit, so they are a no-op (matching a feature with no editable parameters).
func (s *Session) BeginEditWorkPlane(h WorkPlaneHandle) {
	if h.Plane == nil || h.Plane.IsCoordinateSystemElement() {
		return
	}
	if _, ok := h.Plane.OffsetDistance(); !ok {
		return
	}
	s.beginEditScope(h.Plane.Seq())
	s.StartTool(newWorkPlaneEditTool(h.Plane))
}

// --- session bridge for the head dialog (offset in the document's length unit) ---

// ActiveWorkPlaneEdit returns the running work-plane edit tool, or nil.
func (s *Session) ActiveWorkPlaneEdit() *WorkPlaneEditTool {
	if s.tool == nil {
		return nil
	}
	t, _ := s.tool.tool.(*WorkPlaneEditTool)
	return t
}

// EditPlaneName returns the name of the work plane being edited (the dialog title), or "".
func (s *Session) EditPlaneName() string {
	if t := s.ActiveWorkPlaneEdit(); t != nil {
		return t.plane.Name()
	}
	return ""
}

// EditPlaneOffsetDisplay returns the edited offset in the document's length unit.
func (s *Session) EditPlaneOffsetDisplay() float64 {
	t := s.ActiveWorkPlaneEdit()
	if t == nil {
		return 0
	}
	return s.DocumentUnits().ToPreferred(param.Q(t.Distance(), param.Length))
}

// SetEditPlaneOffsetDisplay sets the edited offset from a value in the document's length unit.
func (s *Session) SetEditPlaneOffsetDisplay(value float64) {
	if t := s.ActiveWorkPlaneEdit(); t != nil {
		t.SetDistance(s.DocumentUnits().FromPreferred(value, param.Length).Value)
	}
}
