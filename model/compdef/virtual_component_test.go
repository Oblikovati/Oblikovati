// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/compdef"
)

// TestAddVirtualHasNoGeometryButAppearsInBOM: a virtual component contributes no bounds (no range box,
// so no mass) yet appears as a counted BOM row with its part number and structure (#1979).
func TestAddVirtualHasNoGeometryButAppearsInBOM(t *testing.T) {
	t.Parallel()
	asm := compdef.NewAssemblyComponentDefinition()
	asm.AddVirtual("Paint", "PAINT-100", bom.Purchased, math.Identity4())

	if !asm.Occurrences().RangeBox().IsEmpty() {
		t.Errorf("assembly with only a virtual component has a non-empty range box, want empty (no geometry)")
	}

	rows := bom.New(asm.Occurrences()).PartsOnly().Rows
	if len(rows) != 1 || rows[0].PartNumber != "PAINT-100" || rows[0].Structure != bom.Purchased {
		t.Fatalf("BOM rows = %+v, want one PAINT-100 (purchased) row", rows)
	}
	if rows[0].Quantity != 1 {
		t.Errorf("virtual quantity = %d, want 1", rows[0].Quantity)
	}
}

// TestVirtualComponentSurvivesRoundTrip: a virtual component restores from the recipe without any
// backing document, keeping its part number and structure (#1979).
func TestVirtualComponentSurvivesRoundTrip(t *testing.T) {
	t.Parallel()
	store, ws, dir := assemblyWorkspace(t)
	asm, asmDef := newAssembly(t, ws, dir, "asm.obk")
	asmDef.AddVirtual("Grease", "GREASE-9", bom.Inseparable, math.Identity4())

	def := reopenAssembly(t, store, ws, asm)
	if def.Occurrences().Count() != 1 {
		t.Fatalf("reopened occurrence count = %d, want 1 (the virtual)", def.Occurrences().Count())
	}
	rows := bom.New(def.Occurrences()).PartsOnly().Rows
	if len(rows) != 1 || rows[0].PartNumber != "GREASE-9" || rows[0].Structure != bom.Inseparable {
		t.Errorf("reopened BOM = %+v, want one GREASE-9 (inseparable) row (virtual restored without a file)", rows)
	}
}

// TestAssemblyOptionsSurviveRoundTrip: the assembly editing options (LOD/design-view/section) persist
// through save/reopen (#1981).
func TestAssemblyOptionsSurviveRoundTrip(t *testing.T) {
	t.Parallel()
	store, ws, dir := assemblyWorkspace(t)
	asm, asmDef := newAssembly(t, ws, dir, "asm.obk")
	asmDef.SetOptions(compdef.AssemblyOptions{
		PlaceAndGroundFirstComponentAtOrigin: false,
		SectionAllParts:                      true,
		DefaultLevelOfDetail:                 "All Content Center",
		DefaultDesignView:                    "Default",
	})

	def := reopenAssembly(t, store, ws, asm)
	got := def.Options()
	if got.PlaceAndGroundFirstComponentAtOrigin || !got.SectionAllParts {
		t.Errorf("reopened options = %+v, want ground-first off / sectionAllParts on", got)
	}
	if got.DefaultLevelOfDetail != "All Content Center" || got.DefaultDesignView != "Default" {
		t.Errorf("reopened LOD/design-view = %q/%q, want the saved values", got.DefaultLevelOfDetail, got.DefaultDesignView)
	}
}
