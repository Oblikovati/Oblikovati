// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestAssemblyRestoreRecipeResetsToSnapshot checks RestoreRecipe is a full replace, not a
// union: restoring an earlier snapshot onto an assembly that has since gained occurrences
// yields exactly the snapshot's occurrences after re-binding — the undo invariant the
// transaction stream depends on (#763). The reset would silently union (showing both
// placements) if RestoreRecipe merged like ApplyRecipe instead of clearing first.
func TestAssemblyRestoreRecipeResetsToSnapshot(t *testing.T) {
	_, _, asm, widget, asmDef := placedAssembly(t)

	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())
	oneOccurrence, err := asmDef.MarshalRecipe() // snapshot with exactly one occurrence
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	placeFromFile(t, asm, widget, asmDef, "widget:2", math.Identity4())
	if got := asmDef.Occurrences().Count(); got != 2 {
		t.Fatalf("after second place: occurrence count = %d, want 2", got)
	}

	if err := asmDef.RestoreRecipe(oneOccurrence); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := asmDef.Occurrences().Count(); got != 0 {
		t.Fatalf("restore stashes occurrences as pending until re-bind: count = %d, want 0", got)
	}
	if err := asmDef.ResolveReferences(asm); err != nil {
		t.Fatalf("resolve references: %v", err)
	}
	if got := asmDef.Occurrences().Count(); got != 1 {
		t.Errorf("after re-bind: occurrence count = %d, want 1 (the snapshot, not a union)", got)
	}
	if name := asmDef.Occurrences().Item(0).Name(); name != "widget:1" {
		t.Errorf("re-bound occurrence name = %q, want %q", name, "widget:1")
	}
	if cn := asmDef.Occurrences().Item(0).ComponentName(); cn != widget.FullDocumentName() {
		t.Errorf("re-bound occurrence component = %q, want %q", cn, widget.FullDocumentName())
	}
}

// TestAssemblyRestoreRecipeToEmptyRemovesOccurrences checks restoring the empty baseline
// clears every occurrence — undo of the very first placement back to a bare assembly (#763).
func TestAssemblyRestoreRecipeToEmptyRemovesOccurrences(t *testing.T) {
	_, _, asm, widget, asmDef := placedAssembly(t)
	empty, err := asmDef.MarshalRecipe() // baseline: no occurrences yet
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())

	if err := asmDef.RestoreRecipe(empty); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := asmDef.ResolveReferences(asm); err != nil {
		t.Fatalf("resolve references: %v", err)
	}
	if got := asmDef.Occurrences().Count(); got != 0 {
		t.Errorf("after restore to empty baseline: occurrence count = %d, want 0", got)
	}
}

// TestRestoreRewiresOccurrenceEvents guards the SetListener re-wire in resetOccurrences: a
// restore swaps in a fresh occurrence collection, and a subsequent placement must still raise
// an OccurrenceAdd event. A dropped listener would pass every count assertion yet silently stop
// the browser and event surface from updating after an undo (#763).
func TestRestoreRewiresOccurrenceEvents(t *testing.T) {
	_, _, asm, widget, asmDef := placedAssembly(t)
	empty, err := asmDef.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())
	if err := asmDef.RestoreRecipe(empty); err != nil { // swaps the occurrence collection
		t.Fatalf("restore: %v", err)
	}

	added := 0
	event.Subscribe(asmDef.Events().Bus(), event.After, func(_ event.Context, _ compdef.OccurrenceAdd) event.Outcome {
		added++
		return event.Continue()
	})
	placeFromFile(t, asm, widget, asmDef, "widget:2", math.Identity4())
	if added != 1 {
		t.Errorf("placement after restore raised %d OccurrenceAdd events, want 1 (the listener must survive the reset)", added)
	}
}
