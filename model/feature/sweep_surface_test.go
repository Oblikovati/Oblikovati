// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSweepSurfaceMakesOpenSheet: a 2×2 square (perimeter 8) swept along a straight 5-long path with
// the Surface operation (kSurfaceOperation, #1858) builds an OPEN swept sheet — the profile boundary
// swept, no end caps — via sweepTool → sweptShell. For a straight path the four side faces are exact
// planes, so the area is perimeter × length = 40 exactly.
func TestSweepSurfaceMakesOpenSheet(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(0, 0, 5)),
	}, false)
	pf := NewSweepFeatures(fs).Add(centeredSquareOn(sketch.XYPlane(), 1), 0, path, nil, ops.Surface)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("surface sweep went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if body.IsSolid() {
		t.Error("surface-operation sweep should be an OPEN sheet, got a solid")
	}
	want := 8.0 * 5.0 // profile perimeter 8 × path length 5
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Area; relErr(got, want) > 0.02 {
		t.Errorf("swept sheet area = %g, want ≈%g (perimeter × length)", got, want)
	}
}
