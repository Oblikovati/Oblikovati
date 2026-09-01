// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// sketchWithPerpConstraint sets up a part in an open sketch with two lines and a committed
// Perpendicular constraint (its own undo step), leaving the sketch active. It is the shared
// fixture for the keyboard-undo regressions below: an undoable step exists and the session
// is in the exact state a user is in mid-edit.
func sketchWithPerpConstraint(t *testing.T) *Session {
	t.Helper()
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 4))
	s.RecordActiveEdit("Sketch Geometry") // baseline: the two lines, before the constraint

	s.StartTool(constraintToolDefs[3].new()) // Perpendicular
	s.feedPick(SketchEntityHandle{Entity: sk.Lines().Item(0)})
	s.feedPick(SketchEntityHandle{Entity: sk.Lines().Item(1)}) // auto-commits via OK → records "Perpendicular"
	if !s.CanUndo() || s.UndoLabel() != "Perpendicular" {
		t.Fatalf("fixture: CanUndo=%v UndoLabel=%q; want true/Perpendicular", s.CanUndo(), s.UndoLabel())
	}
	return s
}

// TestKeyboardUndoWorksWhileToolArmed is the #1750 regression: Ctrl+Z routed through the real
// keyboard path (PressKey → RunChord → dispatchUndo) must undo the last committed step even when
// an interactive tool is armed — the state a user is in for the ENTIRE duration of sketch drawing.
// The old guard `s.tool != nil` silently no-op'd here, so undo appeared dead during sketching; the
// fix narrows the guard to an actually-open transaction (s.InTransaction()).
//
// The armed tool was the LINE tool until the line chain became a transactional tool
// (tool_transaction.go) — it now deliberately DOES open a group on activation, so it can no
// longer stand for "merely armed". The rectangle tool is the same shape of sketch drawing tool
// and does not opt in, so it still pins the exact property this test was written for: arming a
// tool is not, by itself, a transaction. TestKeyboardUndoBlockedWhileLineChainOpen below covers
// the tool that now is one.
func TestKeyboardUndoWorksWhileToolArmed(t *testing.T) {
	t.Parallel()
	s := sketchWithPerpConstraint(t)
	s.StartTool(NewRectangleTool()) // arm a tool: this is precisely the old guard's kill condition
	if s.InTransaction() {
		t.Fatal("fixture: no bounded transaction should be open with a merely-armed tool")
	}

	if err := s.PressKey(KeyEvent{Key: "z", Mods: CtrlMod}); err != nil { // Ctrl+Z
		t.Fatalf("PressKey(Ctrl+Z): %v", err)
	}

	if n := len(s.ActiveSketch().Constraints()); n != 0 {
		t.Errorf("after Ctrl+Z with a tool armed, sketch has %d constraints; want 0 (undo must fire)", n)
	}
	if !s.InSketch() {
		t.Error("undo of the constraint should keep the sketch open for editing")
	}
	if !s.CanRedo() {
		t.Error("after undo, the step should be redoable")
	}
}

// TestKeyboardUndoBlockedDuringOpenTransaction pins the invariant the guard exists to protect:
// while a bounded transaction (a recipe group) is genuinely open, Ctrl+Z must NOT fire — undoing
// mid-record would corrupt the unit being written. This is the narrow condition the #1750 fix
// keeps blocking (the old guard's over-broad `s.tool != nil` conflated this with "any tool armed").
func TestKeyboardUndoBlockedDuringOpenTransaction(t *testing.T) {
	t.Parallel()
	s := sketchWithPerpConstraint(t)
	if err := s.BeginTransaction("group edit"); err != nil {
		t.Fatalf("BeginTransaction: %v", err)
	}
	if !s.InTransaction() {
		t.Fatal("fixture: BeginTransaction should open a bounded transaction")
	}

	if err := s.PressKey(KeyEvent{Key: "z", Mods: CtrlMod}); err != nil { // Ctrl+Z — must be a no-op
		t.Fatalf("PressKey(Ctrl+Z): %v", err)
	}

	if n := len(s.ActiveSketch().Constraints()); n != 1 {
		t.Errorf("Ctrl+Z during an open transaction changed the sketch (%d constraints); want it blocked at 1", n)
	}
	if !s.CanUndo() || s.UndoLabel() != "Perpendicular" {
		t.Errorf("the committed step must survive a blocked undo; CanUndo=%v label=%q", s.CanUndo(), s.UndoLabel())
	}
	if err := s.EndTransaction(); err != nil {
		t.Fatalf("EndTransaction: %v", err)
	}
}
