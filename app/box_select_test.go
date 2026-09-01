// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// fakeRegionPicker records the last region query and returns canned hits keyed on crossing
// mode, so tests drive box-select without the renderer projection.
type fakeRegionPicker struct {
	windowHits   []Selectable
	crossingHits []Selectable
	lastCrossing bool
	calls        int
}

func (f *fakeRegionPicker) PickRegion(_, _, _, _ float64, crossing bool, _ *SelectionFilter) []Selectable {
	f.calls++
	f.lastCrossing = crossing
	if crossing {
		return f.crossingHits
	}
	return f.windowHits
}

func TestBeginBoxSelectNoopWithoutRegionPicker(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.BeginBoxSelect(10, 10)
	if s.BoxSelectActive() {
		t.Error("box-select must not begin when no RegionPicker is installed")
	}
}

func TestBoxSelectWindowVsCrossingDirection(t *testing.T) {
	t.Parallel()
	a, b := FaceHandle{Face: aFace()}, FaceHandle{Face: aFace()}
	rp := &fakeRegionPicker{windowHits: []Selectable{a}, crossingHits: []Selectable{a, b}}
	s := NewSession()
	s.SetRegionPicker(rp)

	// Drag left→right → window select (only fully-enclosed): one hit.
	s.BeginBoxSelect(20, 20)
	s.UpdateBoxSelect(80, 60)
	if _, _, _, _, crossing := s.BoxSelectRect(); crossing {
		t.Error("left→right drag should be a window (non-crossing) select")
	}
	if n := s.CommitBoxSelect(0); n != 1 || s.Selection().Count() != 1 {
		t.Fatalf("window select: hits=%d count=%d, want 1/1", n, s.Selection().Count())
	}
	if rp.lastCrossing {
		t.Error("window select must query the region picker with crossing=false")
	}
	if s.BoxSelectActive() {
		t.Error("commit must end the drag")
	}

	// Drag right→left → crossing select (enclosed + intersected): two hits, replacing.
	s.BeginBoxSelect(80, 60)
	s.UpdateBoxSelect(20, 20)
	if _, _, _, _, crossing := s.BoxSelectRect(); !crossing {
		t.Error("right→left drag should be a crossing select")
	}
	if n := s.CommitBoxSelect(0); n != 2 || s.Selection().Count() != 2 {
		t.Fatalf("crossing select: hits=%d count=%d, want 2/2", n, s.Selection().Count())
	}
}

func TestBoxSelectModifiers(t *testing.T) {
	t.Parallel()
	a, b, c := FaceHandle{Face: aFace()}, FaceHandle{Face: aFace()}, FaceHandle{Face: aFace()}
	rp := &fakeRegionPicker{windowHits: []Selectable{a, b}}
	s := NewSession()
	s.SetRegionPicker(rp)
	s.Selection().Add(c) // pre-existing selection

	// Plain box replaces: c is dropped, {a,b} selected.
	s.BeginBoxSelect(0, 0)
	s.UpdateBoxSelect(10, 10)
	s.CommitBoxSelect(0)
	if s.Selection().Count() != 2 || s.Selection().Contains(c) {
		t.Fatalf("plain box should replace: count=%d containsC=%v", s.Selection().Count(), s.Selection().Contains(c))
	}

	// Shift box adds (union) without dropping the current set.
	rp.windowHits = []Selectable{c}
	s.BeginBoxSelect(0, 0)
	s.UpdateBoxSelect(10, 10)
	s.CommitBoxSelect(ShiftMod)
	if s.Selection().Count() != 3 {
		t.Fatalf("shift box should add: count=%d, want 3", s.Selection().Count())
	}

	// Ctrl box inverts: a and b were selected → removed; nothing added.
	rp.windowHits = []Selectable{a, b}
	s.BeginBoxSelect(0, 0)
	s.UpdateBoxSelect(10, 10)
	s.CommitBoxSelect(CtrlMod)
	if s.Selection().Count() != 1 || !s.Selection().Contains(c) {
		t.Fatalf("ctrl box should invert a,b off leaving c: count=%d containsC=%v",
			s.Selection().Count(), s.Selection().Contains(c))
	}
}

func TestBoxSelectGuardsWhenInactive(t *testing.T) {
	t.Parallel()
	rp := &fakeRegionPicker{windowHits: []Selectable{FaceHandle{Face: aFace()}}}
	s := NewSession()
	s.SetRegionPicker(rp)

	// Update/Commit are no-ops when no drag is in progress.
	s.UpdateBoxSelect(5, 5) // must not panic or activate
	if s.BoxSelectActive() {
		t.Error("UpdateBoxSelect with no active drag must not activate one")
	}
	if n := s.CommitBoxSelect(0); n != 0 || rp.calls != 0 {
		t.Errorf("CommitBoxSelect with no active drag should be a no-op: n=%d calls=%d", n, rp.calls)
	}

	// Commit with an active box but no RegionPicker installed clears the box and selects nothing.
	s2 := NewSession()
	s2.boxSelect = BoxSelection{X0: 0, Y0: 0, X1: 9, Y1: 9, Active: true}
	if n := s2.CommitBoxSelect(0); n != 0 || s2.BoxSelectActive() {
		t.Errorf("commit without a RegionPicker should drop the box: n=%d active=%v", n, s2.BoxSelectActive())
	}
}

func TestCancelBoxSelectLeavesSelectionUntouched(t *testing.T) {
	t.Parallel()
	a := FaceHandle{Face: aFace()}
	rp := &fakeRegionPicker{windowHits: []Selectable{a}}
	s := NewSession()
	s.SetRegionPicker(rp)
	s.BeginBoxSelect(0, 0)
	s.UpdateBoxSelect(10, 10)
	s.CancelBoxSelect()
	if s.BoxSelectActive() || rp.calls != 0 || s.Selection().Count() != 0 {
		t.Errorf("cancel must drop the box without querying or selecting: active=%v calls=%d count=%d",
			s.BoxSelectActive(), rp.calls, s.Selection().Count())
	}
}
