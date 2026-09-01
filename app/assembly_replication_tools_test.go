// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestCopyComponentsAddsIndependentCopy: with one component selected, Copy adds a second
// occurrence (same component, "-copy" suffix) and the add is one undo step (#765).
func TestCopyComponentsAddsIndependentCopy(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	s.selection.Add(OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})
	trackFromHere(s)

	if err := s.CopyComponents(); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 2 {
		t.Fatalf("after Copy: occurrence count = %d, want 2", got)
	}
	if name := asm.Occurrences().Item(1).Name(); name != "widget:1-copy" {
		t.Errorf("copy name = %q, want widget:1-copy", name)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 1 {
		t.Errorf("undo should remove the copy: count = %d, want 1", got)
	}
}

// TestCopyComponentsRequiresSelection: Copy with nothing selected errors rather than no-op.
func TestCopyComponentsRequiresSelection(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	if err := s.CopyComponents(); err == nil {
		t.Error("Copy with no occurrence selected should error")
	}
}

// TestRectPatternPlacesGrid: a 2×2 rectangular pattern of the seed adds 3 occurrences (the seed
// is element 0,0), for 4 total, in one undo step.
func TestRectPatternPlacesGrid(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	trackFromHere(s)

	tool := NewAssemblyRectPatternTool()
	tool.Pick(s, OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})
	tool.count1, tool.count2 = 2, 2
	tool.spacing1, tool.spacing2 = 3, 3
	if !tool.CanCommit() {
		t.Fatal("a 2×2 pattern with a seed should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 4 {
		t.Fatalf("2×2 pattern: occurrence count = %d, want 4", got)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 1 {
		t.Errorf("one undo should remove all pattern copies: count = %d, want 1", got)
	}
}

// TestCircPatternPlacesRing: a count-4 circular pattern adds 3 occurrences (the seed is element
// 0), for 4 total.
func TestCircPatternPlacesRing(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")

	tool := NewAssemblyCircPatternTool() // count 4 default
	tool.Pick(s, OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})
	if !tool.CanCommit() {
		t.Fatal("a 4-up ring with a seed should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 4 {
		t.Errorf("4-up ring: occurrence count = %d, want 4", got)
	}
}

// TestMirrorComponentsAddsMirror: Mirror adds one mirrored occurrence (the "-mirror" suffix) per
// source.
func TestMirrorComponentsAddsMirror(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")

	tool := NewAssemblyMirrorTool() // normal +X
	tool.Pick(s, OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})
	if !tool.CanCommit() {
		t.Fatal("a mirror with a source and a nonzero normal should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 2 {
		t.Fatalf("mirror: occurrence count = %d, want 2", got)
	}
	if name := asm.Occurrences().Item(1).Name(); name != "widget:1-mirror" {
		t.Errorf("mirror name = %q, want widget:1-mirror", name)
	}
}

// TestMirrorRejectsZeroNormal: a degenerate (zero) mirror-plane normal is not committable and
// errors on commit rather than placing a bogus reflection.
func TestMirrorRejectsZeroNormal(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	tool := NewAssemblyMirrorTool()
	tool.Pick(s, OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})
	tool.normalX, tool.normalY, tool.normalZ = 0, 0, 0
	if tool.CanCommit() {
		t.Error("a zero normal should block commit")
	}
	if err := tool.Commit(s); err == nil {
		t.Error("commit with a zero normal should error")
	}
}

// TestReplicationToolSeedsFromSelection: starting a tool with a component pre-selected adopts it
// as the source, so "select then Pattern" needs no re-pick.
func TestReplicationToolSeedsFromSelection(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	s.selection.Add(OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})

	tool := NewAssemblyRectPatternTool()
	tool.Start(s)
	if len(tool.sources) != 1 {
		t.Fatalf("Start adopted %d sources from the selection, want 1", len(tool.sources))
	}
}

// TestReplicationToolsRequireSeed: a pattern with no source errors on commit (the active-assembly
// + non-empty-source gate).
func TestReplicationToolsRequireSeed(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	tool := NewAssemblyCircPatternTool() // no Pick, no selection
	if err := tool.Commit(s); err == nil {
		t.Error("a pattern with no selected component should error on commit")
	}
}
