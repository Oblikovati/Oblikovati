// SPDX-License-Identifier: GPL-2.0-only

package exchange_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange"
	"oblikovati.org/model/feature"
)

// stepFixture is a hand-authored AP203 fixture shared with the kernel step tests.
func stepFixture(name string) string {
	return filepath.Join("..", "..", "kernel", "exchange", "step", "testdata", name)
}

// TestImportStepRoutesToBrep checks the unified dispatch routes a .step file through the STEP
// reader (not the mesh reader) and lands a valid solid in the part.
func TestImportStepRoutesToBrep(t *testing.T) {
	t.Parallel()
	part := compdef.NewPartComponentDefinition()
	res, err := exchange.Import(part, stepFixture("cube.step"), types.FormatSTEP)
	if err != nil {
		t.Fatalf("Import step: %v", err)
	}
	if res.BodyCount < 1 || !res.Solid {
		t.Fatalf("step import: bodyCount=%d solid=%v, want >=1 solid", res.BodyCount, res.Solid)
	}
	b := part.SurfaceBodies().Item(0)
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("imported step body not a valid solid: %+v", r)
	}
}

// TestStepRoundTripThroughDispatch imports a STEP solid, exports the part back to STEP, and
// re-imports — the body stays a valid solid with the same volume (the dispatch wires both ways).
func TestStepRoundTripThroughDispatch(t *testing.T) {
	t.Parallel()
	src := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(src, stepFixture("cube.step"), types.FormatSTEP); err != nil {
		t.Fatalf("import: %v", err)
	}
	v0 := ops.BodyGeometryProperties(src.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume

	out := filepath.Join(t.TempDir(), "roundtrip.step")
	if _, err := exchange.Export(src, out, types.FormatSTEP, ""); err != nil {
		t.Fatalf("export: %v", err)
	}
	back := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(back, out, types.FormatSTEP); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	b := back.SurfaceBodies().Item(0)
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("round-tripped step body not a valid solid: %+v", r)
	}
	if v1 := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; relErr(v0, v1) > 0.02 {
		t.Errorf("step round-trip volume changed: %.4f → %.4f", v0, v1)
	}
}

// TestFormatFromPath maps extensions to formats (case-insensitive), including STEP's two spellings.
func TestFormatFromPath(t *testing.T) {
	t.Parallel()
	cases := map[string]types.ExchangeFormat{
		"a.stl": types.FormatSTL, "b.OBJ": types.FormatOBJ, "c.3mf": types.Format3MF,
		"d.step": types.FormatSTEP, "e.STP": types.FormatSTEP, "f.DWG": types.FormatDWG,
	}
	for path, want := range cases {
		if got, ok := exchange.FormatFromPath(path); !ok || got != want {
			t.Errorf("FormatFromPath(%q) = %q,%v; want %q", path, got, ok, want)
		}
	}
	if _, ok := exchange.FormatFromPath("x.iges"); ok {
		t.Error("FormatFromPath accepted an unknown extension")
	}
}

func relErr(a, b float64) float64 {
	if a == 0 {
		return b
	}
	d := (a - b) / a
	if d < 0 {
		return -d
	}
	return d
}

// TestStepExportDeclaresDocumentUnit checks the dispatch threads the document's
// preferred length unit into the STEP export's declared unit (#146): the same
// part exports as inch or centimetre depending on its units.
func TestStepExportDeclaresDocumentUnit(t *testing.T) {
	t.Parallel()
	part := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(part, stepFixture("cube.step"), types.FormatSTEP); err != nil {
		t.Fatalf("import: %v", err)
	}
	for unit, marker := range map[string]string{
		"in": "CONVERSION_BASED_UNIT",
		"cm": "SI_UNIT(.CENTI.,.METRE.)",
		"mm": "SI_UNIT(.MILLI.,.METRE.)",
	} {
		if err := part.SetLengthUnit(unit); err != nil {
			t.Fatalf("SetLengthUnit(%q): %v", unit, err)
		}
		out := filepath.Join(t.TempDir(), "u.step")
		if _, err := exchange.Export(part, out, types.FormatSTEP, ""); err != nil {
			t.Fatalf("export %s: %v", unit, err)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read %s: %v", unit, err)
		}
		if !strings.Contains(string(data), marker) {
			t.Errorf("a %q document's STEP must declare %q", unit, marker)
		}
	}
}

