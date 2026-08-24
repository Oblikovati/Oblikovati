// SPDX-License-Identifier: GPL-2.0-only

package material

import "testing"

func TestBuiltinsPresentAndReadOnly(t *testing.T) {
	lib := NewLibrary()
	if len(lib.Appearances()) == 0 || len(lib.Materials()) == 0 {
		t.Fatal("built-in catalog is empty")
	}
	steel, ok := lib.Material("steel")
	if !ok {
		t.Fatal("built-in material \"steel\" missing")
	}
	if steel.Source() != SourceBuiltin || steel.Source().Editable() {
		t.Errorf("steel source = %q (editable=%v), want builtin/read-only", steel.Source(), steel.Source().Editable())
	}
	// SetSpec must be a no-op on a built-in.
	before := steel.Density()
	steel.SetSpec(MaterialSpec{Density: 999})
	if steel.Density() != before {
		t.Error("SetSpec mutated a built-in material")
	}

	def := lib.DefaultAppearance()
	if def == nil {
		t.Fatal("built-in default appearance missing")
	}
	if def.Source() != SourceBuiltin || def.Source().Editable() {
		t.Errorf("default appearance source = %q (editable=%v), want builtin/read-only", def.Source(), def.Source().Editable())
	}
	beforeWeight := def.Base().Weight
	def.SetSpec(AppearanceSpec{Base: OpenPBRBase{Weight: 0}})
	if def.Base().Weight != beforeWeight {
		t.Error("SetSpec mutated a built-in appearance")
	}
}

// Every built-in material must reference an appearance that exists, or the renderer
// resolver would fall through to the default for a "known" material.
func TestBuiltinMaterialsReferenceRealAppearances(t *testing.T) {
	lib := NewLibrary()
	for _, m := range lib.Materials() {
		if _, ok := lib.Appearance(m.AppearanceID()); !ok {
			t.Errorf("material %q references missing appearance %q", m.ID(), m.AppearanceID())
		}
	}
}

func TestDuplicateAppearanceIsEditableSnapshot(t *testing.T) {
	lib := NewLibrary()
	r0 := lib.Revision()
	dup, err := lib.DuplicateAppearance(DefaultAppearanceID, "My Appearance", SourceProject)
	if err != nil {
		t.Fatalf("DuplicateAppearance: %v", err)
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
	lib.EditAppearance(dup.ID(), spec)
	def := lib.DefaultAppearance()
	if def.Coat().Weight == dup.Coat().Weight {
		t.Error("editing the duplicate leaked into the built-in appearance")
	}
}

func TestDuplicateNamesGetDistinctIDs(t *testing.T) {
	lib := NewLibrary()
	a, _ := lib.DuplicateAppearance(DefaultAppearanceID, "Custom", SourceProject)
	b, _ := lib.DuplicateAppearance(DefaultAppearanceID, "Custom", SourceProject)
	if a.ID() == b.ID() {
		t.Errorf("two duplicates share id %q; ids must be unique", a.ID())
	}
}

func TestRemoveRejectsBuiltin(t *testing.T) {
	lib := NewLibrary()
	if err := lib.RemoveMaterial("steel"); err == nil {
		t.Error("removing a built-in material should fail")
	}
	dup, _ := lib.DuplicateMaterial("steel", "Temp", SourceProject)
	if err := lib.RemoveMaterial(dup.ID()); err != nil {
		t.Fatalf("RemoveMaterial(custom): %v", err)
	}
	if _, ok := lib.Material(dup.ID()); ok {
		t.Error("removed material still present")
	}

	if err := lib.RemoveAppearance(DefaultAppearanceID); err == nil {
		t.Error("removing a built-in appearance should fail")
	}
	adup, _ := lib.DuplicateAppearance(DefaultAppearanceID, "Temp", SourceProject)
	if err := lib.RemoveAppearance(adup.ID()); err != nil {
		t.Fatalf("RemoveAppearance(custom): %v", err)
	}
	if _, ok := lib.Appearance(adup.ID()); ok {
		t.Error("removed appearance still present")
	}
}
