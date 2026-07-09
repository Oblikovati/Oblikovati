// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// TestLoftSurfaceOpenSheet: a 4×4 section lofted to a 2×2 section with the Surface operation
// (kSurfaceOperation, #1858) skins an OPEN sheet — the lateral skin only, no end caps — via
// skinTool → skinShell (not a solid). Cross-checked against the solid loft: the sheet is the solid's
// lateral surface, so solid_area − sheet_area ≈ the two end-cap areas (4²+2² = 20).
func TestLoftSurfaceOpenSheet(t *testing.T) {
	mk := func(op ops.PartFeatureOperation) *topo.Body {
		fs := NewPartFeatures(nil)
		pf := NewLoftFeatures(fs).Add([]LoftSection{
			{Sketch: centeredSquareOn(sketch.XYPlane(), 2), ProfileIndex: 0},
			{Sketch: centeredSquareOn(planeAtZ(5), 1), ProfileIndex: 0},
		}, false, op)
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("loft (%v) went sick: %+v", op, pf.Health())
		}
		return fs.Result()[0]
	}
	area := func(b *topo.Body) float64 { return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Area }

	sheet, solid := mk(ops.Surface), mk(ops.NewBody)
	if sheet.IsSolid() {
		t.Error("surface-operation loft should be an OPEN sheet, got a solid")
	}
	if diff := area(solid) - area(sheet); relErr(diff, 20.0) > 0.05 {
		t.Errorf("solid loft area − sheet = %g, want ≈20 (the 4×4 + 2×2 end caps)", diff)
	}
}

// TestLoftSurfaceTubeShell: two annulus sections lofted with the Surface operation build an OPEN pipe
// surface — outer and inner walls, no annular end caps — via tubeShellLoops → tubeShell (#1858).
func TestLoftSurfaceTubeShell(t *testing.T) {
	fs := NewPartFeatures(nil)
	bottom, bi := annulusOn(sketch.XYPlane(), 4, 2)
	top, ti := annulusOn(planeAtZ(5), 3, 1)
	pf := NewLoftFeatures(fs).Add([]LoftSection{{Sketch: bottom, ProfileIndex: bi}, {Sketch: top, ProfileIndex: ti}}, false, ops.Surface)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("surface tube loft went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if body.IsSolid() {
		t.Error("surface-operation hollow loft should be an OPEN pipe shell, got a solid")
	}
	if a := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Area; a <= 0 {
		t.Error("expected a non-empty open pipe surface")
	}
}
