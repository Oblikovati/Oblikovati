// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// cmBox is a 6×6×6 database-unit (centimetre) cube — 216 cm³.
func cmBox(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(6, 6, 6), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

func boxVolume(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	return query.BodyGeometryProperties(b, tessellate.DefaultQuality()).Volume
}

// TestMeshExportImportUnitRoundTrip checks each mesh format preserves the physical
// size across an export-in-a-unit / import round-trip back into the centimetre
// kernel (#146): 216 cm³ regardless of the file unit.
func TestMeshExportImportUnitRoundTrip(t *testing.T) {
	box := cmBox(t)
	for _, format := range []types.ExchangeFormat{types.FormatSTL, types.FormatOBJ, types.Format3MF} {
		for _, unit := range []string{"mm", "cm", "in"} {
			data, _, err := ExportBodies(format, []*topo.Body{box}, types.ResolutionHigh,
				exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: unit})
			if err != nil {
				t.Fatalf("export %s/%s: %v", format, unit, err)
			}
			body, _, err := ImportBody(format, data, "rt", DefaultWeldTolerance,
				exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
			if err != nil {
				t.Fatalf("import %s/%s: %v", format, unit, err)
			}
			if v := boxVolume(t, body); v < 215 || v > 217 {
				t.Errorf("%s/%s round-trip volume = %.2f cm³, want ~216", format, unit, v)
			}
		}
	}
}

// TestThreeMFDeclaresAndReadsUnit checks 3MF writes the document unit attribute and
// reads it back: an inch export declares inch, and importing it (which assumes the
// declared unit) yields the same 216 cm³ as a millimetre export.
func TestThreeMFDeclaresAndReadsUnit(t *testing.T) {
	in, _, err := ExportBodies(types.Format3MF, []*topo.Body{cmBox(t)}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "in"})
	if err != nil {
		t.Fatalf("export 3mf inch: %v", err)
	}
	// 3MF is a ZIP, so read the declared unit back through the decoder (not the raw bytes).
	if got := read3MFUnit(in); got != "inch" {
		t.Errorf("read3MFUnit = %q, want inch", got)
	}
}

// TestSTLImportAssumesMillimetres pins the convention: a unitless 60-unit STL cube
// imports as a 6 cm (216 cm³) body, not 60 cm.
func TestSTLImportAssumesMillimetres(t *testing.T) {
	// Author a 60 mm STL by exporting the 6 cm box in millimetres.
	stl, _, err := ExportBodies(types.FormatSTL, []*topo.Body{cmBox(t)}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "mm"})
	if err != nil {
		t.Fatalf("author stl: %v", err)
	}
	body, _, err := ImportBody(types.FormatSTL, stl, "rt", DefaultWeldTolerance,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if v := boxVolume(t, body); v < 215 || v > 217 {
		t.Errorf("60 mm STL imported as %.2f cm³, want ~216 (6 cm)", v)
	}
}

// TestRead3MFUnitBadInput returns "" for non-3MF bytes (the caller then assumes mm).
func TestRead3MFUnitBadInput(t *testing.T) {
	if got := read3MFUnit([]byte("not a zip")); got != "" {
		t.Errorf("read3MFUnit(garbage) = %q, want \"\"", got)
	}
}
