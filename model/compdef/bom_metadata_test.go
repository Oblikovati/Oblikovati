// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/bom"
)

// TestBOMReadsComponentIProperties is the #718 payoff: a BOM over an assembly shows each
// component's part number, description, and custom property columns, sourced from the component
// document's iProperties — not the blank default.
func TestBOMReadsComponentIProperties(t *testing.T) {
	part := NewPartComponentDefinition()
	part.Properties().Set(attr.DesignTracking).Put(propPartNumber, attr.StringValue("BRK-001"))
	part.Properties().Set(attr.DesignTracking).Put(propDescription, attr.StringValue("Bracket"))
	part.Properties().Set(attr.UserDefined).Put("Material", attr.StringValue("Steel"))

	asm := NewAssemblyComponentDefinition()
	asm.Occurrences().AddByComponentDefinition("p:1", part, math.Identity4())

	rows := bom.New(asm.Occurrences()).PartsOnly().Rows
	if len(rows) != 1 {
		t.Fatalf("BOM rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.PartNumber != "BRK-001" {
		t.Errorf("row PartNumber = %q, want BRK-001", r.PartNumber)
	}
	if r.Description != "Bracket" {
		t.Errorf("row Description = %q, want Bracket", r.Description)
	}
	if r.Properties["Material"] != "Steel" {
		t.Errorf("custom column Material = %q, want Steel", r.Properties["Material"])
	}
}

// TestBOMStructureFromProperty: a component's BOM structure is read from its "BOM Structure"
// Design Tracking property (default Normal when unset).
func TestBOMStructureFromProperty(t *testing.T) {
	d := NewPartComponentDefinition()
	if got := d.BOMStructure(); got != bom.Normal {
		t.Errorf("default BOMStructure = %v, want Normal", got)
	}
	d.Properties().Set(attr.DesignTracking).Put(propBOMStructure, attr.StringValue("purchased"))
	if got := d.BOMStructure(); got != bom.Purchased {
		t.Errorf("BOMStructure after setting the property = %v, want Purchased", got)
	}
}

// TestEmptyDocumentBOMMetadataIsBlank: a document with no iProperties reports blank metadata (the
// BOM falls back to a normal, unnamed line), so the default is unchanged.
func TestEmptyDocumentBOMMetadataIsBlank(t *testing.T) {
	d := NewPartComponentDefinition()
	if d.PartNumber() != "" || d.Description() != "" || d.CustomProperties() != nil {
		t.Errorf("empty document metadata = (%q, %q, %v), want blank", d.PartNumber(), d.Description(), d.CustomProperties())
	}
}
