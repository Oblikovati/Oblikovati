// SPDX-License-Identifier: GPL-2.0-only

package material

import "testing"

func TestOpenPBRBuiltinPresentAndReadOnly(t *testing.T) {
	lib := NewLibrary()
	def := lib.DefaultOpenPBRAppearance()
	if def == nil {
		t.Fatal("built-in openpbr appearance missing")
	}
	if def.Source() != SourceBuiltin || def.Source().Editable() {
		t.Errorf("default source = %q (editable=%v), want builtin/read-only", def.Source(), def.Source().Editable())
	}
	// SetSpec must be a no-op on a built-in.
	before := def.Base().Weight
	def.SetSpec(OpenPBRAppearanceSpec{Base: OpenPBRBase{Weight: 0}})
	if def.Base().Weight != before {
		t.Error("SetSpec mutated a built-in openpbr appearance")
	}
}

func TestDuplicateOpenPBRAppearanceIsEditableSnapshot(t *testing.T) {
	lib := NewLibrary()
	r0 := lib.Revision()
	dup, err := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "My OpenPBR", SourceProject)
	if err != nil {
		t.Fatalf("DuplicateOpenPBRAppearance: %v", err)
	}
	if dup.Source() != SourceProject || !dup.Source().Editable() {
		t.Errorf("dup source = %q, want editable project", dup.Source())
	}
	if lib.Revision() <= r0 {
		t.Error("duplicate did not bump the revision")
	}
	// Editing the copy must not touch the built-in it came from.
	spec := dup.Spec()
	spec.Coat.Weight = 1
	lib.EditOpenPBRAppearance(dup.ID(), spec)
	def := lib.DefaultOpenPBRAppearance()
	if def.Coat().Weight == dup.Coat().Weight {
		t.Error("editing the duplicate leaked into the built-in openpbr appearance")
	}
}

func TestDuplicateOpenPBRAppearanceNamesGetDistinctIDs(t *testing.T) {
	lib := NewLibrary()
	a, _ := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "Custom", SourceProject)
	b, _ := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "Custom", SourceProject)
	if a.ID() == b.ID() {
		t.Errorf("two duplicates share id %q; ids must be unique", a.ID())
	}
}

func TestRemoveOpenPBRAppearanceRejectsBuiltin(t *testing.T) {
	lib := NewLibrary()
	if err := lib.RemoveOpenPBRAppearance(DefaultOpenPBRAppearanceID); err == nil {
		t.Error("removing a built-in openpbr appearance should fail")
	}
	dup, _ := lib.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "Temp", SourceProject)
	if err := lib.RemoveOpenPBRAppearance(dup.ID()); err != nil {
		t.Fatalf("RemoveOpenPBRAppearance(custom): %v", err)
	}
	if _, ok := lib.OpenPBRAppearance(dup.ID()); ok {
		t.Error("removed openpbr appearance still present")
	}
}
