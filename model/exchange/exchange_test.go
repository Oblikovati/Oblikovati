// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// TestMeshExchangeGLTFCapability: Formats includes glTF; CanImport excludes it
// (export-only honesty, R1-2); CanExport includes it.
func TestMeshExchangeGLTFCapability(t *testing.T) {
	me := MeshExchange{}
	found := false
	for _, f := range me.Formats() {
		if f == types.FormatGLTF {
			found = true
		}
	}
	if !found {
		t.Error("Formats() does not include FormatGLTF")
	}
	if me.CanImport(types.FormatGLTF) {
		t.Error("CanImport(gltf) = true; v1 is export-only and must not over-promise")
	}
	if !me.CanExport(types.FormatGLTF) {
		t.Error("CanExport(gltf) = false; the exporter ships in v1")
	}
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
	// The fixture is a 5-unit cube; mesh files are read as millimetres, so it imports
	// as a 0.5 cm cube (0.125 cm³) and the OBJ round-trip preserves that.
	if want := 0.125; got < want-1e-4 || got > want+1e-4 {
		t.Errorf("round-trip volume = %v, want %v cm³", got, want)
	}
}

// TestExportFromGLTFWritesValidGLB: MeshExchange{}.ExportFrom with FormatGLTF
// delegates to the canonical Export path and writes a valid GLB (CHG2-2).
func TestExportFromGLTFWritesValidGLB(t *testing.T) {
	dir := t.TempDir()
	part := compdef.NewPartComponentDefinition()
	src := writeCubeSTL(t, dir, 4)
	if _, err := (MeshExchange{}).ImportInto(part, src, types.FormatSTL); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	dst := filepath.Join(dir, "box.glb")
	res, err := (MeshExchange{}).ExportFrom(part, dst, types.FormatGLTF, types.ResolutionHigh)
	if err != nil {
		t.Fatalf("ExportFrom glb: %v", err)
	}
	if res.TriangleCount == 0 {
		t.Errorf("triangle count = 0, want > 0")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read %s: %v", dst, err)
	}
	if len(data) < 12 || string(data[0:4]) != "glTF" {
		t.Fatalf("exported file is not a GLB: %d bytes, magic %q", len(data), data[0:4])
	}
}

// TestExportFromGLTFRejectsGLTFDestination: MeshExchange{}.ExportFrom with
// FormatGLTF to a .gltf path is a typed error naming .glb — the delegation
// enforces the canonical .glb-only contract from the second entry point
// (CHG2-2).
func TestExportFromGLTFRejectsGLTFDestination(t *testing.T) {
	dir := t.TempDir()
	part := compdef.NewPartComponentDefinition()
	src := writeCubeSTL(t, dir, 4)
	if _, err := (MeshExchange{}).ImportInto(part, src, types.FormatSTL); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	dst := filepath.Join(dir, "box.gltf")
	_, err := (MeshExchange{}).ExportFrom(part, dst, types.FormatGLTF, types.ResolutionHigh)
	if err == nil || !strings.Contains(err.Error(), ".glb") {
		t.Fatalf("ExportFrom .gltf err = %v, want a typed error naming .glb", err)
	}
	if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
		t.Errorf("ExportFrom(.gltf) created the destination despite the rejection")
	}
}

// TestExportFromSTLWritesAtomically: MeshExchange{}.ExportFrom for STL writes
// through the same atomic writeExportFile helper as Export — a pre-existing
// destination is replaced via temp+rename and no temp files are left behind
// (CHG2-3).
func TestExportFromSTLWritesAtomically(t *testing.T) {
	dir := t.TempDir()
	part := compdef.NewPartComponentDefinition()
	src := writeCubeSTL(t, dir, 4)
	if _, err := (MeshExchange{}).ImportInto(part, src, types.FormatSTL); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	dst := filepath.Join(dir, "box.stl")
	if err := os.WriteFile(dst, []byte("stale bytes"), 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	res, err := (MeshExchange{}).ExportFrom(part, dst, types.FormatSTL, types.ResolutionHigh)
	if err != nil {
		t.Fatalf("ExportFrom stl: %v", err)
	}
	if res.TriangleCount == 0 {
		t.Errorf("triangle count = 0, want > 0")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if len(data) < 84 || string(data[0:5]) != "solid" && data[0] != 0 {
		t.Errorf("destination not replaced with STL content: %d bytes", len(data))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %q after successful ExportFrom", e.Name())
		}
	}
}

// TestExportFromSTLWriteFailureLeavesDestinationUntouched: a write failure
// through ExportFrom (the rename over a read-only destination is denied on
// Windows) must leave a pre-existing destination byte-for-byte unchanged and
// return a typed error (CHG2-3 — the atomic write applies to the second
// public entry point too). On POSIX, rename over a read-only file succeeds,
// so the failure injection is Windows-only.
func TestExportFromSTLWriteFailureLeavesDestinationUntouched(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("rename over a read-only destination succeeds on POSIX; failure injection is Windows-only (CHG-6 limitation)")
	}
	dir := t.TempDir()
	part := compdef.NewPartComponentDefinition()
	src := writeCubeSTL(t, dir, 4)
	if _, err := (MeshExchange{}).ImportInto(part, src, types.FormatSTL); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	dst := filepath.Join(dir, "box.stl")
	known := []byte("pre-existing destination bytes")
	if err := os.WriteFile(dst, known, 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if err := os.Chmod(dst, 0o444); err != nil {
		t.Fatalf("make destination read-only: %v", err)
	}
	defer os.Chmod(dst, 0o644)

	_, err := (MeshExchange{}).ExportFrom(part, dst, types.FormatSTL, types.ResolutionHigh)
	if err == nil {
		t.Fatal("ExportFrom succeeded; want a typed write error")
	}
	if !strings.Contains(err.Error(), "rename over") {
		t.Errorf("err = %q, want a typed rename error", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != string(known) {
		t.Errorf("destination changed: got %q, want %q", got, known)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %q after failed ExportFrom", e.Name())
		}
	}
}
