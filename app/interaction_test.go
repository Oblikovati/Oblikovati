// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// stubPicker returns a fixed selectable for any in-range click, honoring the filter
// — the headless stand-in for viewport ID-buffer picking (so tests "click on" it).
type stubPicker struct{ sel Selectable }

func (p stubPicker) Pick(_, _ float64, filter *SelectionFilter) (Selectable, bool) {
	if p.sel == nil || !filter.Accepts(p.sel.SelectionKind()) {
		return nil, false
	}
	return p.sel, true
}

func aFace() *topo.Face {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v, v, topo.NewLineage(topo.Tok("f", "edge", 0)))
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	return bld.Build().Faces()[0]
}

func TestClickSelectsHonoringFilterAndFiresEvent(t *testing.T) {
	s := NewSession()
	s.SetPicker(stubPicker{sel: FaceHandle{Face: aFace()}})

	changes := 0
	event.Subscribe(s.Events(), event.After, func(_ event.Context, _ SelectionChanged) event.Outcome { changes++; return event.Continue() })

	s.Click(50, 50)
	if s.Selection().Count() != 1 || changes != 1 {
		t.Fatalf("click select: count=%d changes=%d, want 1/1", s.Selection().Count(), changes)
	}
	if _, ok := s.Selection().First().(FaceHandle); !ok {
		t.Error("selected item is not a FaceHandle")
	}

	// A filter that excludes faces rejects the same click.
	s.Selection().Clear()
	s.Selection().SetFilter(NewSelectionFilter(SelectEdge))
	s.Click(50, 50)
	if s.Selection().Count() != 0 {
		t.Error("face selected despite an edge-only filter")
	}
}

func TestPlainClickReplacesShiftClickTogglesPerObject(t *testing.T) {
	s := NewSession()
	faceA := FaceHandle{Face: aFace()}
	faceB := FaceHandle{Face: aFace()}

	s.SetPicker(stubPicker{sel: faceA})
	s.Click(10, 10)
	s.Click(20, 20) // plain click replaces → still 1 (and no duplicate)
	if s.Selection().Count() != 1 {
		t.Fatalf("plain re-click count = %d, want 1 (replace, de-duplicated)", s.Selection().Count())
	}

	// Shift+click a different object adds it (GUID-B8F6E805).
	s.SetPicker(stubPicker{sel: faceB})
	s.Pointer(PointerEvent{X: 30, Y: 30, Button: LeftButton, Mods: ShiftMod})
	if s.Selection().Count() != 2 {
		t.Fatalf("shift-click new object count = %d, want 2 (add)", s.Selection().Count())
	}

	// Ctrl+click an already-selected object removes just that object, leaving the rest.
	s.Pointer(PointerEvent{X: 31, Y: 31, Button: LeftButton, Mods: CtrlMod})
	if s.Selection().Count() != 1 || s.Selection().Contains(faceB) {
		t.Fatalf("ctrl-click selected object should toggle it off: count=%d containsB=%v",
			s.Selection().Count(), s.Selection().Contains(faceB))
	}
	if !s.Selection().Contains(faceA) {
		t.Error("toggling faceB off must leave faceA selected")
	}
}

func TestEmptyClickClearsSelection(t *testing.T) {
	s := NewSession()
	s.SetPicker(stubPicker{sel: FaceHandle{Face: aFace()}})
	s.Click(10, 10)
	if s.Selection().Count() != 1 {
		t.Fatalf("setup: expected 1 selected, got %d", s.Selection().Count())
	}

	// A plain click on empty space (picker miss) deselects.
	cleared := 0
	event.Subscribe(s.Events(), event.After, func(_ event.Context, _ SelectionChanged) event.Outcome { cleared++; return event.Continue() })
	s.SetPicker(stubPicker{sel: nil}) // miss everywhere
	s.Click(99, 99)
	if s.Selection().Count() != 0 || cleared != 1 {
		t.Fatalf("empty click should clear + fire once: count=%d events=%d", s.Selection().Count(), cleared)
	}

	// A modifier-held empty click is a no-op (does not clear an existing selection).
	s.SetPicker(stubPicker{sel: FaceHandle{Face: aFace()}})
	s.Click(10, 10)
	s.SetPicker(stubPicker{sel: nil})
	s.Pointer(PointerEvent{X: 99, Y: 99, Button: LeftButton, Mods: ShiftMod})
	if s.Selection().Count() != 1 {
		t.Errorf("shift+empty-click must not clear: count=%d", s.Selection().Count())
	}
}

