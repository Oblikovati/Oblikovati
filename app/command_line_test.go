// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/cmdline"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// sketchCommandSession returns a session in an open XY sketch with the standard commands
// registered, plus the command-line engine — the harness for driving sketch commands.
func sketchCommandSession(t *testing.T) (*Session, *sketch.Sketch, *CommandLine) {
	t.Helper()
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	return s, sk, s.CommandLine()
}

func TestCommandLineDrawsLineFromText(t *testing.T) {
	s, sk, cl := sketchCommandSession(t)
	for _, in := range []string{"LINE", "0,0", "10,0"} {
		if err := cl.Submit(s, in); err != nil {
			t.Fatalf("Submit(%q): %v", in, err)
		}
	}
	if sk.Lines().Count() != 1 {
		t.Fatalf("got %d lines, want 1", sk.Lines().Count())
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have auto-committed and deactivated")
	}
	l := sk.Lines().Item(0)
	if l.StartPoint().Position() != math.P2(0, 0) || l.EndPoint().Position() != math.P2(10, 0) {
		t.Errorf("line endpoints = %v..%v, want (0,0)..(10,0)", l.StartPoint().Position(), l.EndPoint().Position())
	}
}

func TestCommandLineInlineArguments(t *testing.T) {
	s, sk, cl := sketchCommandSession(t)
	if err := cl.Submit(s, "L 2,2 8,8"); err != nil {
		t.Fatalf("Submit inline: %v", err)
	}
	if sk.Lines().Count() != 1 || s.ActiveTool() != nil {
		t.Fatalf("inline line: lines=%d activeTool=%v", sk.Lines().Count(), s.ActiveTool() != nil)
	}
}

func TestCommandLineRelativeCoordinate(t *testing.T) {
	s, sk, cl := sketchCommandSession(t)
	for _, in := range []string{"LINE", "5,5", "@10,0"} {
		if err := cl.Submit(s, in); err != nil {
			t.Fatalf("Submit(%q): %v", in, err)
		}
	}
	l := sk.Lines().Item(0)
	if l.EndPoint().Position() != math.P2(15, 5) {
		t.Errorf("relative endpoint = %v, want (15,5)", l.EndPoint().Position())
	}
}

func TestCommandLineUnknownCommand(t *testing.T) {
	s, _, cl := sketchCommandSession(t)
	if err := cl.Submit(s, "FLERP"); err == nil {
		t.Fatal("unknown command should error")
	}
	lines := cl.Scrollback().Lines()
	last := lines[len(lines)-1]
	if last.Severity != cmdline.Error {
		t.Errorf("last scrollback line severity = %v, want Error", last.Severity)
	}
}

func TestCommandLineRepeatLastOnEmpty(t *testing.T) {
	s, _, cl := sketchCommandSession(t)
	if err := cl.Submit(s, "LINE"); err != nil {
		t.Fatalf("Submit LINE: %v", err)
	}
	if err := cl.Submit(s, ""); err != nil { // empty while a tool is active → finish (cancel, 0 pts)
		t.Fatalf("Submit empty (finish): %v", err)
	}
	if s.ActiveTool() != nil {
		t.Fatal("empty submit should have finished the line tool")
	}
	if err := cl.Submit(s, ""); err != nil { // empty while idle → repeat last command
		t.Fatalf("Submit empty (repeat): %v", err)
	}
	if s.ActiveTool() == nil || s.ActiveTool().Name() != "Line" {
		t.Errorf("empty idle submit should repeat LINE, active tool = %v", s.ActiveTool())
	}
}

func TestCommandLineDrawsCircleFromText(t *testing.T) {
	s, sk, cl := sketchCommandSession(t)
	for _, in := range []string{"CIRCLE", "0,0", "5,0"} {
		if err := cl.Submit(s, in); err != nil {
			t.Fatalf("Submit(%q): %v", in, err)
		}
	}
	if sk.Circles().Count() != 1 || s.ActiveTool() != nil {
		t.Fatalf("circle: count=%d activeTool=%v", sk.Circles().Count(), s.ActiveTool() != nil)
	}
}

func TestCommandLineDrawsRectangleFromText(t *testing.T) {
	s, sk, cl := sketchCommandSession(t)
	for _, in := range []string{"REC", "0,0", "10,5"} {
		if err := cl.Submit(s, in); err != nil {
			t.Fatalf("Submit(%q): %v", in, err)
		}
	}
	if sk.Lines().Count() != 4 || s.ActiveTool() != nil {
		t.Fatalf("rectangle: lines=%d activeTool=%v", sk.Lines().Count(), s.ActiveTool() != nil)
	}
}

func TestCommandLineExtrudeFromValue(t *testing.T) {
	s, profile := newPartWithSquare(t, 4)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	s.SetPicker(stubPicker{sel: profile})
	cl := s.CommandLine()

	if err := cl.Submit(s, "EXTRUDE"); err != nil {
		t.Fatalf("Submit EXTRUDE: %v", err)
	}
	ext, ok := s.ActiveTool().Tool().(*ExtrudeTool)
	if !ok {
		t.Fatalf("active tool = %T, want *ExtrudeTool", s.ActiveTool().Tool())
	}
	s.Click(120, 90) // pick the region in the viewport
	if err := cl.Submit(s, "5"); err != nil {
		t.Fatalf("Submit height: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Error("extrude should have committed after the height value")
	}
	if ext.AddedFeature() == nil {
		t.Fatal("no extrude feature was created")
	}
}
