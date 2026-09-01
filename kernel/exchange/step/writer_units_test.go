// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
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

// TestExportHonorsFileUnit checks that an export declares the requested file unit
// and scales the kernel's centimetre geometry into it (#146): a 6 cm cube becomes
// "60" in a millimetre file and "6" in a centimetre file, and inch uses a
// conversion-based unit. The centimetre kernel is anchored by DBUnitMM.
func TestExportHonorsFileUnit(t *testing.T) {
	t.Parallel()
	box := cmBox(t)

	mm, _, err := Writer{}.ExportSolids([]*topo.Body{box},
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "mm"})
	if err != nil {
		t.Fatalf("export mm: %v", err)
	}
	if !strings.Contains(string(mm), "SI_UNIT(.MILLI.,.METRE.)") {
		t.Error("mm export must declare a millimetre SI unit")
	}
	if !strings.Contains(string(mm), "60.") {
		t.Error("a 6 cm edge must be written as 60 in a millimetre file")
	}

	cm, _, err := Writer{}.ExportSolids([]*topo.Body{box},
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "cm"})
	if err != nil {
		t.Fatalf("export cm: %v", err)
	}
	if !strings.Contains(string(cm), "SI_UNIT(.CENTI.,.METRE.)") {
		t.Error("cm export must declare a centimetre SI unit")
	}

	in, _, err := Writer{}.ExportSolids([]*topo.Body{box},
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "in"})
	if err != nil {
		t.Fatalf("export in: %v", err)
	}
	if !strings.Contains(string(in), "CONVERSION_BASED_UNIT") {
		t.Error("inch export must declare a conversion-based unit")
	}

	_, _, badErr := Writer{}.ExportSolids([]*topo.Body{box}, exchange.TranslationOptions{FileUnit: "furlong"})
	if badErr == nil {
		t.Error("an unknown file unit must error")
	}
}

// TestExportImportUnitRoundTrip checks the physical size survives a round-trip in
// each file unit: export the 6 cm cube, re-import into the centimetre kernel, and
// the volume is 216 cm³ regardless of the file unit used.
func TestExportImportUnitRoundTrip(t *testing.T) {
	t.Parallel()
	box := cmBox(t)
	for _, unit := range []string{"mm", "cm", "m", "in", "ft"} {
		data, _, err := Writer{}.ExportSolids([]*topo.Body{box},
			exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: unit})
		if err != nil {
			t.Fatalf("export %s: %v", unit, err)
		}
		bodies, _, err := Reader{}.ImportSolids(data,
			exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
		if err != nil || len(bodies) != 1 {
			t.Fatalf("re-import %s: %v (%d bodies)", unit, err, len(bodies))
		}
		got := query.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
		if rel := (got - 216) / 216; rel < -0.01 || rel > 0.01 {
			t.Errorf("%s round-trip volume = %.3f cm³, want 216", unit, got)
		}
	}
}

// TestImportScalesFileUnitToCentimetres pins the import-side fix: a millimetre STEP
// (declared mm) of a 60 mm cube imports as a 6 cm (216 cm³) body, not 60 cm.
func TestImportScalesFileUnitToCentimetres(t *testing.T) {
	t.Parallel()
	// A 60 mm cube authored by exporting the 6 cm kernel box in millimetres.
	mmFile, _, err := Writer{}.ExportSolids([]*topo.Body{cmBox(t)},
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "mm"})
	if err != nil {
		t.Fatalf("author mm file: %v", err)
	}
	bodies, _, err := Reader{}.ImportSolids(mmFile,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import: %v (%d bodies)", err, len(bodies))
	}
	got := query.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
	if rel := (got - 216) / 216; rel < -0.01 || rel > 0.01 {
		t.Errorf("imported volume = %.3f cm³, want 216 (a 60 mm cube is 6 cm)", got)
	}
}
