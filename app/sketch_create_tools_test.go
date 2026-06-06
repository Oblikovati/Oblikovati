// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	gmath "oblikovati/math"
)

// Chamfer bevels the corner of two picked lines, adding the bevel line.
func TestSketchChamferTool(t *testing.T) {
	s, sk := sketchSession(t)
	corner := sk.Points().Add(gmath.P2(0, 0))
	l1 := sk.Lines().Add(corner, sk.Points().Add(gmath.P2(4, 0)))
	l2 := sk.Lines().Add(corner, sk.Points().Add(gmath.P2(0, 4)))
	before := sk.Lines().Count()

	tool := NewSketchChamferTool(1)
	s.StartTool(tool)
	s.feedPick(SketchEntityHandle{Entity: l1})
	s.feedPick(SketchEntityHandle{Entity: l2}) // second pick auto-commits

	if s.ActiveTool() != nil {
		t.Fatal("chamfer tool should deactivate after two picks")
	}
	if sk.Lines().Count() != before+1 {
		t.Fatalf("lines after chamfer = %d, want %d (the bevel)", sk.Lines().Count(), before+1)
	}
}

// Slot draws a straight slot from two centre clicks (two lines + two arc caps).
func TestSketchSlotTool(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewSketchSlotTool(1)
	s.StartTool(tool)
	s.Click(80, 100)
	s.Click(120, 100) // second click auto-commits

	if s.ActiveTool() != nil {
		t.Fatal("slot tool should deactivate after two clicks")
	}
	if sk.Lines().Count() < 2 || sk.Arcs().Count() < 2 {
		t.Fatalf("slot made %d lines / %d arcs, want >=2 each", sk.Lines().Count(), sk.Arcs().Count())
	}
}

// Text places a text box at the clicked anchor once a string is set.
func TestSketchTextTool(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewSketchTextTool()
	s.StartTool(tool)
	s.Click(100, 100) // anchor (does not auto-commit — text is entered first)
	tool.SetText("HELLO")
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sk.TextBoxes().Count() != 1 {
		t.Fatalf("text boxes = %d, want 1", sk.TextBoxes().Count())
	}
}

// Text with no string is not ready to commit.
func TestSketchTextToolNeedsString(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewSketchTextTool()
	s.StartTool(tool)
	s.Click(100, 100)
	if err := s.OK(); err == nil {
		t.Error("text without a string should not commit")
	}
}

func TestSketchCreateToolsRegistered(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{"Sketch.Slot", "Sketch.Chamfer", "Sketch.Text"} {
		if _, ok := s.Commands().ByID(id); !ok {
			t.Errorf("command %q not registered", id)
		}
	}
}
