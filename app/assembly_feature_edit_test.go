// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// chamferedAssembly returns a session whose active assembly has a box placed and a 0.2 chamfer on a
// vertical edge — the fixture the feature-edit tests start from.
func chamferedAssembly(t *testing.T) (*Session, *compdef.AssemblyComponentDefinition, *occurrence.Occurrence) {
	t.Helper()
	s, asm, o := assemblyWithBoxComponent(t, 0)
	tool := NewAssemblyChamferTool()
	tool.Pick(s, worldVerticalEdge(t, s))
	tool.distance = 0.2
	if err := tool.Commit(s); err != nil {
		t.Fatalf("chamfer: %v", err)
	}
	return s, asm, o
}

// setEditParam sets a named editable parameter on the edit tool, failing if it is absent.
func setEditParam(t *testing.T, tool *AssemblyFeatureEditTool, label string, v float64) {
	t.Helper()
	for _, p := range tool.params {
		if p.Label == label {
			p.Set(v)
			return
		}
	}
	t.Fatalf("edit tool has no %q parameter", label)
}

// TestAssemblyBrowserListsFeatures: a committed machining feature appears in a Features folder as a
// selectable AssemblyFeatureHandle row (#766).
func TestAssemblyBrowserListsFeatures(t *testing.T) {
	s, asm, _ := chamferedAssembly(t)
	name := asm.Features().Item(0).Name()

	node := findBrowserNode(BuildBrowser(s), "assemblyFeature", name)
	if node == nil {
		t.Fatalf("the assembly browser has no feature row for %q", name)
	}
	if _, ok := node.Select.(AssemblyFeatureHandle); !ok {
		t.Errorf("feature node selects %T, want AssemblyFeatureHandle", node.Select)
	}
}

// TestAssemblyFeatureEditChangesParameter: editing the chamfer's distance re-machines the
// participant — a bigger setback removes more material (#766).
func TestAssemblyFeatureEditChangesParameter(t *testing.T) {
	s, asm, occ := chamferedAssembly(t)
	before := participantMachinedVolume(asm, occ) // 16 − 0.2²/2·4 = 15.92

	edit := NewAssemblyFeatureEditTool(asm.Features().Item(0))
	setEditParam(t, edit, "Distance", 0.4)
	if err := edit.Commit(s); err != nil {
		t.Fatalf("edit commit: %v", err)
	}
	// A 0.4 setback removes 0.4²/2·4 = 0.32 ⇒ 15.68.
	got := participantMachinedVolume(asm, occ)
	if got >= before {
		t.Errorf("a bigger chamfer should remove more: %g → %g", before, got)
	}
	if stdmath.Abs(got-15.68) > 0.02 {
		t.Errorf("edited chamfered volume = %g, want 15.68 (a 0.4 setback)", got)
	}
}

// TestAssemblyFeatureEditCancelRestores: cancelling an edit restores the parameter captured at open.
func TestAssemblyFeatureEditCancelRestores(t *testing.T) {
	s, asm, occ := chamferedAssembly(t)
	original := participantMachinedVolume(asm, occ)

	edit := NewAssemblyFeatureEditTool(asm.Features().Item(0))
	setEditParam(t, edit, "Distance", 0.4)
	edit.Cancel(s)
	if got := participantMachinedVolume(asm, occ); stdmath.Abs(got-original) > 1e-9 {
		t.Errorf("cancel should restore the volume: %g, want %g", got, original)
	}
}

// TestAssemblyFeatureEditIsUndoable: a committed parameter edit is one undo step.
func TestAssemblyFeatureEditIsUndoable(t *testing.T) {
	s, asm, occ := chamferedAssembly(t)
	original := participantMachinedVolume(asm, occ)
	trackFromHere(s) // baseline: the 0.2 chamfer

	edit := NewAssemblyFeatureEditTool(asm.Features().Item(0))
	setEditParam(t, edit, "Distance", 0.4)
	if err := edit.Commit(s); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	// Undo re-creates the occurrences (full-replace restore), so re-fetch the participant.
	restored := asm.Occurrences().Item(0)
	if got := participantMachinedVolume(asm, restored); stdmath.Abs(got-original) > 0.02 {
		t.Errorf("undo should restore the 0.2 chamfer: volume %g, want %g", got, original)
	}
}

// TestSuppressAssemblyFeatureToCurrentStateRecordsNothing: suppressing a feature to the state it is
// already in is a no-op and pushes no undo step (#766).
func TestSuppressAssemblyFeatureToCurrentStateRecordsNothing(t *testing.T) {
	s, asm, _ := chamferedAssembly(t)
	trackFromHere(s) // baseline: the feature is unsuppressed
	if err := s.SuppressAssemblyFeature(asm.Features().Item(0), false); err != nil {
		t.Fatalf("suppress to current state: %v", err)
	}
	if s.CanUndo() {
		t.Error("suppressing to the current state should record no undo step")
	}
}

// TestDeleteAssemblyFeatureRejectsUnknown: deleting a feature that is not in the assembly (here, a
// second delete of an already-removed feature) errors and names the feature is not present.
func TestDeleteAssemblyFeatureRejectsUnknown(t *testing.T) {
	s, asm, _ := chamferedAssembly(t)
	af := asm.Features().Item(0)
	if err := s.DeleteAssemblyFeature(af); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := s.DeleteAssemblyFeature(af); err == nil {
		t.Error("deleting a feature not in the assembly should error")
	}
}

// TestAssemblyFeatureMenuAndDelete: the feature menu offers Edit/Suppress/Delete, and Delete
// removes the feature (undoably).
func TestAssemblyFeatureMenuAndDelete(t *testing.T) {
	s, asm, _ := chamferedAssembly(t)
	af := asm.Features().Item(0)
	node := findBrowserNode(BuildBrowser(s), "assemblyFeature", af.Name())

	if labels := strings.Join(menuLabels(BrowserMenu(s, *node)), "|"); labels != "Edit|Suppress|Delete" {
		t.Errorf("feature menu = %q, want Edit|Suppress|Delete", labels)
	}
	trackFromHere(s)
	if err := s.DeleteAssemblyFeature(af); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if asm.Features().Count() != 0 {
		t.Fatalf("after delete: feature count = %d, want 0", asm.Features().Count())
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if asm.Features().Count() != 1 {
		t.Errorf("undo should restore the feature: count = %d, want 1", asm.Features().Count())
	}
}
