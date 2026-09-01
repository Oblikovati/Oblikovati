// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Coaxial cylinder union exactness (M2 Phase 3, Oblikovati/Oblikovati#1336, the coplanar/tangent overlap
// case). Two coaxial equal-radius cylinders that overlap union to ONE taller cylinder. Through ops.Boolean
// the result must be exact — an analytic cylinder face survives and the volume is πr²·(merged height),
// not the faceted CSG value (the case used to come back ~2.5% under with zero analytic faces).

// TestCoaxialCylinderUnionIsExact unions z∈[0,4] with z∈[3,7] (radius 2) and checks the result keeps a
// cylinder face and has the exact merged volume πr²·7.
func TestCoaxialCylinderUnionIsExact(t *testing.T) {
	t.Parallel()
	a, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder a: %v", err)
	}
	b, err := brep.SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder b: %v", err)
	}

	res, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("coaxial union not a watertight solid: %+v", r)
	}
	if !hasCylinderFace(res) {
		t.Error("coaxial union has no geom.Cylinder face — it fell back to faceted CSG")
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi * 2 * 2 * 7 // one cylinder r=2, merged height 7
	if rel := stdmath.Abs(got-want) / want; rel > 0.01 {
		t.Errorf("coaxial union volume %.4f, want %.4f (rel %.4f > 0.01); union is not exact", got, want, rel)
	}
}
