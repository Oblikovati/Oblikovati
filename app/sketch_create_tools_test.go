// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
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

// The text window's font + alignment choices wire through to the committed entity.
func TestSketchTextToolStyleWiresThrough(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewSketchTextTool()
	s.StartTool(tool)
	s.Click(100, 100)
	tool.SetText("HI")

	choices := tool.Params().Choices
	if len(choices) != 3 {
		t.Fatalf("text tool choices = %d, want 3 (font, alignment, vertical)", len(choices))
	}
	choices[1].Set(int(sketch.TextCenter)) // Alignment → Center
	choices[2].Set(int(sketch.TextMiddle)) // Vertical → Middle

	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	tb := sk.TextBoxes().Item(0)
	if tb.Justify != sketch.TextCenter || tb.VJustify != sketch.TextMiddle {
		t.Errorf("committed text = %+v, want center/middle alignment", tb)
	}
	if tb.Family != "Liberation Sans" {
		t.Errorf("committed font = %q, want Liberation Sans (default choice)", tb.Family)
	}
}

// The committed text embeds its chosen font as a document resource (ADR-0031): the default is
// a bundled face, recorded as a bytes-less "embedded" resource the text resolves by.
func TestSketchTextToolEmbedsFontResource(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewSketchTextTool()
	s.StartTool(tool)
	s.Click(100, 100)
	tool.SetText("HI")
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	tb := sk.TextBoxes().Item(0)
	if tb.FontResource == "" {
		t.Fatal("committed text has no font resource (ADR-0031 embed-on-commit)")
	}
	part, err := activePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	r, ok := part.Resource(tb.FontResource)
	if !ok || r.Type != "TrueTypeFont" || r.Encoding != "embedded" || len(r.Value) != 0 {
		t.Errorf("font resource = %+v, want a bytes-less embedded TrueTypeFont (bundled face)", r)
	}
	if ft, err := part.Resolve(tb.FontResource); err != nil || ft == nil {
		t.Errorf("Resolve(font resource) = %v, %v", ft, err)
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
