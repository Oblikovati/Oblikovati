// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/math"
)

// TestAssemblyBOMViewOverWire reads parts-only and structured BOM views of an assembly
// holding two placements of one component: each view totals to a single row of quantity 2.
func TestAssemblyBOMViewOverWire(t *testing.T) {
	r, s, asm, _ := assemblySessionWithBoxes(t)
	part := blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	asm.Place("box:1", part, math.Identity4())
	asm.Place("box:2", part, math.Translation4(math.V3(2, 0, 0)))

	var parts wire.BOMViewResult
	call(t, r, s, "assembly.bomView", `{"view":"partsOnly"}`, &parts)
	if parts.View != types.BOMPartsOnly || len(parts.Rows) != 1 || parts.Rows[0].Quantity != 2 {
		t.Fatalf("parts-only = %+v, want one row of quantity 2", parts.Rows)
	}
	if parts.Rows[0].Structure != types.BOMNormal {
		t.Errorf("structure = %q, want normal", parts.Rows[0].Structure)
	}

	var structured wire.BOMViewResult
	call(t, r, s, "assembly.bomView", `{"view":"structured"}`, &structured)
	if len(structured.Rows) != 1 || structured.Rows[0].Quantity != 2 {
		t.Errorf("structured = %+v, want one row of quantity 2", structured.Rows)
	}

	if _, err := r.Handle(s, "assembly.bomView", []byte(`{"view":"bogus"}`)); err == nil {
		t.Error("an unknown view should fail")
	}
}

// TestAssemblyBOMExportOverWire exports a BOM to CSV with a custom property column and
// checks the header carries the standard columns plus the requested one.
func TestAssemblyBOMExportOverWire(t *testing.T) {
	r, s, asm, _ := assemblySessionWithBoxes(t)
	asm.Place("box:1", blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Identity4())

	var res wire.BOMExportResult
	call(t, r, s, "assembly.bomExport", `{"view":"partsOnly","columns":["Material"]}`, &res)
	header, _, _ := strings.Cut(res.CSV, "\n")
	for _, col := range []string{"Item", "Part Number", "Description", "QTY", "Structure", "Material"} {
		if !strings.Contains(header, col) {
			t.Errorf("CSV header %q missing column %q", header, col)
		}
	}

	if _, err := r.Handle(s, "assembly.bomExport", []byte(`{"view":"bogus"}`)); err == nil {
		t.Error("an unknown view should fail")
	}
}

// TestAssemblySetBOMStructureOverWire sets a per-occurrence BOM structure over the wire: marking one
// of two placements Reference drops it from the parts-only view, and an invalid structure is rejected
// (#1978).
func TestAssemblySetBOMStructureOverWire(t *testing.T) {
	r, s, asm, _ := assemblySessionWithBoxes(t)
	part := blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	asm.Place("box:1", part, math.Identity4())
	o2 := asm.Place("box:2", part, math.Translation4(math.V3(2, 0, 0)))

	var set wire.SetBOMStructureResult
	call(t, r, s, "assembly.setBOMStructure", mustJSON(t, wire.SetBOMStructureArgs{
		Occurrence: o2.ID(), Structure: types.BOMReference,
	}), &set)
	if set.Structure != types.BOMReference {
		t.Errorf("set structure = %q, want reference", set.Structure)
	}

	var parts wire.BOMViewResult
	call(t, r, s, "assembly.bomView", `{"view":"partsOnly"}`, &parts)
	if len(parts.Rows) != 1 || parts.Rows[0].Quantity != 1 {
		t.Fatalf("parts-only = %+v, want quantity 1 (the reference placement dropped)", parts.Rows)
	}

	if _, err := r.Handle(s, "assembly.setBOMStructure", []byte(mustJSON(t, wire.SetBOMStructureArgs{
		Occurrence: o2.ID(), Structure: types.BOMStructure("varies"),
	}))); err == nil {
		t.Error("setBOMStructure with a computed value (varies) should fail")
	}
}
