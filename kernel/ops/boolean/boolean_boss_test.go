// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Cylindrical boss union exactness (M2 Phase 3, Oblikovati/Oblikovati#1336 — the coplanar curved+planar
// overlap case). A cylinder seated flush on a plate face (a boss/spigot) unions to an exact solid: the
// seat face's base disk is coplanar with the boss base — the coplanar overlap the general boolean faceted
// through CSG. Through ops.Boolean the result must keep an analytic cylinder face and have the exact
// volume plate + πr²h.

// TestCylindricalBossUnionIsExact seats a radius-1.5, height-3 boss on a 10×10×2 plate and checks the
// union keeps a cylinder face and has volume 200 + π·1.5²·3.
func TestCylindricalBossUnionIsExact(t *testing.T) {
	t.Parallel()
	plate, err := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	boss, err := brep.SolidCylinder(math.P3(0, 0, 2), math.V3(0, 0, 1), 1.5, 3)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}

	res, err := ops.Boolean(ops.Join, plate, boss)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("boss union not a watertight solid: %+v", r)
	}
	if !hasCylinderFace(res) {
		t.Error("boss union has no geom.Cylinder face — it fell back to faceted CSG")
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := 200.0 + stdmath.Pi*1.5*1.5*3 // plate 10×10×2 + boss πr²h
	if rel := stdmath.Abs(got-want) / want; rel > 0.01 {
		t.Errorf("boss union volume %.4f, want %.4f (rel %.4f > 0.01); union is not exact", got, want, rel)
	}
}
