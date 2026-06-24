// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
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

// coneConeBodies builds the test pair: the radius-0.8→1.5 rod cone (axis x) and the radius-2→4 fat cone
// (axis z) it crosses.
func coneConeBodies() (thin, fat *topo.Body) {
	thin, _ = brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ = brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	return thin, fat
}

// TestBooleanCutConeConeDrillsFat drills the fat cone with the crossing rod cone (fat − cone): the exact
// analytic solid (two fat-cone caps, the holed fat-cone wall, the rod-cone tunnel) whose volume is the fat
// cone minus the cone∩cone.
func TestBooleanCutConeConeDrillsFat(t *testing.T) {
	thin, fat := coneConeBodies()
	res, err := ops.Boolean(ops.Cut, fat, thin)
	if err != nil {
		t.Fatalf("Boolean(Cut fatCone−rodCone): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("drilled fat cone is not a valid closed manifold solid: %+v", v)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneFrustumVolume(2, 4, 12) - coneConeIntersectVolume()
	// 4%: the faceted fat-cone wall/caps and the tapered tunnel inscribe their curvature, so the meshed
	// volume runs under the analytic fat − tunnel (the B-rep is exact; this bounds the property-mesh error).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("drilled fat-cone volume %.4f, want %.4f (fat − cone∩cone) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutConeConeStubs subtracts the fat cone from the rod cone (cone − fat): the two disconnected
// tapered stubs (a two-shell solid) whose total volume is the rod cone minus the cone∩cone.
func TestBooleanCutConeConeStubs(t *testing.T) {
	thin, fat := coneConeBodies()
	res, err := ops.Boolean(ops.Cut, thin, fat)
	if err != nil {
		t.Fatalf("Boolean(Cut rodCone−fatCone): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("rod − fat (cones) is not a valid closed manifold solid: %+v", v)
	}
	if n := len(res.Shells()); n != 2 {
		t.Errorf("rod − fat (cones) has %d shells, want 2 (a disconnected stub each side)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneFrustumVolume(0.8, 1.5, 12) - coneConeIntersectVolume()
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("rod − fat (cones) volume %.4f, want %.4f (rod − cone∩cone) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanJoinConeCone joins the fat cone and the crossing rod cone (fat ∪ cone): the connected analytic
// solid (fat-cone caps, holed fat-cone wall, a tapered stub each side) whose volume is fat + rod − the
// cone∩cone.
func TestBooleanJoinConeCone(t *testing.T) {
	thin, fat := coneConeBodies()
	res, err := ops.Boolean(ops.Join, fat, thin)
	if err != nil {
		t.Fatalf("Boolean(Join fatCone∪rodCone): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("joined cone∪cone is not a valid closed manifold solid: %+v", v)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneFrustumVolume(2, 4, 12) + coneFrustumVolume(0.8, 1.5, 12) - coneConeIntersectVolume()
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("joined cone∪cone volume %.4f, want %.4f (fat + rod − cone∩cone) — rel %.4f > 4%%", got, want, rel)
	}
}
