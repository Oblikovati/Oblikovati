// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// writeCLIPartWithSketch saves a part document carrying one sketch with a single line, for
// the DXF-export CLI test.
func writeCLIPartWithSketch(t *testing.T, dir string) string {
	t.Helper()
	opd := filepath.Join(dir, "plate.opd")
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := compdef.AddPart(ws, opd, true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	part := d.Content().(*compdef.PartComponentDefinition)
	sk := part.Sketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(5, 5))
	if err := ws.Save(d); err != nil {
		t.Fatalf("save: %v", err)
	}
	return opd
}

// TestCLIExportSketchDXF exports a part's sketch to a .dxf at a chosen version via the CLI.
func TestCLIExportSketchDXF(t *testing.T) {
	dir := t.TempDir()
	opd := writeCLIPartWithSketch(t, dir)
	dxfPath := filepath.Join(dir, "plate.dxf")

	out, err := runCLI(t, "export", opd, dxfPath, "r2018")
	if err != nil {
		t.Fatalf("export dxf: %v", err)
	}
	if !strings.Contains(out, "curves") || !strings.Contains(out, "r2018") {
		t.Errorf("export output = %q, want curve count + r2018", out)
	}
	data, err := os.ReadFile(dxfPath)
	if err != nil {
		t.Fatalf("expected %s on disk: %v", dxfPath, err)
	}
	if !strings.Contains(string(data), "AC1032") || !strings.Contains(string(data), "\nLINE\n") {
		t.Errorf("exported DXF missing AC1032 header or LINE entity")
	}
}
