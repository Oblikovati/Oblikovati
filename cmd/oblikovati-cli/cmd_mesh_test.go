// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// writeCLICubeSTL writes a watertight cube STL fixture for the CLI tests.
func writeCLICubeSTL(t *testing.T, dir string, s float64) string {
	t.Helper()
	body, _, err := meshio.SolidOrSurface(cliCubeSoup(s), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build cube: %v", err)
	}
	path := filepath.Join(dir, "cube.stl")
	if err := os.WriteFile(path, meshio.EncodeBinarySTL(body, ops.DefaultQuality()), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func cliCubeSoup(s float64) meshio.RawMesh {
	v := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s) }
	p := [8]math.Point3{
		v(0, 0, 0), v(1, 0, 0), v(1, 1, 0), v(0, 1, 0),
		v(0, 0, 1), v(1, 0, 1), v(1, 1, 1), v(0, 1, 1),
	}
	quads := [6][4]int{{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {2, 3, 7, 6}, {1, 2, 6, 5}, {0, 4, 7, 3}}
	var m meshio.RawMesh
	for _, q := range quads {
		m.AddTriangle(p[q[0]], p[q[1]], p[q[2]])
		m.AddTriangle(p[q[0]], p[q[2]], p[q[3]])
	}
	return m
}

func TestCLIImportThenExportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	stl := writeCLICubeSTL(t, dir, 4)
	obk := filepath.Join(dir, "part.obk")

	out, err := runCLI(t, "import", stl, obk)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "as a solid") {
		t.Errorf("import output = %q, want it to report a solid", out)
	}
	if _, statErr := os.Stat(obk); statErr != nil {
		t.Fatalf("expected %s on disk: %v", obk, statErr)
	}

	exportObj := filepath.Join(dir, "out.obj")
	eout, err := runCLI(t, "export", obk, exportObj, "high")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(eout, "triangles") || !strings.Contains(eout, "high resolution") {
		t.Errorf("export output = %q, want triangle count + high resolution", eout)
	}
	if _, statErr := os.Stat(exportObj); statErr != nil {
		t.Fatalf("expected exported %s on disk: %v", exportObj, statErr)
	}
}

func TestCLIImportUnknownExtensionErrors(t *testing.T) {
	_, err := runCLI(t, "import", "model.foo", filepath.Join(t.TempDir(), "p.obk"))
	if err == nil {
		t.Fatalf("expected an unsupported-extension error")
	}
}
