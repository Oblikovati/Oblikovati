// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestSelectionFilterStateDefaultAcceptsAll(t *testing.T) {
	st := NewSelectionFilterState()
	f := st.Filter()
	if f.IsRestricted() {
		t.Fatal("default state must build an unrestricted (accept-all) filter")
	}
	for _, k := range st.Order() {
		if !st.Enabled(k) || !f.Accepts(k) {
			t.Errorf("default state should enable and accept kind %d", k)
		}
	}
}

func TestSelectionFilterStateDisableRestricts(t *testing.T) {
	st := NewSelectionFilterState()
	st.SetEnabled(SelectFace, false)
	f := st.Filter()
	if !f.IsRestricted() {
		t.Fatal("disabling a kind must produce a restricted filter")
	}
	if f.Accepts(SelectFace) {
		t.Error("a disabled face must not be accepted")
	}
	if !f.Accepts(SelectEdge) {
		t.Error("a still-enabled edge must be accepted")
	}
}

func TestSelectionFilterStateDeselectAllBlocks(t *testing.T) {
	st := NewSelectionFilterState()
	st.DisableAll()
	f := st.Filter()
	if !f.IsRestricted() {
		t.Fatal("Deselect All must produce a restricted (blocking) filter, not accept-all")
	}
	for _, k := range st.Order() {
		if f.Accepts(k) {
			t.Errorf("Deselect All must reject every kind, accepted %d", k)
		}
	}
}

func TestSelectionFilterStateEnableAllRestoresDefault(t *testing.T) {
	st := NewSelectionFilterState()
	st.DisableAll()
	st.EnableAll()
	if st.Filter().IsRestricted() {
		t.Error("Select All must restore the accept-all filter")
	}
}

func TestSelectionFilterStateMoveReordersRank(t *testing.T) {
	st := NewSelectionFilterState()
	if st.Rank(SelectEdge) >= st.Rank(SelectFace) {
		t.Fatalf("default: edge (%d) should outrank face (%d)", st.Rank(SelectEdge), st.Rank(SelectFace))
	}
	face := st.Rank(SelectFace)
	st.Move(face, 0) // drag Faces to the top
	if st.Rank(SelectFace) != 0 {
		t.Errorf("after moving face to top, rank = %d, want 0", st.Rank(SelectFace))
	}
	if st.Rank(SelectFace) >= st.Rank(SelectEdge) {
		t.Error("face moved to top must now outrank edge")
	}
}

func TestSelectionFilterStateMoveIgnoresOutOfRange(t *testing.T) {
	st := NewSelectionFilterState()
	before := st.Order()
	st.Move(-1, 0)
	st.Move(0, len(before))
	st.Move(2, 2)
	after := st.Order()
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("out-of-range/no-op Move changed the order at %d", i)
		}
	}
}

func TestSelectionFilterStateOrderIsCopy(t *testing.T) {
	st := NewSelectionFilterState()
	got := st.Order()
	got[0] = SelectFace
	if st.Order()[0] == SelectFace && len(defaultFilterableKinds()) > 0 && defaultFilterableKinds()[0] != SelectFace {
		t.Error("Order() must return a copy; mutating it changed internal state")
	}
}

// TestSelectionPriorityPresetsMatchPriorityFilter pins that the four priority presets write the
// exact kind set priorityFilter defines, so the ribbon combo and the window agree (#1222).
func TestSelectionPriorityPresetsMatchPriorityFilter(t *testing.T) {
	priorities := []SelectionPriority{PriorityGeneral, PriorityPart, PriorityFace, PriorityEdge}
	kinds := []SelectionKind{SelectFace, SelectEdge, SelectBody, SelectOccurrence, SelectVertex, SelectProfile}
	for _, p := range priorities {
		st := NewSelectionFilterState()
		st.applyPriorityPreset(p)
		got, want := st.Filter(), priorityFilter(p)
		for _, k := range kinds {
			if got.Accepts(k) != want.Accepts(k) {
				t.Errorf("priority %d kind %d: preset accepts %v, priorityFilter accepts %v",
					p, k, got.Accepts(k), want.Accepts(k))
			}
		}
	}
}

func TestPickFilterReflectsAmbientState(t *testing.T) {
	s := NewSession()
	s.SelectionFilterState().SetEnabled(SelectFace, false)
	if s.pickFilter().Accepts(SelectFace) {
		t.Error("with no tool, pickFilter must reflect the ambient state (face disabled)")
	}
	if !s.pickFilter().Accepts(SelectEdge) {
		t.Error("with no tool, an enabled edge must still be accepted")
	}
}
