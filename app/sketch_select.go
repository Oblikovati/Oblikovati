// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati/event"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// In the sketch environment a plain click selects the nearest sketch entity (a point,
// line, circle or arc) rather than 3D geometry — the input for the constraint and
// dimension commands. Shift/Ctrl extend the selection (multi-select), matching the 3D
// behavior. Picking is pure plane math: the click is mapped to the sketch plane and the
// closest entity within the zoom-scaled tolerance wins, points before curves.

// IsSelectedEntity reports whether the given sketch entity is highlighted — either in
// the current selection or picked by the active constraint/dimension tool.
func (s *Session) IsSelectedEntity(e sketch.Entity) bool {
	for _, it := range s.selection.Items() {
		if h, ok := it.(SketchEntityHandle); ok && h.Entity == e {
			return true
		}
	}
	for _, picked := range s.toolPicked() {
		if picked == e {
			return true
		}
	}
	return false
}

// toolPicked returns the entities the active sketch-entity tool has picked, or nil.
func (s *Session) toolPicked() []sketch.Entity {
	if s.tool == nil {
		return nil
	}
	if t, ok := s.tool.tool.(SketchEntityTool); ok {
		return t.Picked()
	}
	return nil
}

// HoverCandidate returns the sketch entity under the cursor that the active
// constraint/dimension tool would accept — the geometry the head highlights to show
// what is selectable for the current constraint. Returns false when no such tool is
// active, nothing is under the cursor, or the entity is not a valid pick.
func (s *Session) HoverCandidate(px, py float64) (sketch.Entity, bool) {
	if s.tool == nil {
		return nil, false
	}
	t, ok := s.tool.tool.(SketchEntityTool)
	if !ok {
		return nil, false
	}
	ent, found := s.pickSketchEntity(px, py)
	if !found || !t.Accepts(ent) {
		return nil, false
	}
	return ent, true
}

// sketchEntityPointer feeds a sketch-entity pick to the active constraint/dimension
// tool, or (with no such tool) updates the selection — replacing it, or extending it
// when Shift/Ctrl is held.
func (s *Session) sketchEntityPointer(e PointerEvent) {
	ent, found := s.pickSketchEntity(e.X, e.Y)
	if s.tool != nil {
		if t, ok := s.tool.tool.(SketchEntityTool); ok && found {
			snap, _ := s.SnapAt(e.X, e.Y)
			t.PickSnap(ent, snap)
			s.autoCommitAfterPick()
		}
		return
	}
	additive := e.Mods.Has(ShiftMod) || e.Mods.Has(CtrlMod)
	if !found {
		if !additive {
			s.selection.Clear()
			event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
		}
		return
	}
	if !additive {
		s.selection.Clear()
	}
	if s.selection.Add(SketchEntityHandle{Entity: ent}) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// pickSketchEntity returns the active sketch's entity nearest the clicked pixel within
// the selection tolerance, preferring points over curves.
func (s *Session) pickSketchEntity(px, py float64) (sketch.Entity, bool) {
	if s.activeSketch == nil {
		return nil, false
	}
	p, ok := screenToSketch(s, px, py)
	if !ok {
		return nil, false
	}
	tol := s.snapTolerance()
	if pt, ok := s.nearestEntityPoint(p, tol); ok {
		return pt, true
	}
	return s.nearestEntityCurve(p, tol)
}

// nearestEntityPoint returns the standalone/defining point nearest p within tol.
func (s *Session) nearestEntityPoint(p math.Point2, tol float64) (sketch.Entity, bool) {
	var best *sketch.Point
	bestD := tol
	for _, pt := range s.activeSketch.AllPoints() {
		if d := p.DistanceTo(pt.Position()); d <= bestD {
			best, bestD = pt, d
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

// nearestEntityCurve returns the line/circle/arc whose outline is nearest p within tol.
func (s *Session) nearestEntityCurve(p math.Point2, tol float64) (sketch.Entity, bool) {
	var best sketch.Entity
	bestD := tol
	consider := func(ent sketch.Entity, d float64) {
		if d <= bestD {
			best, bestD = ent, d
		}
	}
	sk := s.activeSketch
	for i := 0; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		consider(l, segmentDistance(p, l.A.Position(), l.B.Position()))
	}
	for i := 0; i < sk.Circles().Count(); i++ {
		c := sk.Circles().Item(i)
		consider(c, circleOutlineDistance(p, c.Center.Position(), c.Radius))
	}
	for i := 0; i < sk.Arcs().Count(); i++ {
		a := sk.Arcs().Item(i)
		consider(a, circleOutlineDistance(p, a.Center.Position(), a.Radius()))
	}
	return best, best != nil
}

// segmentDistance returns the distance from p to the segment a–b.
func segmentDistance(p, a, b math.Point2) float64 {
	return p.DistanceTo(segmentClosestPoint(p, a, b))
}
