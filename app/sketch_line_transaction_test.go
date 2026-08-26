// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #1750 GUI trigger: the in-transaction Undo/Redo guard had no interactive path to fire on —
// BeginTransaction/EndTransaction were called only by the add-in wire router, so a smoke test
// found QAT Undo mid-sketch-line undoing normally instead of being blocked. The line chain now
// holds a bounded transaction for its whole session (tool_transaction.go).
//
// The tests below split into two halves, and BOTH matter:
//   - the guard is live across the session (the feature), and
//   - every exit path closes the group (the invariant) — a leaked group pins InTransaction()
//     true forever and kills Undo/Redo for the rest of the document session, which is strictly
//     worse than the missing guard.

// lineChainSession is a part with an open sketch holding one committed, undoable step. The
// committed step is what makes the guard observable: with an empty stack Undo is a no-op for
// the boring reason (nothing to undo) and would pass these tests for the wrong reason.
func lineChainSession(t *testing.T) *Session {
	t.Helper()
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane()) // CreateSketch establishes the undo baseline
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	s.RecordActiveEdit("Sketch Geometry")
	if !s.CanUndo() {
		t.Fatal("fixture: expected an undoable step")
	}
	return s
}

// TestLineToolOpensTransactionOnActivation is smoke-test acceptance sub-case (b): the tool is
// armed and NOT yet edited, and the guard must already hold. Opening the group only once a
// point exists would leave this case failing exactly as it did before the fix.
func TestLineToolOpensTransactionOnActivation(t *testing.T) {
	s := lineChainSession(t)
	s.StartTool(NewLineTool())

	if !s.InTransaction() {
		t.Fatal("no transaction open with the line tool merely armed — sub-case (b) guard is dead")
	}
	if !s.CanUndo() {
		t.Error("arming the tool must not empty the undo stack; the guard, not the grey-out, must block")
	}
}

// TestUndoBlockedWhileLineChainOpen is acceptance sub-case (a): mid-chain, with an undoable step
// behind the cursor, Undo must do nothing. It drives the real keyboard path AND the pure QAT
// decision function's inputs, since both read InTransaction().
func TestUndoBlockedWhileLineChainOpen(t *testing.T) {
	s := lineChainSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40) // mid-chain: points placed, chain not finished

	if err := s.PressKey(KeyEvent{Key: "z", Mods: CtrlMod}); err != nil { // Ctrl+Z — must no-op
		t.Fatalf("PressKey(Ctrl+Z): %v", err)
	}

	if n := s.ActiveSketch().Lines().Count(); n != 1 {
		t.Errorf("undo fired mid-chain: sketch has %d lines, want the 1 committed by the fixture", n)
	}
	if !s.CanUndo() || s.UndoLabel() != "Sketch Geometry" {
		t.Errorf("the committed step must survive a blocked undo; CanUndo=%v label=%q",
			s.CanUndo(), s.UndoLabel())
	}
}

// TestRedoBlockedWhileLineChainOpen is sub-case (b) end to end: undo a step so the redo branch
// is populated, arm the line tool WITHOUT editing, and Redo must not fire. Arming must not
// truncate the forward branch either — a recorded edit would (undo.go TruncateTo).
func TestRedoBlockedWhileLineChainOpen(t *testing.T) {
	s := lineChainSession(t)
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if !s.CanRedo() {
		t.Fatal("fixture: expected a redoable step")
	}

	s.StartTool(NewLineTool()) // armed, no click

	if err := s.PressKey(KeyEvent{Key: "y", Mods: CtrlMod}); err != nil { // Ctrl+Y — must no-op
		t.Fatalf("PressKey(Ctrl+Y): %v", err)
	}
	if n := s.ActiveSketch().Lines().Count(); n != 0 {
		t.Errorf("redo fired with the tool armed: sketch has %d lines, want 0", n)
	}
	if !s.CanRedo() {
		t.Error("arming the tool truncated the redo branch; it must leave the stream untouched")
	}
}

// TestFinishedLineChainCommitsAndReleases is the commit path: finishing normally closes the
// group, records the chain as ONE step still labelled "Line" (the granularity and label the
// tool produced before the wrap), and hands Undo/Redo back.
func TestFinishedLineChainCommitsAndReleases(t *testing.T) {
	s := lineChainSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40)
	s.Click(80, 80)
	_ = s.PressKey(KeyEvent{Key: "Enter"})

	if s.InTransaction() || s.ToolHoldsTransaction() {
		t.Fatal("finishing the chain left the transaction open")
	}
	if got := s.UndoLabel(); got != "Line" {
		t.Errorf("undo label %q, want %q — the wrap must not rename the step", got, "Line")
	}
	before := s.ActiveSketch().Lines().Count()
	if before != 3 { // the fixture's 1 + the chain's 2
		t.Fatalf("chain drew %d lines total, want 3", before)
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo after finish: %v", err)
	}
	if got := s.ActiveSketch().Lines().Count(); got != 1 {
		t.Errorf("one undo left %d lines, want 1 — the chain must be a single undo step", got)
	}
	if err := s.Redo(); err != nil {
		t.Fatalf("Redo after finish: %v", err)
	}
	if got := s.ActiveSketch().Lines().Count(); got != 3 {
		t.Errorf("redo restored %d lines, want 3", got)
	}
}

