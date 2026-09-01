// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

func TestPrimitiveVolumes(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	block, err := brep.SolidBlock(math.P3(1, 1, 1), math.P3(3, 4, 6), "b")
	if err != nil {
		t.Fatal(err)
	}
	if v := query.BodyGeometryProperties(block, q).Volume; stdmath.Abs(v-30) > 1e-9 {
		t.Errorf("block volume %g want 30", v)
	}
	cyl, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 4), 2, 2, "c")
	if err != nil {
		t.Fatal(err)
	}
	if v := query.BodyGeometryProperties(cyl, q).Volume; stdmath.Abs(v-16*stdmath.Pi)/(16*stdmath.Pi) > 0.01 {
		t.Errorf("cylinder volume %g want %g", v, 16*stdmath.Pi)
	}
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 3), 2, 0, "k")
	if err != nil {
		t.Fatal(err)
	}
	if v := query.BodyGeometryProperties(cone, q).Volume; stdmath.Abs(v-4*stdmath.Pi)/(4*stdmath.Pi) > 0.01 {
		t.Errorf("cone volume %g want %g", v, 4*stdmath.Pi)
	}
	sph, err := brep.SolidSphere(math.P3(0, 0, 0), 2, "s")
	if err != nil {
		t.Fatal(err)
	}
	want := 4.0 / 3 * stdmath.Pi * 8
	// The chord-tolerance mesh is inscribed: at 0.05 on r=2 the deficit is ~1.7%.
	if v := query.BodyGeometryProperties(sph, q).Volume; stdmath.Abs(v-want)/want > 0.02 {
		t.Errorf("sphere volume %g want %g", v, want)
	}
	if !sph.IsSolid() || !sph.Shells()[0].IsClosed() {
		t.Error("sphere must be a closed solid")
	}
	tor, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1, "t")
	if err != nil {
		t.Fatal(err)
	}
	wantT := 2 * stdmath.Pi * stdmath.Pi * 5 * 1
	if v := query.BodyGeometryProperties(tor, q).Volume; stdmath.Abs(v-wantT)/wantT > 0.02 {
		t.Errorf("torus volume %g want %g", v, wantT)
	}
}
