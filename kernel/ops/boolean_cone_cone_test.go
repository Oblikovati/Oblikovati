// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// Cone–cone intersection through Boolean (M2 Phase 2, Oblikovati/Oblikovati#1335). A cone (a tapered rod)
// crossing a fatter cone must Intersect to the exact analytic solid — the rod-cone band plus two fat-cone
// lens caps — its volume matching the analytic cone∩cone, not triangle-soup CSG.

// coneConeIntersectVolume is the volume of the thin frustum (axis x, apex x=−19.714, half-angle atan
// 0.7/12, radius 0.8→1.5 over x∈[−6,6]) ∩ the fat frustum (axis z, apex z=−18, half-angle atan 2/12, radius
// 2→4 over z∈[−6,6]). The two cone constraints couple all three coordinates (unlike a cone∩cylinder slab
// clip), so the oracle is a deterministic midpoint sum over a world-space grid bounding the thin rod.
func coneConeIntersectVolume() float64 {
	const thinTan, fatTan = 0.7 / 12.0, 2.0 / 12.0
	thinApexX, fatApexZ := -6.0-0.8/thinTan, -6.0-2.0/fatTan
	const nx, nyz = 360, 150
	lo, hi, yz := -6.0, 6.0, 1.6
	dx, dyz := (hi-lo)/nx, 2*yz/nyz
	cnt := 0
	for i := 0; i < nx; i++ {
		x := lo + (float64(i)+0.5)*dx
		rThin := (x - thinApexX) * thinTan
		for j := 0; j < nyz; j++ {
			y := -yz + (float64(j)+0.5)*dyz
			for k := 0; k < nyz; k++ {
				z := -yz + (float64(k)+0.5)*dyz
				rFat := (z - fatApexZ) * fatTan
				if y*y+z*z <= rThin*rThin && x*x+y*y <= rFat*rFat {
					cnt++
				}
			}
		}
	}
	return float64(cnt) * dx * dyz * dyz
}

// TestBooleanIntersectConeCone crosses a narrow frustum through a fatter frustum and checks the result is the
// exact three-face analytic solid (all cone faces) with the analytic cone∩cone volume.
func TestBooleanIntersectConeCone(t *testing.T) {
	thin, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")

	res, err := ops.Boolean(ops.Intersect, thin, fat)
	if err != nil {
		t.Fatalf("Boolean(Intersect cone∩cone): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cone∩cone is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cone); !ok {
			t.Errorf("face surface %T is not a cone (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 3 {
		t.Errorf("cone∩cone has %d faces, want 3 (rod band + 2 fat-cone lens caps)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneConeIntersectVolume()
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("cone∩cone volume %.4f, want %.4f (analytic) — rel %.4f > 3%%", got, want, rel)
	}
}
