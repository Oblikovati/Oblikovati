// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// assemblyWithComponent returns a session whose active document is an empty assembly, plus a
// component part open in the same workspace ready to place. The component is registered (open),
// so a restore's re-bind resolves it through the reference graph with no disk round trip — the
// fixture the assembly-undo tests build on (#763).
func assemblyWithComponent(t *testing.T) (*Session, *compdef.AssemblyComponentDefinition) {
	t.Helper()
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "widget.obk", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	asm, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asm); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	return s, asm.Content().(*compdef.AssemblyComponentDefinition)
}

// placedWidget places the workspace's widget component into the active assembly under name,
// failing the test on error. It opens the component by name through the assembly's reference
// graph so the placement persists like a real Place would.
func placedWidget(t *testing.T, s *Session, asm *compdef.AssemblyComponentDefinition, name string) {
	t.Helper()
	owner := s.ActiveDocument()
	widget, ok := owner.OpenReference("widget.obk")
	if !ok {
		t.Fatal("widget.obk should resolve as a reference of the active assembly")
	}
	if _, err := asm.PlaceComponentFromFile(owner, widget, name, math.Identity4()); err != nil {
		t.Fatalf("place %q: %v", name, err)
	}
}

// TestAssemblyPlaceIsUndoable is the headline assembly-undo test (#763): placing a component is
// one transaction event, navigated through the same generalized recipe-store stream a part edit
// uses. Undo removes the occurrence (restore the prior recipe, then re-bind), redo restores it.
func TestAssemblyPlaceIsUndoable(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	trackFromHere(s) // baseline = empty assembly, the state undo returns to

	placedWidget(t, s, asm, "widget:1")
	s.recordEdit(asm, "Place Component")
	if got := asm.Occurrences().Count(); got != 1 {
		t.Fatalf("after place: occurrence count = %d, want 1", got)
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 0 {
		t.Errorf("after undo: occurrence count = %d, want 0", got)
	}

	if err := s.Redo(); err != nil {
		t.Fatalf("redo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 1 {
		t.Fatalf("after redo: occurrence count = %d, want 1", got)
	}
	if name := asm.Occurrences().Item(0).Name(); name != "widget:1" {
		t.Errorf("redo re-bound occurrence name = %q, want %q", name, "widget:1")
	}
}

// TestAssemblyEditUndoIsOneStep checks two placements coalesced under a bounded transaction
// undo as a single step — the assembly path through Begin/EndTransaction now that the
// transaction guards accept any recipe store, not only parts (#763).
func TestAssemblyEditUndoIsOneStep(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	trackFromHere(s)

	if err := s.BeginTransaction("Place Two"); err != nil {
		t.Fatalf("begin: %v", err)
	}
	placedWidget(t, s, asm, "widget:1")
	s.recordEdit(asm, "Place Component")
	placedWidget(t, s, asm, "widget:2")
	s.recordEdit(asm, "Place Component")
	if err := s.EndTransaction(); err != nil {
		t.Fatalf("end: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 2 {
		t.Fatalf("after grouped place: occurrence count = %d, want 2", got)
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 0 {
		t.Errorf("one undo should remove the whole group: occurrence count = %d, want 0", got)
	}
}
