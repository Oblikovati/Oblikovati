// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// DeleteEntities removes a set of sketch entities — the model behind the Delete key in the
// 2D sketch editor (issue #1232). Each entity is dropped from the geometry list and its
// typed collection; then the points no surviving entity references any longer (curve
// endpoints/centers and standalone points) are pruned, and the geometric constraints and
// dimensions bound to those dropped points are removed so no dangling relation survives.
// nil and not-present entities are skipped. Returns the number of entities actually removed.
//
//	n := sk.DeleteEntities([]Entity{line, circle}) // n == 2; their orphaned points + constraints go too
func (s *Sketch) DeleteEntities(ents []Entity) int {
	removed := 0
	for _, e := range ents {
		if e == nil {
			continue
		}
		before := len(s.ents)
		s.deleteEntity(e)
		if len(s.ents) < before {
			removed++
		}
	}
	if removed == 0 {
		return 0
	}
	s.dropConstraintsOnVars(droppedPointVars(s.pruneOrphanPoints()))
	return removed
}

// pruneOrphanPoints rebuilds the constrainable-point list from the surviving entities,
// dropping the points no remaining entity references, and returns the dropped points so
// their constraints can be detached. It mirrors [Sketch3D.pruneOrphanPoints3D].
func (s *Sketch) pruneOrphanPoints() []*Point {
	live := map[*Point]bool{}
	for _, e := range s.ents {
		for _, p := range entityPoints(e) {
			live[p] = true
		}
	}
	kept := make([]*Point, 0, len(s.pts))
	var dropped []*Point
	for _, p := range s.pts {
		if live[p] {
			kept = append(kept, p)
		} else {
			dropped = append(dropped, p)
		}
	}
	s.pts = kept
	return dropped
}

// droppedPointVars returns the solver-variable set (the X/Y scalars) of the dropped points,
// the key dropConstraintsOnVars matches constraints against by pointer identity.
func droppedPointVars(dropped []*Point) map[*math.Scalar]bool {
	vars := make(map[*math.Scalar]bool, len(dropped)*2)
	for _, p := range dropped {
		vars[&p.X] = true
		vars[&p.Y] = true
	}
	return vars
}
