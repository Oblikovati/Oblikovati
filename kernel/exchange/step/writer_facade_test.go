// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/exchange/step/schema"
	"oblikovati.org/kernel/topo"
)

func TestExportedFileDeclaresAP203(t *testing.T) {
	body := importOneSolid(t, "cube.step")
	data, _, err := Writer{}.ExportSolids([]*topo.Body{body}, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	f, err := part21.Parse(data)
	if err != nil {
		t.Fatalf("emitted file does not re-parse: %v", err)
	}
	if got := schema.Detect(f.Header.SchemaIdentifiers); got != schema.AP203 {
		t.Errorf("emitted schema = %s, want AP203", got)
	}
	if len(f.Graph.EntitiesOfType("MANIFOLD_SOLID_BREP")) != 1 {
		t.Error("exported cube should contain exactly one MANIFOLD_SOLID_BREP")
	}
}

func TestExportIsByteStable(t *testing.T) {
	body := importOneSolid(t, "cube.step")
	a, _, _ := Writer{}.ExportSolids([]*topo.Body{body}, exchange.TranslationOptions{})
	b, _, _ := Writer{}.ExportSolids([]*topo.Body{body}, exchange.TranslationOptions{})
	if string(a) != string(b) {
		t.Error("export is not byte-stable across runs")
	}
}
