// SPDX-License-Identifier: GPL-2.0-only

package bom

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// TestOverridePhantomPromotesChildren: a per-occurrence Phantom override on a Normal-definition
// sub-assembly promotes its children in both views (#1978).
func TestOverridePhantomPromotesChildren(t *testing.T) {
	plate := &fakePart{num: "PLATE", structure: Normal}
	rig := newSub("RIG", Normal) // the shared definition is Normal…
	rig.children.AddByComponentDefinition("plate:1", plate, math.Identity4())

	top := occurrence.NewOccurrences()
	occ := top.AddByComponentDefinition("rig:1", rig, math.Identity4())
	occ.SetBOMStructureOverride("phantom") // …but this placement is phantom

	sv := New(top).Structured()
	if len(sv.Rows) != 1 || sv.Rows[0].PartNumber != "PLATE" {
		t.Fatalf("structured = %v, want the promoted PLATE (phantom override collapsed the rig)", rowNames(sv.Rows))
	}
	pv := New(top).PartsOnly()
	if len(pv.Rows) != 1 || pv.Rows[0].PartNumber != "PLATE" || pv.Rows[0].Quantity != 1 {
		t.Errorf("parts-only = %v, want PLATE qty 1 (override reflected in the flat view too)", rowNames(pv.Rows))
	}
}

// TestOverrideDiffersFromDefinition: a per-occurrence Purchased override on a Normal sub-assembly
// makes it a single opaque line whose children are not broken out — differing from the definition
// (#1978).
func TestOverrideDiffersFromDefinition(t *testing.T) {
	screw := &fakePart{num: "SCREW", structure: Normal}
	gearbox := newSub("GEARBOX", Normal) // definition Normal (would expand)
	gearbox.children.AddByComponentDefinition("screw:1", screw, math.Identity4())

	top := occurrence.NewOccurrences()
	occ := top.AddByComponentDefinition("gearbox:1", gearbox, math.Identity4())
	occ.SetBOMStructureOverride("purchased")

	sv := New(top).Structured()
	if len(sv.Rows) != 1 || sv.Rows[0].PartNumber != "GEARBOX" || sv.Rows[0].Structure != Purchased {
		t.Fatalf("structured = %+v, want one GEARBOX row marked purchased", sv.Rows)
	}
	if len(sv.Rows[0].Children) != 0 {
		t.Errorf("purchased override still broke out %d children, want opaque", len(sv.Rows[0].Children))
	}
}

// TestMixedOverridesReportVaries: two placements of one definition with differing structures group to
// a single row reported as Varies (iAssembly semantics) (#1978).
func TestMixedOverridesReportVaries(t *testing.T) {
	widget := &fakePart{num: "WIDGET", structure: Normal}
	top := occurrence.NewOccurrences()
	a := top.AddByComponentDefinition("widget:1", widget, math.Identity4())
	b := top.AddByComponentDefinition("widget:2", widget, math.Translation4(math.V3(1, 0, 0)))
	a.SetBOMStructureOverride("purchased")
	b.SetBOMStructureOverride("inseparable")

	sv := New(top).Structured()
	if len(sv.Rows) != 1 || sv.Rows[0].Structure != Varies {
		t.Fatalf("structured = %+v, want one WIDGET row marked varies", sv.Rows)
	}
}

// TestOverrideDefaultInherits: clearing an override with "default" defers to the definition.
func TestOverrideDefaultInherits(t *testing.T) {
	widget := &fakePart{num: "WIDGET", structure: Reference}
	top := occurrence.NewOccurrences()
	occ := top.AddByComponentDefinition("widget:1", widget, math.Identity4())
	occ.SetBOMStructureOverride("normal")
	if q := len(New(top).PartsOnly().Rows); q != 1 {
		t.Fatalf("with normal override, rows = %d, want 1", q)
	}
	occ.SetBOMStructureOverride("default") // back to the definition, which is Reference (dropped)
	if q := len(New(top).PartsOnly().Rows); q != 0 {
		t.Errorf("with default (inherit Reference), rows = %d, want 0", q)
	}
}

// TestStructureParsesDefaultAndVaries: the new Default and Varies structures round-trip (#1978).
func TestStructureParsesDefaultAndVaries(t *testing.T) {
	for _, s := range []Structure{Default, Varies} {
		got, ok := ParseStructure(s.String())
		if !ok || got != s {
			t.Errorf("ParseStructure(%q) = (%v, %v), want (%v, true)", s.String(), got, ok, s)
		}
	}
}
