// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/event"

// Box-select — Inventor's window/crossing rubber-band selection. A left-drag that starts
// on empty space sweeps a rectangle; on release every covered object joins the selection.
// Direction sets the mode (GUID-B8F6E805): dragging left→right is a WINDOW select (only
// objects fully enclosed by the box), dragging right→left is a CROSSING select (objects
// enclosed OR merely intersected). Shift adds the covered objects to the current set and
// Ctrl inverts their membership; a plain box replaces the selection.
//
// The geometry hit-test (projecting bodies/sketch entities to screen and testing them
// against the rectangle) lives behind RegionPicker so the model stays free of the renderer;
// this file owns only the state machine and how a region result updates the selection set.

// RegionPicker resolves a screen-space selection rectangle to the selectables it covers.
// crossing=false (window) returns only fully-enclosed objects; crossing=true also returns
// objects the rectangle intersects. Coordinates are viewport-local pixels. The real
// implementation projects geometry with the active camera; tests supply a fake.
type RegionPicker interface {
	PickRegion(x0, y0, x1, y1 float64, crossing bool, filter *SelectionFilter) []Selectable
}

// SetRegionPicker installs the hit-test used to resolve box-select rectangles. Passing nil
// disables box-select (BeginBoxSelect becomes a no-op).
func (s *Session) SetRegionPicker(p RegionPicker) { s.regionPicker = p }

// BoxSelection is an in-progress rubber-band rectangle in viewport-local pixels. (X0,Y0) is
// the anchor (press point); (X1,Y1) follows the cursor. It is Active between Begin and the
// commit/cancel that ends the drag.
type BoxSelection struct {
	X0, Y0, X1, Y1 float64
	Active         bool
}

// Crossing reports the select mode from the drag direction: right→left (X1 < X0) is a
// crossing select; left→right is a window select.
func (b BoxSelection) Crossing() bool { return b.X1 < b.X0 }

// BeginBoxSelect starts a rubber-band rectangle anchored at the press point. It is a no-op when
// box-select cannot resolve hits — no RegionPicker installed and not editing a sketch — so callers
// can begin unconditionally.
func (s *Session) BeginBoxSelect(x, y float64) {
	if s.regionPicker == nil && s.activeSketch == nil {
		return
	}
	s.boxSelect = BoxSelection{X0: x, Y0: y, X1: x, Y1: y, Active: true}
}

// UpdateBoxSelect moves the rectangle's free corner to the current cursor position.
func (s *Session) UpdateBoxSelect(x, y float64) {
	if !s.boxSelect.Active {
		return
	}
	s.boxSelect.X1, s.boxSelect.Y1 = x, y
}

// BoxSelectActive reports whether a rubber-band drag is in progress.
func (s *Session) BoxSelectActive() bool { return s.boxSelect.Active }

// BoxSelectRect returns the current rectangle (anchor + free corner) for the head to draw,
// plus whether it is a crossing select (so the head can style it differently).
func (s *Session) BoxSelectRect() (x0, y0, x1, y1 float64, crossing bool) {
	b := s.boxSelect
	return b.X0, b.Y0, b.X1, b.Y1, b.Crossing()
}

// CancelBoxSelect abandons the in-progress rectangle without changing the selection (e.g.
// the drag was reinterpreted, or Esc was pressed).
func (s *Session) CancelBoxSelect() { s.boxSelect = BoxSelection{} }

// CommitBoxSelect resolves the rectangle through the RegionPicker and folds the covered
// objects into the selection per the held modifier (plain=replace, Shift=add, Ctrl=invert),
// then ends the drag. It emits SelectionChanged when the set changed. Returns the number of
// objects the rectangle covered.
func (s *Session) CommitBoxSelect(mods Modifier) int {
	if !s.boxSelect.Active {
		s.boxSelect = BoxSelection{}
		return 0
	}
	b := s.boxSelect
	hits := s.regionHits(b)
	s.boxSelect = BoxSelection{}
	if s.applyRegionToSelection(hits, mods) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
	return len(hits)
}

// regionHits resolves the box rectangle to selectables: the active sketch's entities while
// editing a sketch (2D), otherwise the model bodies via the installed RegionPicker (3D).
func (s *Session) regionHits(b BoxSelection) []Selectable {
	if s.activeSketch != nil {
		return s.pickSketchRegion(b.X0, b.Y0, b.X1, b.Y1, b.Crossing())
	}
	if s.regionPicker == nil {
		return nil
	}
	return s.regionPicker.PickRegion(b.X0, b.Y0, b.X1, b.Y1, b.Crossing(), s.selection.Filter())
}

// applyRegionToSelection folds box-select hits into the selection set: a plain box replaces
// it, Shift adds (union), Ctrl inverts each hit's membership (GUID-B8F6E805). Reports whether
// the set changed.
func (s *Session) applyRegionToSelection(hits []Selectable, mods Modifier) bool {
	changed := false
	if !mods.Has(ShiftMod) && !mods.Has(CtrlMod) {
		changed = s.selection.Count() > 0
		s.selection.Clear()
	}
	for _, h := range hits {
		if mods.Has(CtrlMod) {
			changed = s.selection.Toggle(h) || changed
		} else {
			changed = s.selection.Add(h) || changed
		}
	}
	return changed
}
