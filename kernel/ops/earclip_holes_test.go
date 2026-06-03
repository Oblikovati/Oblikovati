// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// signedTriArea2 returns twice the signed area of triangle abc.
func signedTriArea2(a, b, c math.Point2) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (c.X-a.X)*(b.Y-a.Y)
}

// TestMergeHolesTriangulatesAnnulus bridges a square hole into a square outer and checks
// the ear-clipped triangles cover exactly the annulus area (outer 16 − hole 4 = 12), all
// wound CCW. Regression for the tessellator that ignored holes and over-counted volume on
// frame faces away from the origin plane (chained-difference +X pocket; see kernel/brep).
func TestMergeHolesTriangulatesAnnulus(t *testing.T) {
	outer := []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}} // CCW, 16
	hole := []math.Point2{{X: 1, Y: 1}, {X: 1, Y: 3}, {X: 3, Y: 3}, {X: 3, Y: 1}}  // 4
	o3 := lift(outer)
	merged2, merged3 := mergeHoles(outer, o3, [][]math.Point2{hole}, [][]math.Point3{lift(hole)})
	if len(merged2) != len(merged3) {
		t.Fatalf("merged 2D/3D length mismatch: %d vs %d", len(merged2), len(merged3))
	}
	var area float64
	tris := earClip(merged2)
	if len(tris) == 0 {
		t.Fatal("ear-clipping a bridged annulus produced no triangles")
	}
	for _, tr := range tris {
		a := signedTriArea2(merged2[tr[0]], merged2[tr[1]], merged2[tr[2]])
		if a <= 0 {
			t.Errorf("triangle %v not CCW (2·area = %g)", tr, a)
		}
		area += a / 2
	}
	if stdmath.Abs(area-12) > 1e-9 {
		t.Errorf("annulus triangulated area = %g, want 12", area)
	}
}

// lift trivially embeds 2D points at z=0 for the lockstep 3D loop.
func lift(p []math.Point2) []math.Point3 {
	out := make([]math.Point3, len(p))
	for i, q := range p {
		out[i] = math.P3(q.X, q.Y, 0)
	}
	return out
}