func TestSelectionAddDedupesAndToggle(t *testing.T) {
	sel := NewSelection()
	a := FaceHandle{Face: aFace()}
	b := EdgeHandle{Edge: nil}

	if !sel.Add(a) || sel.Add(a) { // second Add is a no-op (de-dup)
		t.Fatalf("Add must add once then de-dup: count=%d", sel.Count())
	}
	if sel.Count() != 1 || !sel.Contains(a) {
		t.Fatalf("after Add(a): count=%d contains=%v", sel.Count(), sel.Contains(a))
	}
	if !sel.Toggle(b) || sel.Count() != 2 { // toggle a new item adds it
		t.Fatalf("Toggle(new) should add: count=%d", sel.Count())
	}
	if !sel.Toggle(a) || sel.Count() != 1 || sel.Contains(a) { // toggle present removes it
		t.Fatalf("Toggle(present) should remove a: count=%d containsA=%v", sel.Count(), sel.Contains(a))
	}
	if !sel.Remove(b) || sel.Remove(b) { // Remove reports presence; second is a no-op
		t.Fatalf("Remove must remove once then report absent: count=%d", sel.Count())
	}
}

func TestKeyAliasInvokesCommand(t *testing.T) {
	s := NewSession()
	ran := false
	_ = s.Commands().Add(NewCommand("line", "Line", "Sketch", func(*Session) error { ran = true; return nil }).WithAlias("L"))
	if err := s.PressKey(KeyEvent{Key: "L"}); err != nil || !ran {
		t.Errorf("alias key did not run command: err=%v ran=%v", err, ran)
	}
}

// stubTool records lifecycle for tool-routing tests.
type stubTool struct {
	started, cancelled bool
	picks              int
	ready              bool
	committed          bool
}

func (s *stubTool) Name() string              { return "Stub" }
func (s *stubTool) Start(*Session)            { s.started = true }
func (s *stubTool) Pick(*Session, Selectable) { s.picks++ }
func (s *stubTool) CanCommit() bool           { return s.ready }
func (s *stubTool) Commit(*Session) error     { s.committed = true; return nil }
func (s *stubTool) Cancel(*Session)           { s.cancelled = true }

func TestToolLifecycleRouting(t *testing.T) {
	s := NewSession()
	s.SetPicker(stubPicker{sel: FaceHandle{Face: aFace()}})
	tool := &stubTool{}
	s.StartTool(tool)
	if !tool.started || s.ActiveTool().Tool() != tool {
		t.Fatal("StartTool did not activate the tool")
	}
	// While a tool is active, clicks feed the tool, not the selection set.
	s.Click(1, 1)
	if tool.picks != 1 || s.Selection().Count() != 0 {
		t.Errorf("tool routing: picks=%d selection=%d, want 1/0", tool.picks, s.Selection().Count())
	}
	// OK before ready is refused.
	if err := s.OK(); err == nil {
		t.Error("OK should refuse a not-ready tool")
	}
	tool.ready = true
	if err := s.PressKey(KeyEvent{Key: "Enter"}); err != nil || !tool.committed {
		t.Errorf("Enter did not commit ready tool: err=%v committed=%v", err, tool.committed)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should clear after commit")
	}
	// Escape cancels an active tool.
	s.StartTool(&stubTool{})
	_ = s.PressKey(KeyEvent{Key: "Escape"})
	if s.ActiveTool() != nil {
		t.Error("Escape did not cancel the tool")
	}
}
