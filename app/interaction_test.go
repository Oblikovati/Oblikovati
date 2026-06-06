// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/event"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
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

func TestPlainClickReplacesSelectionShiftAdds(t *testing.T) {
	s := NewSession()
	s.SetPicker(stubPicker{sel: FaceHandle{Face: aFace()}})
	s.Click(10, 10)
	s.Click(20, 20) // plain click replaces → still 1
	if s.Selection().Count() != 1 {
		t.Errorf("plain re-click count = %d, want 1 (replace)", s.Selection().Count())
	}
	s.Pointer(PointerEvent{X: 30, Y: 30, Button: LeftButton, Mods: ShiftMod}) // shift adds
	if s.Selection().Count() != 2 {
		t.Errorf("shift-click count = %d, want 2 (add)", s.Selection().Count())
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
