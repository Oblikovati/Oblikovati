// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Cone–cone intersection assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). A cone crossing a fatter cone
// must weld into the same three-face shape as the crossing-cylinder and cone–cylinder intersections — a
// rod-cone band plus two fat-cone lens caps — watertight, with every face an analytic cone. Volume is checked
// through ops_test; here the concern is the watertight topology and that the surfaces stay analytic.

// TestConeConeIntersectThreeFaces crosses a narrow frustum through a fatter frustum and checks the result is a
// watertight three-face solid whose every face is a cone (the rod band and the two fat-cone lens caps).
func TestConeConeIntersectThreeFaces(t *testing.T) {
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")

	res, ok := ConeConeIntersect(thin, fat, nil)
	if !ok {
		t.Fatal("cone–cone intersection declined; want a three-face solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 3 || cyls != 0 || planes != 0 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 3 cones (rod band + 2 fat-cone lens caps)",
			cones, cyls, planes)
	}
}

// TestConeConeIntersectOrderIndependent: resolving works whichever cone is passed first.
func TestConeConeIntersectOrderIndependent(t *testing.T) {
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	if _, ok := ConeConeIntersect(fat, thin, nil); !ok {
		t.Error("cone–cone intersection should resolve with the fat cone passed first too")
	}
}

// TestConeConeIntersectConeCylinderDefer: a cone and a cylinder are the cone–cylinder case, not this one.
func TestConeConeIntersectConeCylinderDefer(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeConeIntersect(cone, cyl, nil); ok {
		t.Error("a cone and a cylinder should defer from the cone–cone intersection (ok=false)")
	}
}
