// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// TestCreateSketch3DEnterAddLineFinish drives the full 3D-sketch UI flow end to end: the
// "3D Sketch" ribbon command enters the environment, the 3D Line tool builds a polyline,
// and "Finish 3D Sketch" returns to the model with the geometry committed.
func TestCreateSketch3DEnterAddLineFinish(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := s.Execute("Sketch.Create3D"); err != nil {
		t.Fatalf("3D Sketch: %v", err)
	}
	if !s.InSketch3D() || def.Sketches3D().Count() != 1 {
		t.Fatalf("3D Sketch command did not enter a 3D sketch environment")
	}

	if err := s.Execute("Sketch3D.Line"); err != nil {
		t.Fatalf("3D Line: %v", err)
	}
	tool, ok := s.ActiveTool().Tool().(*Line3DTool)
	if !ok {
		t.Fatalf("active tool = %T, want *Line3DTool", s.ActiveTool().Tool())
	}
	tool.AddPoint(math.P3(0, 0, 0))
	tool.AddPoint(math.P3(1, 0, 0))
	tool.AddPoint(math.P3(1, 1, 2))
	if !tool.CanCommit() {
		t.Fatal("line tool should be ready to commit after 3 points")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("commit line: %v", err)
	}

	sk := def.Sketches3D().Item(0)
	if sk.EntityCount() != 2 {
		t.Fatalf("3D sketch has %d entities, want 2 lines", sk.EntityCount())
	}

	if err := s.Execute("Sketch3D.Finish"); err != nil {
		t.Fatalf("Finish 3D Sketch: %v", err)
	}
	if s.InSketch3D() {
		t.Error("Finish should leave the 3D-sketch environment")
	}
}

// TestSketch3DPointToolEndToEnd drives the 3D Point tool to place a standalone point.
func TestSketch3DPointToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := s.Execute("Sketch.Create3D"); err != nil {
		t.Fatalf("3D Sketch: %v", err)
	}
	if err := s.Execute("Sketch3D.Point"); err != nil {
		t.Fatalf("3D Point: %v", err)
	}
	pt := s.ActiveTool().Tool().(*Point3DTool)
	pt.SetPoint(math.P3(3, 4, 5))
	if err := s.OK(); err != nil {
		t.Fatalf("commit point: %v", err)
	}
	sk := def.Sketches3D().Item(0)
	if sk.EntityCount() != 1 {
		t.Fatalf("3D sketch has %d entities, want 1 point", sk.EntityCount())
	}
}

// TestCanCreateSketch3DPredicate checks the enable predicate gates on an active part and
// no open sketch edit.
func TestCanCreateSketch3DPredicate(t *testing.T) {
	s, _ := emptyPartSession(t)
	if !s.CanCreateSketch3D() {
		t.Fatal("with an active part and no edit, 3D Sketch should be enabled")
	}
	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	if s.CanCreateSketch3D() {
		t.Error("while editing a 3D sketch, 3D Sketch should be disabled")
	}
	// A second create is rejected while one is open.
	if _, err := s.CreateSketch3D(); err == nil {
		t.Error("nesting a 3D sketch edit should error")
	}
}

// TestSketch3DCircleArcHelixTools drives the circle, arc and helix tools end to end.
func TestSketch3DCircleArcHelixTools(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := s.Execute("Sketch.Create3D"); err != nil {
		t.Fatalf("3D Sketch: %v", err)
	}

	// 3D Circle.
	if err := s.Execute("Sketch3D.Circle"); err != nil {
		t.Fatalf("3D Circle: %v", err)
	}
	circ := s.ActiveTool().Tool().(*Circle3DTool)
	circ.SetCenter(math.P3(0, 0, 0))
	circ.SetRadius(5)
	if err := s.OK(); err != nil {
		t.Fatalf("commit circle: %v", err)
	}

	// 3D Arc.
	if err := s.Execute("Sketch3D.Arc"); err != nil {
		t.Fatalf("3D Arc: %v", err)
	}
	arc := s.ActiveTool().Tool().(*Arc3DTool)
	arc.AddPoint(math.P3(0, 0, 0))
	arc.AddPoint(math.P3(1, 0, 0))
	arc.AddPoint(math.P3(0, 1, 0))
	if err := s.OK(); err != nil {
		t.Fatalf("commit arc: %v", err)
	}

	// Helical curve.
	if err := s.Execute("Sketch3D.Helix"); err != nil {
		t.Fatalf("Helix: %v", err)
	}
	hx := s.ActiveTool().Tool().(*Helix3DTool)
	hx.SetOrigin(math.P3(0, 0, 0))
	hx.SetRadius(2)
	hx.SetPitch(1)
	hx.SetTurns(5)
	if err := s.OK(); err != nil {
		t.Fatalf("commit helix: %v", err)
	}

	sk := def.Sketches3D().Item(0)
	if sk.EntityCount() != 3 {
		t.Fatalf("3D sketch has %d entities, want 3 (circle/arc/helix)", sk.EntityCount())
	}
}

