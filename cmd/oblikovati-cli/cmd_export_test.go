// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// saveCLIPart builds a part with build and saves it to dir/<name>, returning the path — the
// shared fixture for the DXF-export CLI tests.
func saveCLIPart(t *testing.T, dir, name string, build func(*compdef.PartComponentDefinition)) string {
	t.Helper()
	opd := filepath.Join(dir, name)
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := compdef.AddPart(ws, opd, true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	build(d.Content().(*compdef.PartComponentDefinition))
	if err := ws.Save(d); err != nil {
		t.Fatalf("save: %v", err)
	}
	return opd
}

// runCLIExport runs an export subcommand and returns its stdout plus the written DXF's contents.
func runCLIExport(t *testing.T, dxfPath string, args ...string) (stdout, dxf string) {
	t.Helper()
	out, err := runCLI(t, args...)
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	data, err := os.ReadFile(dxfPath)
	if err != nil {
		t.Fatalf("expected %s on disk: %v", dxfPath, err)
	}
	return out, string(data)
}

// TestCLIExportSketchDXF exports a part's sketch to a .dxf at a chosen version via the CLI.
func TestCLIExportSketchDXF(t *testing.T) {
	dir := t.TempDir()
	opd := saveCLIPart(t, dir, "plate.opd", func(p *compdef.PartComponentDefinition) {
		p.Sketches().Add(sketch.XYPlane()).Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(5, 5))
	})
	dxfPath := filepath.Join(dir, "plate.dxf")

	out, data := runCLIExport(t, dxfPath, "export", opd, dxfPath, "r2018")
	if !strings.Contains(out, "curves") || !strings.Contains(out, "r2018") {
		t.Errorf("export output = %q, want curve count + r2018", out)
	}
	if !strings.Contains(data, "AC1032") || !strings.Contains(data, "\nLINE\n") {
		t.Errorf("exported DXF missing AC1032 header or LINE entity")
	}
}

// TestCLIExportFlat develops a sheet-metal part's flat pattern to DXF via the CLI, asserting the
// outline lands on the Outline layer.
func TestCLIExportFlat(t *testing.T) {
	dir := t.TempDir()
	opd := saveCLIPart(t, dir, "tray.opd", func(p *compdef.PartComponentDefinition) {
		if _, err := p.EnableSheetMetal(); err != nil {
			t.Fatalf("EnableSheetMetal: %v", err)
		}
		sk := p.Sketches().Add(sketch.XYPlane())
		sk.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 4))
		feature.NewSheetMetalFaceFeatures(p.Features()).Add(&feature.SheetMetalFaceDefinition{Sketch: sk, ProfileIndex: 0, Operation: ops.NewBody})
		p.Recompute()
	})
	dxfPath := filepath.Join(dir, "tray-flat.dxf")

	out, data := runCLIExport(t, dxfPath, "export-flat", opd, dxfPath, "r2018")
	if !strings.Contains(out, "flat pattern") || !strings.Contains(out, "r2018") {
		t.Errorf("export-flat output = %q, want flat-pattern entity count + r2018", out)
	}
	if !strings.Contains(data, "\n2\nOutline\n") || !strings.Contains(data, "\nLINE\n") {
		t.Errorf("exported flat DXF missing the Outline layer or outline geometry")
	}
}
