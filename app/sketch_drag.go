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
// entities fall through to click-select. Each frame the dragged entities' points are pinned to
// their start position plus the cursor delta and the sketch is re-solved (sketch.DragSolve), so
// the geometry follows the cursor *within its constraints* (#909): a coincident neighbour tracks
// along, a horizontal line stays horizontal, and a free point lands exactly on the cursor.

// dragAnchor is one point being dragged, remembered with its position at the drag's start so the
// solve can pin it to start+delta absolutely (no per-frame drift).
type dragAnchor struct {
	p     *sketch.Point
	start math.Point2
}

// sketchDrag is an in-progress direct drag of one or more sketch entities. grabPt is the cursor
// position (sketch coordinates) where the drag began, so the per-frame delta is absolute.
type sketchDrag struct {
	anchors []dragAnchor
	grabPt  math.Point2
	active  bool
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
	if !ok || !s.draggableForMode(ent) {
		return false
	}
	grab, ok := screenToSketch(s, px, py)
	if !ok {
		return false
	}
	s.entityDrag = sketchDrag{anchors: s.dragAnchors(ent), grabPt: grab, active: true}
	return true
}

// draggableForMode reports whether ent may be direct-dragged in the current mode. Normally
// only MoveableFree geometry drags (fixed / fully-dimensioned entities click-select instead);
// in Relax Mode any entity drags, because the solver relaxes the dimensions holding it (#791).
func (s *Session) draggableForMode(ent sketch.Entity) bool {
	if s.relaxMode {
		return true
	}
	return s.activeSketch.MoveableClassifier().Of(ent) == types.MoveableFree
}

// dragAnchors collects the distinct points to pin while dragging — the defining points of every
// entity in the drag set (selection or the clicked entity), each captured at its start position.
func (s *Session) dragAnchors(ent sketch.Entity) []dragAnchor {
	seen := map[*sketch.Point]bool{}
	var anchors []dragAnchor
	for _, e := range s.dragSet(ent) {
		for _, p := range sketch.DefiningPoints(e) {
			if seen[p] {
				continue
			}
			seen[p] = true
			anchors = append(anchors, dragAnchor{p: p, start: p.Position()})
		}
	}
	return anchors
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

// UpdateEntityDrag pins each dragged point to its start position plus the cursor delta and
// re-solves the sketch, so the geometry follows the pointer while honouring its constraints. The
// sketch overlay rebuilds from the live sketch each frame, so the move is visible immediately.
func (s *Session) UpdateEntityDrag(px, py float64) {
	if !s.entityDrag.active {
		return
	}
	cur, ok := screenToSketch(s, px, py)
	if !ok {
		return
	}
	delta := s.entityDrag.grabPt.VectorTo(cur)
	pins := make([]sketch.PinTarget, len(s.entityDrag.anchors))
	for i, a := range s.entityDrag.anchors {
		pins[i] = sketch.PinTarget{P: a.p, Target: a.start.TranslateBy(delta)}
	}
	if s.relaxMode {
		s.activeSketch.RelaxSolve(pins) // relax dimensions so over/fully-constrained geometry follows (#791)
		return
	}
	s.activeSketch.DragSolve(pins)
}

// CommitEntityDrag ends the drag. The geometry was moved live during the drag, so this clears
// the drag state and records the move as one undo step (#1270); the recipe no-op guard ignores a
// drag that didn't actually move anything.
func (s *Session) CommitEntityDrag() {
	active := s.entityDrag.active
	s.entityDrag = sketchDrag{}
	if active {
		s.RecordActiveEdit("Move Sketch Geometry")
	}
}

// CancelEntityDrag abandons the drag state (e.g. on Escape). Live edits already applied are not
// rolled back in v1.
func (s *Session) CancelEntityDrag() { s.entityDrag = sketchDrag{} }
