// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/sketch"
)

// DeleteSelectedSketchEntities removes the sketch entities AND dimensions currently selected in
// the 2D sketch editor — the Delete-key action (issue #1232: pressing Delete on selected sketch
// geometry did nothing because no action was bound). It drops each entity along with its
// orphaned points and the constraints bound to them (sketch.DeleteEntities), clears the
// selection, recomputes the active part so downstream features see the change, and records
// one undoable edit. It is a no-op (nil) when not editing a sketch, a tool is mid-operation,
// or nothing is selected.
//
// Dimensions were originally left out because nothing could select one (#2017): the handle
// existed but no pick produced it, so a selected-dimension branch here would have been dead
// code. Both halves landed together. Geometric constraints joined on the same terms once Show
// Constraints gave them a marker to click.
func (s *Session) DeleteSelectedSketchEntities() error {
	if s.activeSketch == nil || s.tool != nil {
		return nil
	}
	ents, dims, cons := s.selectedSketchEntities(), s.SelectedSketchDimensions(), s.SelectedSketchConstraints()
	if len(ents) == 0 && len(dims) == 0 && len(cons) == 0 {
		return nil
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if s.activeSketch.DeleteEntities(ents)+s.deleteSketchDimensions(dims)+s.deleteSketchConstraints(cons) == 0 {
		return nil
	}
	s.selection.Clear()
	event.Emit(s.bus, event.After, SelectionChanged{Count: 0})
	part.Recompute()
	s.recordEdit(part, deleteEditName(ents, dims, cons))
	return nil
}

// DeleteSelectedSketch3DConstraints removes the geometric constraints selected in the active 3D
// sketch — the Delete-key action once a 3D constraint marker can be clicked (#1998 follow-up). It is
// scoped to CONSTRAINTS on purpose: 3D sketch entities are not deleted from the viewport (that stays
// on the browser, so a stray Delete never destroys 3D geometry), only the relations a marker
// selects. Removing one frees the degrees of freedom it held, so the part recomputes and re-solves.
// A no-op (nil) when not editing a 3D sketch, a tool is mid-operation, or nothing is selected.
func (s *Session) DeleteSelectedSketch3DConstraints() error {
	if s.activeSketch3D == nil || s.tool != nil {
		return nil
	}
	cons := s.SelectedSketchConstraints()
	if len(cons) == 0 {
		return nil
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	gc := s.activeSketch3D.GeometricConstraints3D()
	n := 0
	for _, c := range cons {
		if err := gc.DeleteAllowed(c); err == nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	s.selection.Clear()
	event.Emit(s.bus, event.After, SelectionChanged{Count: 0})
	part.Recompute()
	s.recordEdit(part, "Delete Sketch Constraints")
	return nil
}

// deleteSketchConstraints drops each selected geometric constraint, returning how many were
// removed. It goes through DeleteAllowed rather than Delete so a system-owned relation is refused
// even if one somehow reached the selection — Show Constraints already declines to draw those, and
// this keeps the two halves from drifting. Removing a constraint frees the degrees of freedom it
// held, which is why the caller re-solves.
func (s *Session) deleteSketchConstraints(cons []sketch.Constraint) int {
	gc := s.activeSketch.GeometricConstraints()
	n := 0
	for _, c := range cons {
		if err := gc.DeleteAllowed(c); err == nil {
			n++
		}
	}
	return n
}

// deleteSketchDimensions drops each selected dimension from the active sketch, returning how many
// were present. Deleting one frees the degree of freedom it held, which is why the caller
// recomputes: the sketch re-solves and every downstream feature sees the looser geometry (#2017).
// The dimension owns its backing parameter, and DimensionConstraints.Delete removes that with it.
func (s *Session) deleteSketchDimensions(dims []*sketch.DimensionConstraint) int {
	dc := s.activeSketch.DimensionConstraints()
	n := 0
	for _, d := range dims {
		if dc.Delete(d) {
			n++
		}
	}
	return n
}

// deleteEditName names the undo step for what was actually deleted, so the undo stack reads
// truthfully when only dimensions or only constraints were selected.
func deleteEditName(ents []sketch.Entity, dims []*sketch.DimensionConstraint, cons []sketch.Constraint) string {
	switch {
	case len(ents) == 0 && len(dims) == 0:
		return "Delete Sketch Constraints"
	case len(ents) == 0 && len(cons) == 0:
		return "Delete Sketch Dimensions"
	case len(dims) == 0 && len(cons) == 0:
		return "Delete Sketch Entities"
	}
	return "Delete Sketch Geometry"
}