// TestEscapeMidChainCommitsAndReleases pins that the wrap did not change Escape's meaning: with
// segments placed, Escape FINISHES the chain and keeps them (#2024) — so it takes the commit
// path, and the group must close there too.
func TestEscapeMidChainCommitsAndReleases(t *testing.T) {
	s := lineChainSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40)
	_ = s.PressKey(KeyEvent{Key: "Escape"})

	if s.InTransaction() || s.ToolHoldsTransaction() {
		t.Fatal("Escape mid-chain left the transaction open")
	}
	if got := s.ActiveSketch().Lines().Count(); got != 2 {
		t.Errorf("Escape kept %d lines, want 2 (fixture's 1 + the placed segment)", got)
	}
	if !s.CanUndo() {
		t.Error("Undo must work immediately after Escape")
	}
}

// TestCancelledLineChainReleasesWithoutRecording is the critical regression: a session cancelled
// before it can commit anything must close its group AND add no undo entry. One click places no
// segment, so Escape abandons rather than finishing (LineTool.FinishesOnCancel needs two points).
func TestCancelledLineChainReleasesWithoutRecording(t *testing.T) {
	s := lineChainSession(t)
	labelBefore := s.UndoLabel()
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	_ = s.PressKey(KeyEvent{Key: "Escape"}) // abandons: nothing placed yet

	if s.InTransaction() || s.ToolHoldsTransaction() {
		t.Fatal("cancelling the tool leaked an open transaction — Undo/Redo are now dead for the session")
	}
	if s.ActiveTool() != nil {
		t.Error("Escape should end the tool")
	}
	if got := s.UndoLabel(); got != labelBefore {
		t.Errorf("a cancelled session recorded an undo step (%q, was %q); it must record nothing",
			got, labelBefore)
	}
	if got := s.ActiveSketch().Lines().Count(); got != 1 {
		t.Errorf("cancelling changed the model: %d lines, want the fixture's 1", got)
	}

	// The whole point of closing the group: Undo works again immediately afterwards.
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo after a cancelled chain: %v", err)
	}
	if got := s.ActiveSketch().Lines().Count(); got != 0 {
		t.Errorf("undo after cancel left %d lines, want 0", got)
	}
}

// TestSwitchingToolMidChainReleases covers the exit path with no explicit cancel gesture: the
// user picks a different command while the chain is open. StartTool cancels the outgoing tool,
// and its group must die with it rather than be inherited by the incoming one.
func TestSwitchingToolMidChainReleases(t *testing.T) {
	s := lineChainSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	if !s.InTransaction() {
		t.Fatal("fixture: the chain should hold a transaction")
	}

	s.StartTool(NewRectangleTool()) // a different, non-transactional command

	if s.InTransaction() || s.ToolHoldsTransaction() {
		t.Fatal("switching tools inherited the line chain's transaction — it must close with its tool")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo after a tool switch: %v", err)
	}
}

// TestRestartingLineToolDoesNotNestTransactions is the leak that a naive begin-per-activation
// would cause: groups nest by depth (undo.go groupDepth), so re-arming the line tool five times
// without closing would need five Ends. One close must always be enough.
func TestRestartingLineToolDoesNotNestTransactions(t *testing.T) {
	s := lineChainSession(t)
	for range 5 {
		s.StartTool(NewLineTool())
		s.Click(40, 40)
	}
	s.CancelTool()

	if s.InTransaction() || s.ToolHoldsTransaction() {
		t.Fatal("re-arming the line tool nested transactions; one cancel left a group open")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo after repeated re-arming: %v", err)
	}
}

// TestClosingDocumentMidChainReleases is the exit path with no user gesture at all. The tool's
// group is tracked by document id, so closing must not leave the flag set (which would make the
// NEXT document's first teardown close a group it never opened).
func TestClosingDocumentMidChainReleases(t *testing.T) {
	s := lineChainSession(t)
	d := s.ActiveDocument()
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	if !s.ToolHoldsTransaction() {
		t.Fatal("fixture: the chain should hold a transaction")
	}

	if err := s.CloseDocument(d, true); err != nil {
		t.Fatalf("CloseDocument: %v", err)
	}

	if s.ToolHoldsTransaction() {
		t.Fatal("closing the document mid-chain left the tool transaction flagged open")
	}
	if s.InTransaction() {
		t.Fatal("a transaction is reported open after the document holding it closed")
	}
	if s.ActiveTool() != nil {
		t.Error("closing the document should abandon the armed tool (#2040)")
	}
}

// TestNonTransactionalSketchToolIsUnchanged is the blast-radius check: the wrap is opt-in, so a
// sketch tool that did not opt in must behave exactly as before — no group, undo live throughout.
func TestNonTransactionalSketchToolIsUnchanged(t *testing.T) {
	s := lineChainSession(t)
	s.StartTool(NewRectangleTool())

	if s.InTransaction() || s.ToolHoldsTransaction() {
		t.Fatal("a tool that did not opt in opened a transaction")
	}
	s.Click(40, 40)
	if s.InTransaction() {
		t.Fatal("a non-transactional tool opened a transaction on its first click")
	}
	if err := s.Undo(); err != nil { // undo stays live for tools outside the opt-in (#1750)
		t.Fatalf("Undo with a non-transactional tool armed: %v", err)
	}
}
