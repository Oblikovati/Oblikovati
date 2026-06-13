// SPDX-License-Identifier: GPL-2.0-only

package bom

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// TestExportCSVWithCustomColumns is the PBI-124 acceptance: the BOM exports with a
// custom property column sourced from a component's iProperties.
func TestExportCSVWithCustomColumns(t *testing.T) {
	widget := &fakePart{num: "W-1", desc: "Widget", structure: Normal, props: map[string]string{"Material": "Steel"}}
	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("w:1", widget, math.Identity4())
	top.AddByComponentDefinition("w:2", widget, math.Identity4())

	columns := append(StandardColumns(), PropertyColumn("Material"))
	csvText, err := ExportCSV(New(top).PartsOnly(), columns)
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	for _, want := range []string{
		"Item,Part Number,Description,QTY,Structure,Material",
		"1,W-1,Widget,2,normal,Steel",
	} {
		if !strings.Contains(csvText, want) {
			t.Errorf("CSV missing %q; got:\n%s", want, csvText)
		}
	}
}

// TestExportFlattensStructuredView checks a structured view exports each row before
// its children, depth-first.
func TestExportFlattensStructuredView(t *testing.T) {
	bolt := &fakePart{num: "BOLT", structure: Normal}
	sub := newSub("SUB", Normal)
	sub.children.AddByComponentDefinition("bolt:1", bolt, math.Identity4())
	top := occurrence.NewOccurrences()
	top.AddByComponentDefinition("sub:1", sub, math.Identity4())

	csvText, err := ExportCSV(New(top).Structured(), StandardColumns())
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	subAt, boltAt := strings.Index(csvText, "SUB"), strings.Index(csvText, "BOLT")
	if subAt < 0 || boltAt < 0 || subAt > boltAt {
		t.Errorf("structured export should list SUB before its child BOLT; got:\n%s", csvText)
	}
}
