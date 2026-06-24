// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Cone–cylinder join assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). A cone passing through a fatter
// cylinder must weld into one watertight solid — the fat with a tapered stub each side — the same shape as
// the crossing-cylinder join, only the rod is a cone. Volume is checked through ops_test; here the concern is
// the watertight topology and that the surfaces stay analytic.

// TestConeCylinderJoinFatWithStubs joins a radius-3 cylinder with a crossing frustum and checks the result is
// a watertight seven-face solid: two fat caps, the holed fat wall, and two cone stubs each capped by a disc.
func TestConeCylinderJoinFatWithStubs(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")

	res, ok := ConeCylinderJoin(cyl, cone)
	if !ok {
		t.Fatal("cone ∪ cylinder declined; want the fat with two tapered stubs")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 1 {
		t.Errorf("cone ∪ fat has %d shells, want 1 (one connected solid)", n)
	}
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 2 || cyls != 1 || planes != 4 {
		t.Errorf("cone ∪ fat got %d cone + %d cyl + %d plane faces, want 2 (stub bands) + 1 (holed wall) + 4 (2 fat caps + 2 cone end caps)",
			cones, cyls, planes)
	}
}

// TestConeCylinderJoinOrderIndependent: joining works whichever body is passed first.
func TestConeCylinderJoinOrderIndependent(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	if _, ok := ConeCylinderJoin(cone, cyl); !ok {
		t.Error("cone ∪ cylinder should resolve with the cone passed first too")
	}
}
