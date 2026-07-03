// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/math"
)

// Mesh-import warning tests (M40 audit S3, #1638): every weld/validation decision that discards
// or degrades geometry surfaces as an ImportResult warning naming the offending entity and value,
// matching the DWG decoder's warn-and-continue policy — no silent drops.

// TestSolidOrSurfaceWarnsOnDroppedDegenerateTriangles: a soup with degenerate triangles imports
// the good geometry and reports how many triangles were dropped, instead of silently thinning.
func TestSolidOrSurfaceWarnsOnDroppedDegenerateTriangles(t *testing.T) {
	raw := cubeSoup(2)
	// Two degenerate triangles: a repeated corner and three collinear-welded points.
	raw.AddTriangle(math.P3(0, 0, 0), math.P3(0, 0, 0), math.P3(2, 0, 0))
	raw.AddTriangle(math.P3(0, 0, 0), math.P3(0, 0, 0), math.P3(0, 2, 0))
	body, warns, err := SolidOrSurface(raw, "import:test#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("cube with extra degenerate triangles should still weld into a solid; warns=%v", warns)
	}
	want := fmt.Sprintf("2 of %d triangles", raw.TriangleCount())
	if len(warns) != 1 || !strings.Contains(warns[0], want) || !strings.Contains(warns[0], "degenerate") {
		t.Errorf("warns = %v, want one naming %q dropped as degenerate", warns, want)
	}
}

// TestSolidOrSurfaceCleanSoupHasZeroWarnings: the regression guard — a clean watertight soup
// imports without any warning (#1638).
func TestSolidOrSurfaceCleanSoupHasZeroWarnings(t *testing.T) {
	_, warns, err := SolidOrSurface(cubeSoup(2), "import:test#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("clean cube import warns = %v, want none", warns)
	}
}

// threeMFWithUnit assembles a minimal valid 3MF (ZIP + model XML) of one triangle, declaring the
// given unit spelling — a named fake for the on-disk container.
func threeMFWithUnit(t *testing.T, unit string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("3D/3dmodel.model")
	if err != nil {
		t.Fatal(err)
	}
	xml := fmt.Sprintf(`<?xml version="1.0"?>
<model unit=%q xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
 <resources><object id="1" type="model"><mesh>
  <vertices><vertex x="0" y="0" z="0"/><vertex x="10" y="0" z="0"/><vertex x="0" y="10" z="0"/></vertices>
  <triangles><triangle v1="0" v2="1" v3="2"/></triangles>
 </mesh></object></resources>
</model>`, unit)
	if _, err := w.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestImportBodyWarnsOnUnknown3MFUnit: an unrecognised 3MF unit spelling falls back to
// millimetres WITH a warning naming the offending value — no silent wrong-scale import (#1638).
func TestImportBodyWarnsOnUnknown3MFUnit(t *testing.T) {
	data := threeMFWithUnit(t, "parsec")
	_, warns, err := ImportBody("3mf", data, "import:3mf#0", 0, exchange.TranslationOptions{TargetUnitMM: 10})
	if err != nil {
		t.Fatalf("ImportBody: %v", err)
	}
	joined := strings.Join(warns, "; ")
	if !strings.Contains(joined, "parsec") || !strings.Contains(joined, "millimet") {
		t.Errorf("warns = %v, want one naming the unknown unit \"parsec\" and the millimetre fallback", warns)
	}
}

// TestImportBodyKnown3MFUnitNoUnitWarning: a spec unit spelling scales silently (the warning is
// only for the unknown-value fallback).
func TestImportBodyKnown3MFUnitNoUnitWarning(t *testing.T) {
	data := threeMFWithUnit(t, "meter")
	body, warns, err := ImportBody("3mf", data, "import:3mf#0", 0, exchange.TranslationOptions{TargetUnitMM: 10})
	if err != nil {
		t.Fatalf("ImportBody: %v", err)
	}
	for _, w := range warns {
		if strings.Contains(w, "unit") {
			t.Errorf("unexpected unit warning for a spec spelling: %q", w)
		}
	}
	// 10 m in file units = 1000 database cm — proof the declared unit was honored.
	if got := float64(body.RangeBox().Max.X); got != 1000 {
		t.Errorf("meter-unit vertex X = %v database units, want 1000", got)
	}
}
