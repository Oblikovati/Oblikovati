// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Direct manipulation of sketch geometry: in the sketch editor a left-drag that starts on an
// unconstrained entity moves it (and any entities sharing its points) live, following the
// cursor — Inventor's drag-to-move. Only MoveableFree geometry drags; fixed / fully-dimensioned
// entities fall through to click-select. Constrained re-solving during the drag is a follow-up
// (#909); v1 translates the picked free geometry directly, which is the unconstrained case the
// user works in.

// sketchDrag is an in-progress direct drag of one or more sketch entities. lastPt is the cursor
// position (in sketch coordinates) at the previous frame, so each update applies the incremental
// delta.
type sketchDrag struct {
	ents   []sketch.Entity
	lastPt math.Point2
	active bool
}

// EntityDragActive reports whether a sketch entity is currently being dragged.
func (s *Session) EntityDragActive() bool { return s.entityDrag.active }

// BeginEntityDrag arms a direct drag of the unconstrained sketch entity under (px,py). It
// returns false — so the caller falls back to click-select — when not editing a sketch, a tool
// is active, nothing draggable is under the cursor, or the entity is fixed/fully-dimensioned.
// The dragged set is the current sketch selection when the picked entity is part of it,
// otherwise just the picked entity (which it also selects).
func (s *Session) BeginEntityDrag(px, py float64) bool {
	if s.activeSketch == nil || s.tool != nil {
		return false
	}
	ent, ok := s.pickSketchEntity(px, py)
	if !ok || s.activeSketch.MoveableClassifier().Of(ent) != types.MoveableFree {
		return false
	}
	start, ok := screenToSketch(s, px, py)
	if !ok {
		return false
	}
	s.entityDrag = sketchDrag{ents: s.dragSet(ent), lastPt: start, active: true}
	return true
}

// dragSet returns the entities a drag of ent should move: the selected sketch entities when ent
// is among them (drag the whole multi-selection), otherwise just ent — which it makes the new
// selection, mirroring a click.
func (s *Session) dragSet(ent sketch.Entity) []sketch.Entity {
	sel := s.selectedSketchEntities()
	for _, e := range sel {
		if e == ent {
			return sel
		}
	}
	s.selection.Clear()
	if s.selection.Add(SketchEntityHandle{Entity: ent}) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
	return []sketch.Entity{ent}
}

// selectedSketchEntities returns the sketch entities currently in the selection set.
func (s *Session) selectedSketchEntities() []sketch.Entity {
	var out []sketch.Entity
	for _, it := range s.selection.Items() {
		if h, ok := it.(SketchEntityHandle); ok {
			out = append(out, h.Entity)
		}
	}
	return out
}

// UpdateEntityDrag translates the dragged entities by the cursor delta since the last frame, so
// the geometry follows the pointer. The sketch overlay rebuilds from the live sketch each frame,
// so the move is visible immediately.
func (s *Session) UpdateEntityDrag(px, py float64) {
	if !s.entityDrag.active {
		return
	}
	cur, ok := screenToSketch(s, px, py)
	if !ok {
		return
	}
	delta := s.entityDrag.lastPt.VectorTo(cur)
	s.activeSketch.MoveEntities(s.entityDrag.ents, delta)
	s.entityDrag.lastPt = cur
}

// CommitEntityDrag ends the drag. The geometry was moved live during the drag, so this only
// clears the drag state. (An undo snapshot lands when sketch editing joins the transaction
// stream — see [[undo-redo-event-stream]].)
func (s *Session) CommitEntityDrag() { s.entityDrag = sketchDrag{} }

// CancelEntityDrag abandons the drag state (e.g. on Escape). Live edits already applied are not
// rolled back in v1.
func (s *Session) CancelEntityDrag() { s.entityDrag = sketchDrag{} }
