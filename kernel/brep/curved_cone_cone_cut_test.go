// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Cone–cone cut assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). Drilling a fat cone with a crossing rod
// cone, and the two rod-cone stubs of cone − fat, must weld into watertight analytic solids — the same shapes
// as the cone–cylinder drill/stubs, only the fat is a cone too (its holed wall is a holed CONE wall). Volumes
// are checked through ops_test; here the concern is the watertight topology and that the surfaces stay
// analytic.

// TestConeConeCutDrillsFat drills a radius-2→4 fat cone with a crossing radius-0.8→1.5 rod cone and checks the
// result is a watertight four-face solid: two fat-cone caps (planes), the holed fat-cone wall, and the rod
// cone tunnel band.
func TestConeConeCutDrillsFat(t *testing.T) {
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")

	res, ok := ConeConeCut(fat, thin, nil)
	if !ok {
		t.Fatal("cone drill of cone declined; want a four-face solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 2 || cyls != 0 || planes != 2 {
		t.Errorf("drilled fat cone got %d cone + %d cyl + %d plane faces, want 2 (tunnel + holed wall) + 2 (caps)",
			cones, cyls, planes)
	}
}

// TestConeConeCutRodMinusFatStubs subtracts the fat cone from the crossing rod cone and checks the result is
// the two disconnected tapered stubs (a two-shell solid).
func TestConeConeCutRodMinusFatStubs(t *testing.T) {
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")

	res, ok := ConeConeCut(thin, fat, nil)
	if !ok {
		t.Fatal("rod cone − fat cone declined; want two tapered stubs")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 2 {
		t.Errorf("rod − fat has %d shells, want 2 (a disconnected stub each side)", n)
	}
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 4 || cyls != 0 || planes != 2 {
		t.Errorf("rod − fat got %d cone + %d cyl + %d plane faces, want 4 (2 stub bands + 2 lens caps) + 2 (rod end caps)",
			cones, cyls, planes)
	}
}

// TestConeConeCutConeCylinderDefer: a cone and a cylinder are the cone–cylinder case, not this one.
func TestConeConeCutConeCylinderDefer(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeConeCut(cyl, cone, nil); ok {
		t.Error("a cone and a cylinder should defer from the cone–cone cut (ok=false)")
	}
}
