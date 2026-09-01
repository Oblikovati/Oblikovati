// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// fakeSelectingTool is a minimal tool that declares its accepted kinds (changing after the first
// pick, like a multi-step tool) and reports its picks — exercising the selection engine without
// any real geometry.
type fakeSelectingTool struct {
	step      int
	picks     []Selectable
	commitErr error
}

func (t *fakeSelectingTool) Name() string   { return "Fake" }
func (t *fakeSelectingTool) Start(*Session) {}
func (t *fakeSelectingTool) Pick(_ *Session, sel Selectable) {
	t.picks = append(t.picks, sel)
	t.step++
}
func (t *fakeSelectingTool) CanCommit() bool       { return false }
func (t *fakeSelectingTool) Commit(*Session) error { return t.commitErr }
func (t *fakeSelectingTool) Cancel(*Session)       {}
func (t *fakeSelectingTool) Picks() []Selectable   { return t.picks }
func (t *fakeSelectingTool) AcceptedKinds() []SelectionKind {
	if t.step == 0 {
		return []SelectionKind{SelectProfile}
	}
	return []SelectionKind{SelectFace, SelectWorkPlane} // a later step accepts a termination face
}

func TestStartToolInstallsDeclaredFilter(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.StartTool(&fakeSelectingTool{})
	f := s.Selection().Filter()
	if !f.Accepts(SelectProfile) || f.Accepts(SelectFace) {
		t.Fatalf("start step filter = %+v, want only profiles", f)
	}
}

func TestFeedPickReinstallsFilterForNextStep(t *testing.T) {
	t.Parallel()
	s := NewSession()
	tool := &fakeSelectingTool{}
	s.StartTool(tool)
	s.feedPick(ProfileHandle{}) // advance to step 1
	f := s.Selection().Filter()
	if !f.Accepts(SelectFace) || f.Accepts(SelectProfile) {
		t.Fatalf("after the first pick the filter = %+v, want the next step's face/plane kinds", f)
	}
}

// emptyKindsTool declares no restriction, so the engine must leave the filter unrestricted (the
// ambient SelectionFilterState then governs via pickFilter).
type emptyKindsTool struct{ fakeSelectingTool }

func (emptyKindsTool) AcceptedKinds() []SelectionKind { return nil }

func TestEmptyAcceptedKindsLeavesFilterUnrestricted(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.StartTool(&emptyKindsTool{})
	if s.Selection().Filter().IsRestricted() {
		t.Fatal("a tool declaring no kinds must leave the filter unrestricted for the ambient state")
	}
}

func TestCommitRestoresUnrestrictedFilter(t *testing.T) {
	t.Parallel()
	s := NewSession()
	tool := &fakeSelectingTool{}
	s.StartTool(tool)
	if !s.Selection().Filter().IsRestricted() {
		t.Fatal("the tool's declared filter should be restricted while it runs")
	}
	tool.commitErr = nil
	// Make it committable so OK clears it.
	s.tool.tool = &committableSelectingTool{}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if s.Selection().Filter().IsRestricted() {
		t.Fatal("after commit the filter must be unrestricted (handed back to the ambient state)")
	}
}

func TestCancelRestoresUnrestrictedFilter(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.StartTool(&fakeSelectingTool{})
	s.CancelTool()
	if s.Selection().Filter().IsRestricted() {
		t.Fatal("after cancel the filter must be unrestricted")
	}
}

// committableSelectingTool is ready to commit immediately, for the OK path.
type committableSelectingTool struct{ fakeSelectingTool }

func (committableSelectingTool) CanCommit() bool { return true }

func TestToolPicksReportsViaPickingContract(t *testing.T) {
	t.Parallel()
	s := NewSession()
	tool := &fakeSelectingTool{}
	s.StartTool(tool)
	s.feedPick(EdgeHandle{})
	got := s.ToolPicks()
	if len(got) != 1 {
		t.Fatalf("ToolPicks = %d, want 1 picked edge", len(got))
	}
	if _, ok := got[0].(EdgeHandle); !ok {
		t.Fatalf("ToolPicks[0] = %T, want EdgeHandle", got[0])
	}
}

// TestSelectablesWidensTypedPicks pins the shared widening helper (#1657): order and
// element identity preserved, and an empty typed slice widens to an empty non-nil slice
// (matching the make-based behavior of the four per-type helpers it replaced).
func TestSelectablesWidensTypedPicks(t *testing.T) {
	t.Parallel()
	edges := []EdgeHandle{{}, {}}
	got := selectables(edges)
	if len(got) != 2 {
		t.Fatalf("selectables = %d elements, want 2", len(got))
	}
	for i := range got {
		if got[i] != Selectable(edges[i]) {
			t.Errorf("selectables[%d] = %#v, want the widened edge %#v", i, got[i], edges[i])
		}
	}
	if empty := selectables([]FaceHandle{}); empty == nil || len(empty) != 0 {
		t.Errorf("selectables(empty) = %#v, want empty non-nil slice", empty)
	}
}
