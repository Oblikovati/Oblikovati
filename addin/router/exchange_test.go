// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// cubeSTLFixture writes a watertight cube STL to a temp dir and returns its path.
func cubeSTLFixture(t *testing.T, s float64) string {
	t.Helper()
	body, _, err := meshio.SolidOrSurface(routerCubeSoup(s), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build cube: %v", err)
	}
	path := filepath.Join(t.TempDir(), "cube.stl")
	if err := os.WriteFile(path, meshio.EncodeBinarySTL(body, ops.DefaultQuality()), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func routerCubeSoup(s float64) meshio.RawMesh {
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

func TestRouterImportThenExportRoundTrip(t *testing.T) {
	r, s := emptyPartSession(t)
	stl := cubeSTLFixture(t, 4)

	req, _ := json.Marshal(wire.ImportRequest{Path: stl, Format: string(types.FormatSTL)})
	resp, err := r.Handle(s, wire.MethodDocumentsImport, req)
	if err != nil {
		t.Fatalf("documents.import: %v", err)
	}
	var ir wire.ImportResponse
	if err := json.Unmarshal(resp, &ir); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if ir.BodyCount != 1 || !ir.Solid {
		t.Fatalf("import response = %+v, want one solid body", ir)
	}

	out := filepath.Join(t.TempDir(), "out.3mf")
	ereq, _ := json.Marshal(wire.ExportRequest{Path: out, Format: string(types.Format3MF), Resolution: string(types.ResolutionHigh)})
	eresp, err := r.Handle(s, wire.MethodDocumentsExport, ereq)
	if err != nil {
		t.Fatalf("documents.export: %v", err)
	}
	var er wire.ExportResponse
	if err := json.Unmarshal(eresp, &er); err != nil {
		t.Fatalf("decode export response: %v", err)
	}
	if er.TriangleCount <= 0 {
		t.Fatalf("export wrote %d triangles, want > 0", er.TriangleCount)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("export did not write file: %v", err)
	}
}
