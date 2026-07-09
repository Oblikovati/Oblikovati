// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestCoilSurfaceMakesOpenSheet: a 1×1 profile coiled 3 turns at pitch 2 about Z with the Surface
// operation (kSurfaceOperation, #1858) builds an OPEN coiled sheet — the profile boundary swept
// along the helix, no end caps — via coilTool → sweptShell (not a solid). The key correctness is the
// open-sheet classification, cross-checked against the coil SOLID: the sheet is the solid's lateral
// surface, so solid area = sheet area + the two profile end caps (≈2 for the unit profile).
func TestCoilSurfaceMakesOpenSheet(t *testing.T) {
	mk := func(op ops.PartFeatureOperation) *topo.Body {
		fs := NewPartFeatures(nil)
		pf := NewCoilFeatures(fs).AddDefinition(&CoilDefinition{
			Sketch: coilProfileSketch(), Axis: zWorkAxis(),
			Pitch: angleConst(2), Revolutions: angleConst(3), Operation: op,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("coil (%v) went sick: %+v", op, pf.Health())
		}
		return fs.Result()[0]
	}
	area := func(b *topo.Body) float64 { return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Area }

	sheet, solid := mk(ops.Surface), mk(ops.NewBody)
	if sheet.IsSolid() {
		t.Error("surface-operation coil should be an OPEN sheet, got a solid")
	}
	sheetArea := area(sheet)
	if sheetArea < 180 || sheetArea > 320 {
		t.Errorf("coiled sheet area = %g, out of the expected regression band (≈235)", sheetArea)
	}
	// The open sheet is the solid coil's lateral surface: solid area = sheet + the two end caps.
	// The 1×1 profile's two caps add ≈2, so the difference matches within a small margin.
	if diff := area(solid) - sheetArea; relErr(diff, 2.0) > 0.5 {
		t.Errorf("solid coil area − sheet = %g, want ≈2 (the two profile end caps)", diff)
	}
}
