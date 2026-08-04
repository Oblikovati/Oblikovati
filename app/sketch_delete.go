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
// code. Both halves landed together.
func (s *Session) DeleteSelectedSketchEntities() error {
	if s.activeSketch == nil || s.tool != nil {
		return nil
	}
	ents, dims := s.selectedSketchEntities(), s.SelectedSketchDimensions()
	if len(ents) == 0 && len(dims) == 0 {
		return nil
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if s.activeSketch.DeleteEntities(ents)+s.deleteSketchDimensions(dims) == 0 {
		return nil
	}
	s.selection.Clear()
	event.Emit(s.bus, event.After, SelectionChanged{Count: 0})
	part.Recompute()
	s.recordEdit(part, deleteEditName(ents, dims))
	return nil
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
// truthfully when only dimensions were selected.
func deleteEditName(ents []sketch.Entity, dims []*sketch.DimensionConstraint) string {
	if len(ents) == 0 {
		return "Delete Sketch Dimensions"
	}
	if len(dims) == 0 {
		return "Delete Sketch Entities"
	}
	return "Delete Sketch Geometry"
}
