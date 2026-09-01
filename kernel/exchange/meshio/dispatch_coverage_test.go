// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func TestDispatchDecodeImportAndExportErrors(t *testing.T) {
	if got := stlErr("facet", "nan").Error(); got != "STL: facet \"nan\"" {
		t.Fatalf("stlErr = %q", got)
	}
	if _, err := Decode(types.ExchangeFormat("ply"), nil); err == nil {
		t.Fatal("Decode accepted unsupported format")
	}
	if _, _, err := ImportBody(types.ExchangeFormat("ply"), nil, "import:bad", DefaultWeldTolerance, exchange.TranslationOptions{}); err == nil {
		t.Fatal("ImportBody accepted unsupported format")
	}
	body, _, err := SolidOrSurface(cubeSoup(1), "cube", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if _, _, err := ExportBody(types.ExchangeFormat("ply"), body, types.ResolutionLow); err == nil {
		t.Fatal("ExportBody accepted unsupported format")
	}
	if _, _, err := ExportBodies(types.ExchangeFormat("ply"), []*topo.Body{body}, types.ResolutionLow, exchange.TranslationOptions{}); err == nil {
		t.Fatal("ExportBodies accepted unsupported format")
	}
	if raw, err := Decode(types.FormatOBJ, []byte("v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")); err != nil || raw.TriangleCount() != 1 {
		t.Fatalf("Decode OBJ triangles = %d, %v", raw.TriangleCount(), err)
	}
	data := EncodeBinarySTL(tessellateOne(body))
	imported, warns, err := ImportBody(types.FormatSTL, data, "import:stl", DefaultWeldTolerance, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("ImportBody STL: %v", err)
	}
	if !imported.IsSolid() || len(warns) != 0 {
		t.Fatalf("ImportBody STL solid=%v warnings=%v", imported.IsSolid(), warns)
	}
	threeMF, err := Encode3MF(tessellateOne(body), "millimeter")
	if err != nil {
		t.Fatalf("Encode3MF: %v", err)
	}
	if raw, err := Decode(types.Format3MF, threeMF); err != nil || raw.TriangleCount() == 0 {
		t.Fatalf("Decode 3MF triangles = %d, %v", raw.TriangleCount(), err)
	}
}

func TestExportBodiesMergesMeshes(t *testing.T) {
	bodyA, _, err := SolidOrSurface(cubeSoup(1), "cube:a", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface A: %v", err)
	}
	bodyB, _, err := SolidOrSurface(cubeSoup(2), "cube:b", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface B: %v", err)
	}
	data, tris, err := ExportBodies(types.FormatOBJ, []*topo.Body{bodyA, bodyB}, types.ResolutionLow, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("ExportBodies OBJ: %v", err)
	}
	if tris != 24 {
		t.Fatalf("merged triangle count = %d, want 24", tris)
	}
	if len(data) == 0 {
		t.Fatal("ExportBodies OBJ returned empty data")
	}
	if data, tris, err := ExportBodies(types.FormatSTL, []*topo.Body{bodyA}, types.ResolutionLow, exchange.TranslationOptions{}); err != nil || tris != 12 || len(data) == 0 {
		t.Fatalf("ExportBodies STL len=%d tris=%d err=%v", len(data), tris, err)
	}
	if data, tris, err := ExportBodies(types.Format3MF, []*topo.Body{bodyA}, types.ResolutionLow, exchange.TranslationOptions{}); err != nil || tris != 12 || len(data) == 0 {
		t.Fatalf("ExportBodies 3MF len=%d tris=%d err=%v", len(data), tris, err)
	}
}

func TestReverseFacesCopiesAndFlipsWinding(t *testing.T) {
	cage := subd.Mesh{
		Verts: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		Faces: [][]int{{0, 1, 2}},
	}
	reversed := reverseFaces(cage)
	if got := reversed.Faces[0]; got[0] != 2 || got[1] != 1 || got[2] != 0 {
		t.Fatalf("reversed face = %v, want [2 1 0]", got)
	}
	if cage.Faces[0][0] != 0 || cage.Faces[0][2] != 2 {
		t.Fatalf("reverseFaces mutated input: %v", cage.Faces[0])
	}
}
