// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/feature"
)

func TestRenameFeatureSetsNameAndKeepsID(t *testing.T) {
	s, def := extrudedBoxPart(t)
	f := def.Features().Item(0)
	id := f.ID()
	if err := s.RenameFeature(f, "Base Boss"); err != nil {
		t.Fatalf("RenameFeature: %v", err)
	}
	if f.Name() != "Base Boss" {
		t.Errorf("name after rename = %q, want %q", f.Name(), "Base Boss")
	}
	if f.ID() != id {
		t.Errorf("id changed across rename: %d -> %d", id, f.ID())
	}
}

func TestRenameFeatureRejectsEmptyAndDuplicate(t *testing.T) {
	s, _, first, second := twoExtrudePart(t)
	if err := s.RenameFeature(first, ""); err == nil {
		t.Error("expected an error for an empty name")
	}
	if err := s.RenameFeature(second, first.Name()); err == nil {
		t.Errorf("expected an error renaming to the duplicate name %q", first.Name())
	}
	// Renaming a feature to its own current name is a no-op, not a duplicate.
	if err := s.RenameFeature(first, first.Name()); err != nil {
		t.Errorf("rename to own name: %v", err)
	}
}

func TestSetFeatureSuppressedRebuildsAndIsIdempotent(t *testing.T) {
	s, def := extrudedBoxPart(t)
	f := def.Features().Item(0)
	if err := s.SetFeatureSuppressed(f, true); err != nil {
		t.Fatalf("SetFeatureSuppressed(true): %v", err)
	}
	if n := len(def.SurfaceBodies().All()); n != 0 {
		t.Errorf("bodies with the only extrude suppressed = %d, want 0", n)
	}
	if err := s.SetFeatureSuppressed(f, true); err != nil {
		t.Fatalf("repeated SetFeatureSuppressed(true): %v", err)
	}
	if err := s.SetFeatureSuppressed(f, false); err != nil {
		t.Fatalf("SetFeatureSuppressed(false): %v", err)
	}
	if n := len(def.SurfaceBodies().All()); n != 1 {
		t.Errorf("bodies after unsuppress = %d, want 1", n)
	}
}

func TestReorderFeatureMovesHistoryPosition(t *testing.T) {
	s, part, _, second := twoExtrudePart(t)
	if err := s.ReorderFeature(second, 0); err != nil {
		t.Fatalf("ReorderFeature: %v", err)
	}
	if got := part.Features().Item(0); got != second {
		t.Errorf("feature at index 0 after reorder = %q, want %q", got.Name(), second.Name())
	}
	if err := s.ReorderFeature(second, 99); err == nil {
		t.Error("expected an error for an out-of-range index")
	}
}

func TestCommitFeatureEditAppliesScalarChange(t *testing.T) {
	s, def := extrudedBoxPart(t)
	f := def.Features().Item(0)
	ed, ok := f.Definition().(feature.Editable)
	if !ok {
		t.Fatal("extrude definition does not implement Editable")
	}
	ed.EditableParams()[0].Set(8) // distance, database units (cm)
	if err := s.CommitFeatureEdit(f); err != nil {
		t.Fatalf("CommitFeatureEdit: %v", err)
	}
	if got := ed.EditableParams()[0].Get(); got != 8 {
		t.Errorf("distance after edit = %v, want 8", got)
	}
	if n := len(def.SurfaceBodies().All()); n != 1 {
		t.Errorf("bodies after committed edit = %d, want 1", n)
	}
}

func TestFeatureLifecycleRejectsForeignFeature(t *testing.T) {
	s, _ := extrudedBoxPart(t)
	_, otherDef := extrudedBoxPart(t) // a second, unrelated session's part
	foreign := otherDef.Features().Item(0)
	if err := s.RenameFeature(foreign, "x"); err == nil {
		t.Error("expected an error renaming a feature from another part")
	}
	if err := s.SetFeatureSuppressed(foreign, true); err == nil {
		t.Error("expected an error suppressing a feature from another part")
	}
	if err := s.CommitFeatureEdit(foreign); err == nil {
		t.Error("expected an error committing an edit on a feature from another part")
	}
}