// TestSketch3DToolCommitGuards checks each tool refuses to commit before it has input.
func TestSketch3DToolCommitGuards(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	circ, arc, hx := NewCircle3DTool(), NewArc3DTool(), NewHelix3DTool()
	if circ.CanCommit() || arc.CanCommit() || hx.CanCommit() {
		t.Error("tools should not be committable before input")
	}
	for _, tl := range []Tool{circ, arc, hx} {
		if tl.Name() == "" {
			t.Errorf("%T has no name", tl)
		}
		tl.Start(s)
		tl.Pick(s, nil)
		tl.Cancel(s)
		if err := tl.Commit(s); err == nil {
			t.Errorf("%T should refuse to commit with no input", tl)
		}
	}
	// A degenerate axis is rejected by the circle/helix tools.
	circ.SetCenter(math.P3(0, 0, 0))
	circ.SetRadius(3)
	circ.SetAxis(math.V3(0, 0, 0))
	if err := circ.Commit(s); err == nil {
		t.Error("a zero circle axis should be rejected")
	}
}

// TestSketch3DToolInterfaceMethods exercises the trivial Tool surface (Name/Start/Pick/
// Cancel) and the not-ready-to-commit guards.
func TestSketch3DToolInterfaceMethods(t *testing.T) {
	s, _ := emptyPartSession(t)
	line, point := NewLine3DTool(), NewPoint3DTool()
	for _, tl := range []Tool{line, point} {
		if tl.Name() == "" {
			t.Errorf("%T has no name", tl)
		}
		tl.Start(s)
		tl.Pick(s, nil)
		tl.Cancel(s)
		if tl.CanCommit() {
			t.Errorf("%T should not be committable before input", tl)
		}
	}
	// A point tool with no position, and a line tool with one point, both fail to commit.
	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	if err := point.Commit(s); err == nil {
		t.Error("committing a point tool with no position should error")
	}
	line.AddPoint(math.P3(0, 0, 0))
	if err := line.Commit(s); err == nil {
		t.Error("committing a line tool with one point should error")
	}
}

// TestFinishSketch3DNotEditing checks Finish errors when no 3D sketch is open.
func TestFinishSketch3DNotEditing(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := s.FinishSketch3D(); err == nil {
		t.Error("Finish 3D Sketch with nothing open should error")
	}
}

// TestFinishSketch3DCancelsActiveTool checks finishing abandons an in-progress tool.
func TestFinishSketch3DCancelsActiveTool(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	s.StartTool(NewLine3DTool())
	if err := s.FinishSketch3D(); err != nil {
		t.Fatalf("FinishSketch3D: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Error("finishing should cancel the in-progress 3D line tool")
	}
}

// TestLine3DToolNotInSketchErrors checks committing the line tool outside a 3D sketch fails.
func TestLine3DToolNotInSketchErrors(t *testing.T) {
	s, _ := emptyPartSession(t)
	tool := NewLine3DTool()
	tool.AddPoint(math.P3(0, 0, 0))
	tool.AddPoint(math.P3(1, 0, 0))
	if err := tool.Commit(s); err == nil {
		t.Error("committing a 3D line with no active 3D sketch should error")
	}
}
