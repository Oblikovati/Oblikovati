// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"
)

// findBrowserNode returns the first node of the given kind whose label matches, searched
// depth-first — so a test can locate an occurrence row in the assembly tree.
func findBrowserNode(n BrowserNode, kind, label string) *BrowserNode {
	if n.Kind == kind && n.Label == label {
		return &n
	}
	for i := range n.Children {
		if found := findBrowserNode(n.Children[i], kind, label); found != nil {
			return found
		}
	}
	return nil
}

// TestAssemblyBrowserListsOccurrences: an assembly with a placed component shows that occurrence
// as a selectable browser row carrying an OccurrenceHandle, so clicking it selects the occurrence
// (#764). A part document's browser never grew an occurrence row, so this is the assembly path.
func TestAssemblyBrowserListsOccurrences(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")

	node := findBrowserNode(BuildBrowser(s), "occurrence", "widget:1")
	if node == nil {
		t.Fatal("assembly browser has no occurrence row for widget:1")
	}
	h, ok := node.Select.(OccurrenceHandle)
	if !ok {
		t.Fatalf("occurrence node selects %T, want OccurrenceHandle", node.Select)
	}
	if h.Occurrence != asm.Occurrences().Item(0) {
		t.Error("occurrence handle does not point at the placed occurrence")
	}
}

// TestOccurrenceLabelReflectsState: the browser row annotates grounded and suppressed state in
// its label so the text tree is self-describing.
func TestOccurrenceLabelReflectsState(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	o := asm.Occurrences().Item(0)

	if got := occurrenceLabel(o); got != "widget:1" {
		t.Errorf("plain label = %q, want widget:1", got)
	}
	o.SetGrounded(true)
	o.SetSuppressed(true)
	if got := occurrenceLabel(o); got != "widget:1 (grounded) (suppressed)" {
		t.Errorf("annotated label = %q, want \"widget:1 (grounded) (suppressed)\"", got)
	}
}

// TestOccurrenceMenuTogglesLabels: the right-click menu offers Ground/Suppress/Delete, and the
// toggle labels reflect the occurrence's current state (Ground→Unground, Suppress→Unsuppress).
func TestOccurrenceMenuTogglesLabels(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	o := asm.Occurrences().Item(0)
	node := findBrowserNode(BuildBrowser(s), "occurrence", "widget:1")

	if labels := strings.Join(menuLabels(BrowserMenu(*node)), "|"); labels != "Ground|Suppress|Delete" {
		t.Errorf("fresh occurrence menu = %q, want Ground|Suppress|Delete", labels)
	}
	o.SetGrounded(true)
	o.SetSuppressed(true)
	if labels := strings.Join(menuLabels(BrowserMenu(*node)), "|"); labels != "Unground|Unsuppress|Delete" {
		t.Errorf("grounded+suppressed menu = %q, want Unground|Unsuppress|Delete", labels)
	}
}

// TestSuppressOccurrenceIsUndoable: suppressing a component is one undo step — undo unsuppresses,
// redo re-suppresses — proving the change records a recipe delta (#764).
func TestSuppressOccurrenceIsUndoable(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	trackFromHere(s) // baseline: placed, unsuppressed

	if err := s.ToggleOccurrenceSuppressed(asm.Occurrences().Item(0)); err != nil {
		t.Fatalf("suppress: %v", err)
	}
	if !asm.Occurrences().Item(0).Suppressed() {
		t.Fatal("toggle did not suppress the occurrence")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if asm.Occurrences().Item(0).Suppressed() {
		t.Error("undo should restore the unsuppressed state")
	}
	if err := s.Redo(); err != nil {
		t.Fatalf("redo: %v", err)
	}
	if !asm.Occurrences().Item(0).Suppressed() {
		t.Error("redo should re-suppress the occurrence")
	}
}

// TestOccurrenceToggleToCurrentStateRecordsNothing: setting an occurrence to the state it is
// already in is a no-op — it must not push an undo step (#764).
func TestOccurrenceToggleToCurrentStateRecordsNothing(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	o := asm.Occurrences().Item(0)
	trackFromHere(s) // baseline: unsuppressed, not grounded

	if err := s.SuppressOccurrence(o, false); err != nil {
		t.Fatalf("suppress to current state: %v", err)
	}
	if err := s.GroundOccurrence(o, false); err != nil {
		t.Fatalf("ground to current state: %v", err)
	}
	if s.CanUndo() {
		t.Error("toggling to the current state should record no undo step")
	}
}

// TestGroundOccurrenceIsUndoable: grounding a component is one undo step (the grounded flag
// round-trips through the recipe).
func TestGroundOccurrenceIsUndoable(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	trackFromHere(s) // baseline: placed, not grounded

	if err := s.ToggleOccurrenceGrounded(asm.Occurrences().Item(0)); err != nil {
		t.Fatalf("ground: %v", err)
	}
	if !asm.Occurrences().Item(0).Grounded() {
		t.Fatal("toggle did not ground the occurrence")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if asm.Occurrences().Item(0).Grounded() {
		t.Error("undo should release the occurrence")
	}
}

// TestDeleteOccurrenceIsUndoable: deleting a component removes it and clears the selection; undo
// restores it through the assembly recipe (#764).
func TestDeleteOccurrenceIsUndoable(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	s.selection.Add(OccurrenceHandle{Occurrence: asm.Occurrences().Item(0)})
	trackFromHere(s) // baseline: one occurrence placed

	if err := s.DeleteOccurrence(asm.Occurrences().Item(0)); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 0 {
		t.Fatalf("after delete: occurrence count = %d, want 0", got)
	}
	if s.Selection().Count() != 0 {
		t.Error("delete should clear the selection (the node is gone)")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 1 {
		t.Errorf("undo should restore the occurrence: count = %d, want 1", got)
	}
}

// TestOccurrenceActionsRejectNonAssembly: the occurrence actions error on a part document (no
// active assembly), rather than mutating the wrong content.
func TestOccurrenceActionsRejectNonAssembly(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := s.SuppressOccurrence(nil, true); err == nil {
		t.Error("SuppressOccurrence on a part should error")
	}
	if err := s.DeleteOccurrence(nil); err == nil {
		t.Error("DeleteOccurrence on a part should error")
	}
}

// TestOccurrenceHandleSelectionKind pins the new selection kind.
func TestOccurrenceHandleSelectionKind(t *testing.T) {
	if got := (OccurrenceHandle{}).SelectionKind(); got != SelectOccurrence {
		t.Errorf("OccurrenceHandle kind = %d, want SelectOccurrence", got)
	}
}
