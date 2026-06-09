// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// cubeSoup returns a watertight 12-triangle triangle soup of an s-sided axis-aligned cube
// at the origin (vertices repeated per triangle, outward winding) — an independent fixture
// generator for these tests.
func cubeSoup(s float64) meshio.RawMesh {
	v := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s) }
	p := [8]math.Point3{
		v(0, 0, 0), v(1, 0, 0), v(1, 1, 0), v(0, 1, 0),
		v(0, 0, 1), v(1, 0, 1), v(1, 1, 1), v(0, 1, 1),
	}
	quads := [6][4]int{
		{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {2, 3, 7, 6}, {1, 2, 6, 5}, {0, 4, 7, 3},
	}
	var m meshio.RawMesh
	for _, q := range quads {
		m.AddTriangle(p[q[0]], p[q[1]], p[q[2]])
		m.AddTriangle(p[q[0]], p[q[2]], p[q[3]])
	}
	return m
}

// writeCubeSTL writes a watertight unit-cube STL of side s to a temp file and returns its
// path — an independent fixture for the import path (built via the kernel encoder from a
// welded cube body, so the test does not hand-author binary bytes).
func writeCubeSTL(t *testing.T, dir string, s float64) string {
	t.Helper()
	body, _, err := meshio.SolidOrSurface(cubeSoup(s), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build fixture cube: %v", err)
	}
	data := meshio.EncodeBinarySTL(body, ops.DefaultQuality())
	path := filepath.Join(dir, "cube.stl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestImportIntoMakesAWatertightMeshASolid(t *testing.T) {
	dir := t.TempDir()
	path := writeCubeSTL(t, dir, 4)
	part := compdef.NewPartComponentDefinition()

	res, err := MeshExchange{}.ImportInto(part, path, types.FormatSTL)
	if err != nil {
		t.Fatalf("ImportInto: %v", err)
	}
	if !res.Solid {
		t.Fatalf("watertight STL did not import as a solid; warnings=%v", res.Warnings)
	}
	bodies := part.SurfaceBodies().All()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("part has %d bodies; want one solid", len(bodies))
	}
	if r := ops.Validate(bodies[0]); !r.Valid {
		t.Fatalf("imported body is not valid: %v", r.Issues)
	}
}

// TestFeatureOnTopOfImportedBody is the headline requirement: an imported mesh becomes a
// real solid you can build a feature on. We import two cubes and boolean-cut one with the
// other (an overlapping subtraction), proving downstream modeling operates on the
// imported geometry.
func TestFeatureOnTopOfImportedBody(t *testing.T) {
	dir := t.TempDir()
	big := writeNamedCubeSTL(t, dir, "big.stl", 6)
	small := writeNamedCubeSTL(t, dir, "small.stl", 3)
	part := compdef.NewPartComponentDefinition()

	if _, err := (MeshExchange{}).ImportInto(part, big, types.FormatSTL); err != nil {
		t.Fatalf("import big: %v", err)
	}
	if _, err := (MeshExchange{}).ImportInto(part, small, types.FormatSTL); err != nil {
		t.Fatalf("import small: %v", err)
	}
	volBefore := totalVolume(part)

	// Cut body 0 (6³=216) with body 1 (3³=27, fully inside the corner) → a feature on top
	// of the imported solids.
	feature.NewModifyFeatures(part.Features()).AddCombine(0, 1, ops.Cut)
	part.Recompute()

	bodies := part.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("after cut: %d bodies, want 1 (the cut result)", len(bodies))
	}
	if !bodies[0].IsSolid() {
		t.Fatalf("cut result on imported body is not a solid")
	}
	volAfter := ops.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
	if volAfter >= volBefore {
		t.Errorf("cut did not remove material: before=%v after=%v", volBefore, volAfter)
	}
}

// writeNamedCubeSTL writes a named watertight cube STL to dir.
func writeNamedCubeSTL(t *testing.T, dir, name string, s float64) string {
	t.Helper()
	body, _, err := meshio.SolidOrSurface(cubeSoup(s), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build fixture cube: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, meshio.EncodeBinarySTL(body, ops.DefaultQuality()), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// totalVolume sums the part's body volumes.
func totalVolume(part *compdef.PartComponentDefinition) float64 {
	var v float64
	for _, b := range part.SurfaceBodies().All() {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

func TestExportThenImportRoundTripsACylinderVolume(t *testing.T) {
	dir := t.TempDir()
	part := compdef.NewPartComponentDefinition()
	src := writeCubeSTL(t, dir, 5)
	if _, err := (MeshExchange{}).ImportInto(part, src, types.FormatSTL); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	out := filepath.Join(dir, "exported.obj")
	if _, err := (MeshExchange{}).ExportFrom(part, out, types.FormatOBJ, types.ResolutionMedium); err != nil {
		t.Fatalf("ExportFrom: %v", err)
	}
	reimport := compdef.NewPartComponentDefinition()
	if _, err := (MeshExchange{}).ImportInto(reimport, out, types.FormatOBJ); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	got := totalVolume(reimport)
	if want := 125.0; got < want-1e-3 || got > want+1e-3 {
		t.Errorf("round-trip volume = %v, want %v", got, want)
	}
}
