// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// TestRibSurfaceMakesOpenSheet: a rib built with the Surface operation (kSurfaceOperation, #1858)
// produces the rib WALLS only — an open sheet, no end caps — via buildExtrusionShell(caps=false),
// rather than the capped prism. The open sheet is the solid rib's side walls, so a solid rib of the
// same band has strictly more area (its two band end caps).
func TestRibSurfaceMakesOpenSheet(t *testing.T) {
	t.Parallel()
	mk := func(op ops.PartFeatureOperation) *topo.Body {
		fs := NewPartFeatures(nil)
		pf := NewRibFeatures(fs).AddDefinition(&RibDefinition{
			Sketch: lineSketchOn(planeAtZ(3)), ProfileIndex: 0,
			Thickness: angleConst(0.5), Depth: angleConst(2), Operation: op,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("rib (%v) went sick: %+v", op, pf.Health())
		}
		return fs.Result()[0]
	}
	area := func(b *topo.Body) float64 { return query.BodyGeometryProperties(b, ops.DefaultQuality()).Area }

	sheet, solid := mk(ops.Surface), mk(ops.NewBody)
	if sheet.IsSolid() {
		t.Error("surface-operation rib should be an OPEN sheet, got a solid")
	}
	if s, so := area(sheet), area(solid); s <= 0 || so <= s {
		t.Errorf("rib sheet area %g should be > 0 and less than the solid rib area %g (which adds the two band caps)", s, so)
	}
}