// TestMeshExportImportThroughDispatch exercises the unified Import/Export for a mesh
// format (the dispatch's mesh branch) and confirms the size round-trips in centimetres.
func TestMeshExportImportThroughDispatch(t *testing.T) {
	t.Parallel()
	src := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(src, stepFixture("cube.step"), types.FormatSTEP); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	v0 := ops.BodyGeometryProperties(src.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	out := filepath.Join(t.TempDir(), "m.stl")
	if _, err := exchange.Export(src, out, types.FormatSTL, types.ResolutionHigh); err != nil {
		t.Fatalf("export stl: %v", err)
	}
	back := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(back, out, types.FormatSTL); err != nil {
		t.Fatalf("re-import stl: %v", err)
	}
	v1 := ops.BodyGeometryProperties(back.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	if relErr(v0, v1) > 0.05 {
		t.Errorf("mesh dispatch round-trip volume %.4f → %.4f", v0, v1)
	}
}

// partWithCube seeds a part with one solid cube body via the STEP import path.
func partWithCube(t *testing.T) *compdef.PartComponentDefinition {
	t.Helper()
	part := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(part, stepFixture("cube.step"), types.FormatSTEP); err != nil {
		t.Fatalf("seed import: %v", err)
	}
	return part
}

// TestExportGLTFRequiresGLBDestination: the direct model API rejects a
// FormatGLTF export to a non-.glb destination with a typed error naming the
// supported extension (CHG-2) — the CLI guard alone was not enough.
func TestExportGLTFRequiresGLBDestination(t *testing.T) {
	t.Parallel()
	part := partWithCube(t)
	dir := t.TempDir()
	for _, name := range []string{"box.gltf", "box.GLTF", "box.json"} {
		dst := filepath.Join(dir, name)
		_, err := exchange.Export(part, dst, types.FormatGLTF, types.ResolutionHigh)
		if err == nil {
			t.Fatalf("Export(%s) succeeded; want a typed .glb-required error", name)
		}
		if !strings.Contains(err.Error(), ".glb") || !strings.Contains(err.Error(), name) {
			t.Errorf("Export(%s) err = %q, want a typed error naming .glb and the path", name, err)
		}
		if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
			t.Errorf("Export(%s) created the destination despite the rejection", name)
		}
	}
}

// TestExportGLTFPassesThroughGLB: a .glb destination exports successfully
// through the direct model API (CHG-2 pass-through).
func TestExportGLTFPassesThroughGLB(t *testing.T) {
	t.Parallel()
	part := partWithCube(t)
	dst := filepath.Join(t.TempDir(), "box.glb")
	res, err := exchange.Export(part, dst, types.FormatGLTF, types.ResolutionHigh)
	if err != nil {
		t.Fatalf("Export glb: %v", err)
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

// TestExportPreservesDestinationMode: a successful export over a pre-existing
// 0644 destination leaves the destination 0644 — the temp file (0600 from
// os.CreateTemp) must be chmodded to the prior destination's mode before the
// rename (CHG3-3). Windows honors only the read-only bit of FileMode, so the
// strict 0644 assertion runs on non-Windows; on Windows the test asserts the
// write bits are set (the export must not leave the destination read-only).
func TestExportPreservesDestinationMode(t *testing.T) {
	t.Parallel()
	part := partWithCube(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "box.glb")
	if err := os.WriteFile(dst, []byte("stale bytes"), 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if _, err := exchange.Export(part, dst, types.FormatGLTF, types.ResolutionHigh); err != nil {
		t.Fatalf("Export glb: %v", err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	perm := info.Mode().Perm()
	if runtime.GOOS == "windows" {
		if perm&0o222 == 0 {
			t.Errorf("destination mode = %o, want write bits set (not read-only)", perm)
		}
		return
	}
	if perm != 0o644 {
		t.Errorf("destination mode = %o, want 0644 (prior destination mode preserved)", perm)
	}
}

// TestExportWriteFailureLeavesDestinationUntouched: a write failure (the
// rename over a read-only destination is denied on Windows) must leave a
// pre-existing destination byte-for-byte unchanged and return a typed error
// (CHG-6). On POSIX, rename over a read-only file succeeds, so the failure
// injection is Windows-only; the no-truncate-on-encode-error test covers the
// cross-platform guarantee.
func TestExportWriteFailureLeavesDestinationUntouched(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("rename over a read-only destination succeeds on POSIX; failure injection is Windows-only (CHG-6 limitation)")
	}
	part := partWithCube(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "box.glb")
	known := []byte("pre-existing destination bytes")
	if err := os.WriteFile(dst, known, 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if err := os.Chmod(dst, 0o444); err != nil {
		t.Fatalf("make destination read-only: %v", err)
	}
	defer func() { _ = os.Chmod(dst, 0o644) }()

	_, err := exchange.Export(part, dst, types.FormatGLTF, types.ResolutionHigh)
	if err == nil {
		t.Fatal("Export succeeded; want a typed write error")
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
	// No temp files may be left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %q after failed export", e.Name())
		}
	}
}

// TestExportEncodeErrorDoesNotTruncateDestination: an encoder error before any
// write (a part whose only body is empty) must leave a pre-existing
// destination untouched — the atomic write never opens the destination
// (CHG-6 no-truncate guarantee, cross-platform).
func TestExportEncodeErrorDoesNotTruncateDestination(t *testing.T) {
	t.Parallel()
	part := compdef.NewPartComponentDefinition()
	empty := topo.BodyFromShells(topo.NewLineage(topo.Tok("x", "y", 0)), false)
	feature.NewImportedBodies(part.Features()).Add(empty, "dummy-resource", "stl")
	part.Recompute()

	dir := t.TempDir()
	dst := filepath.Join(dir, "box.glb")
	known := []byte("pre-existing destination bytes")
	if err := os.WriteFile(dst, known, 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}

	_, err := exchange.Export(part, dst, types.FormatGLTF, types.ResolutionHigh)
	if err == nil {
		t.Fatal("Export succeeded; want the no-exportable-bodies encoder error")
	}
	if !strings.Contains(err.Error(), "no exportable bodies") {
		t.Errorf("err = %q, want the encoder's no-exportable-bodies error", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != string(known) {
		t.Errorf("destination changed: got %q, want %q", got, known)
	}
}

// TestExportSuccessPathRenamesOverDestination: a successful export replaces a
// pre-existing destination via temp+rename and leaves no temp files (CHG-6
// success path).
func TestExportSuccessPathRenamesOverDestination(t *testing.T) {
	t.Parallel()
	part := partWithCube(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "box.glb")
	if err := os.WriteFile(dst, []byte("stale bytes"), 0o644); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	if _, err := exchange.Export(part, dst, types.FormatGLTF, types.ResolutionHigh); err != nil {
		t.Fatalf("Export glb: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if len(data) < 12 || string(data[0:4]) != "glTF" {
		t.Fatalf("destination not replaced with a GLB: %d bytes, magic %q", len(data), data[0:4])
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %q after successful export", e.Name())
		}
	}
}
