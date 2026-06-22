// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/event"

// DeleteSelectedSketchEntities removes the sketch entities currently selected in the 2D
// sketch editor — the Delete-key action (issue #1232: pressing Delete on selected sketch
// geometry did nothing because no action was bound). It drops each entity along with its
// orphaned points and the constraints bound to them (sketch.DeleteEntities), clears the
// selection, recomputes the active part so downstream features see the change, and records
// one undoable edit. It is a no-op (nil) when not editing a sketch, a tool is mid-operation,
// or nothing is selected.
func (s *Session) DeleteSelectedSketchEntities() error {
	if s.activeSketch == nil || s.tool != nil {
		return nil
	}
	ents := s.selectedSketchEntities()
	if len(ents) == 0 {
		return nil
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if s.activeSketch.DeleteEntities(ents) == 0 {
		return nil
	}
	s.selection.Clear()
	event.Emit(s.bus, event.After, SelectionChanged{Count: 0})
	part.Recompute()
	s.recordEdit(part, "Delete Sketch Entities")
	return nil
}
