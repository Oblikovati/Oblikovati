// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/param"
)

// cubeSTLBytes builds a side-2 cube as binary STL — file bytes for the import readers.
func cubeSTLBytes(t *testing.T) []byte {
	t.Helper()
	body, _, err := meshio.SolidOrSurface(unitCubeSoup(2), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build cube: %v", err)
	}
	return meshio.EncodeBinarySTL(body, ops.DefaultQuality())
}

// TestImportBodiesFromDataScalesByTargetUnit verifies the ADR-0042 Phase 2 import boundary
// (#1248): a smaller working-unit millimetre size yields proportionally larger working
// coordinates, so the same file imports into the document's working scale.
func TestImportBodiesFromDataScalesByTargetUnit(t *testing.T) {
	data := cubeSTLBytes(t)

	coarse, _, err := ImportBodiesFromData(types.FormatSTL, data, 10) // 10 mm per working unit (cm)
	if err != nil {
		t.Fatalf("import at 10mm: %v", err)
	}
	fine, _, err := ImportBodiesFromData(types.FormatSTL, data, 5) // 5 mm per working unit
	if err != nil {
		t.Fatalf("import at 5mm: %v", err)
	}
	// Halving the working-unit size doubles the working coordinates.
	cd := float64(coarse[0].RangeBox().Diagonal().Length())
	fd := float64(fine[0].RangeBox().Diagonal().Length())
	if cd <= 0 || fd/cd < 1.9 || fd/cd > 2.1 {
		t.Errorf("diagonal ratio fine/coarse = %v, want ~2", fd/cd)
	}

	// targetUnitMM 0 selects the centimetre default (== importing at DBUnitMM).
	def, _, err := ImportBodiesFromData(types.FormatSTL, data, 0)
	if err != nil {
		t.Fatalf("import at default: %v", err)
	}
	dd := float64(def[0].RangeBox().Diagonal().Length())
	if dd/cd < 0.99 || dd/cd > 1.01 {
		t.Errorf("default (0) should equal DBUnitMM (%v): ratio %v", exchange.DBUnitMM, dd/cd)
	}

	if _, _, err := ImportBodiesFromData("nope", data, 10); err == nil {
		t.Error("unsupported format should error")
	}

	// The STEP branch scales the same way (covers the B-rep reader path).
	cube, _, err := meshio.SolidOrSurface(unitCubeSoup(2), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build cube: %v", err)
	}
	stepData, _, err := step.Writer{}.ExportSolids([]*topo.Body{cube}, exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "mm"})
	if err != nil {
		t.Fatalf("export step: %v", err)
	}
	if bodies, _, err := ImportBodiesFromData(types.FormatSTEP, stepData, 10); err != nil || len(bodies) == 0 {
		t.Fatalf("import step = %d bodies, err %v; want ≥1", len(bodies), err)
	}
}

// TestImportBodiesReadsFile covers the path-reading wrapper.
func TestImportBodiesReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cube.stl")
	if err := os.WriteFile(path, cubeSTLBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	bodies, _, err := ImportBodies(types.FormatSTL, path, 10)
	if err != nil || len(bodies) != 1 {
		t.Fatalf("ImportBodies = %d bodies, err %v; want 1", len(bodies), err)
	}
	if _, _, err := ImportBodies(types.FormatSTL, filepath.Join(t.TempDir(), "missing.stl"), 10); err == nil {
		t.Error("missing file should error")
	}
}

// TestWorkingScaleResolverFeedsReimport checks the engine resolver: a re-import reads the live
// working scale (cm per working unit) and converts it to the millimetre target, falling back to
// the centimetre default when no resolver is wired.
func TestWorkingScaleResolverFeedsReimport(t *testing.T) {
	fs := NewPartFeatures(param.NewParameters(), identity.NewKeyManager())
	if got := fs.workingTargetMM(); got != 0 {
		t.Errorf("unwired resolver workingTargetMM = %v, want 0 (default applied downstream)", got)
	}
	fs.SetWorkingScaleResolver(func() float64 { return 1e-4 }) // µm working scale
	if got, want := fs.workingTargetMM(), 1e-4*exchange.DBUnitMM; got != want {
		t.Errorf("workingTargetMM = %v, want %v", got, want)
	}
}
