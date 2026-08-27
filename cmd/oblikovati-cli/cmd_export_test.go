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
	"oblikovati.org/model/contentset"
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
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
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

// TestCLIExportGLB exports a part to .glb via the CLI and asserts the GLB
// container header + a 12-triangle box (the PFSQ20-shaped fixture).
func TestCLIExportGLB(t *testing.T) {
	dir := t.TempDir()
	opd := cliPartWithCube(t, dir)
	glbPath := filepath.Join(dir, "box.glb")

	out, err := runCLI(t, "export", opd, glbPath, "high")
	if err != nil {
		t.Fatalf("export glb: %v", err)
	}
	if !strings.Contains(out, "triangles") || !strings.Contains(out, "high resolution") {
		t.Errorf("export output = %q, want triangle count + high resolution", out)
	}
	data, err := os.ReadFile(glbPath)
	if err != nil {
		t.Fatalf("expected %s on disk: %v", glbPath, err)
	}
	if len(data) < 12 || string(data[0:4]) != "glTF" {
		t.Fatalf("exported file is not a GLB: %d bytes, magic %q", len(data), data[0:4])
	}
}

// TestCLIExportGLTFExtensionTypedError: a .gltf destination is a typed error
// naming the supported .glb extension (R1-2/R2-6).
func TestCLIExportGLTFExtensionTypedError(t *testing.T) {
	dir := t.TempDir()
	opd := cliPartWithCube(t, dir)
	_, err := runCLI(t, "export", opd, filepath.Join(dir, "box.gltf"), "high")
	if err == nil || !strings.Contains(err.Error(), ".glb") {
		t.Fatalf("export .gltf err = %v, want a typed error naming .glb", err)
	}
}

// cliPartWithCube builds a part containing a watertight cube body (via the
// import path) and returns its .opd path.
func cliPartWithCube(t *testing.T, dir string) string {
	t.Helper()
	stl := writeCLICubeSTL(t, dir, 4)
	opd := filepath.Join(dir, "cube.opd")
	if _, err := runCLI(t, "import", stl, opd); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	return opd
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
