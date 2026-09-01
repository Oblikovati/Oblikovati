// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cylindrical boss union (M2 Phase 3, Oblikovati/Oblikovati#1336). A cylinder seated flush on a planar
// face (a boss) unions exactly: the seat face gains the boss-footprint hole, the boss adds one analytic
// cylinder wall and a top cap. The shared base disk is the canonical coplanar overlap the CSG fallback
// faceted.

// TestJoinCylindricalBossOnPlate seats a boss on a plate's top face and checks the result is one watertight
// solid with the boss's single analytic cylinder wall and the plate's faces (the seat face kept, holed).
func TestJoinCylindricalBossOnPlate(t *testing.T) {
	t.Parallel()
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	boss, _ := SolidCylinder(math.P3(0, 0, 2), math.V3(0, 0, 1), 1.5, 3)
	res, ok := JoinCylindricalBoss(plate, boss)
	if !ok {
		t.Fatal("boss union declined; want plate + boss")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 1 {
		t.Errorf("boss union has %d shells, want 1", n)
	}
	_, cyls, planes := faceTypeCounts(t, res)
	if cyls != 1 || planes != 7 {
		t.Errorf("boss union got %d cyl + %d plane faces, want 1 (wall) + 7 (6 box + boss cap)", cyls, planes)
	}
}

// TestJoinCylindricalBossOnBottomFace seats a boss on the plate's BOTTOM face (outward normal −Z), so the
// boss protrudes downward — the seat detection must handle either face orientation.
func TestJoinCylindricalBossOnBottomFace(t *testing.T) {
	t.Parallel()
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	boss, _ := SolidCylinder(math.P3(0, 0, -3), math.V3(0, 0, 1), 1.5, 3) // z −3..0, seated on z=0
	res, ok := JoinCylindricalBoss(plate, boss)
	if !ok {
		t.Fatal("boss on bottom face declined")
	}
	assertWatertight(t, res)
	if _, cyls, _ := faceTypeCounts(t, res); cyls != 1 {
		t.Errorf("downward boss got %d cylinder faces, want 1", cyls)
	}
}

// TestJoinCylindricalBossRejectsOffCase: a boss that would protrude INTO the solid, one clipping the seat
// face edge, and a tool not touching any face each defer (ok=false).
func TestJoinCylindricalBossRejectsOffCase(t *testing.T) {
	t.Parallel()
	plate, _ := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	cases := map[string]*topo.Body{
		"into the solid":  mustCyl(math.P3(0, 0, -1), 1.5, 3),  // z −1..2: far cap inside the plate, not a boss
		"clips seat edge": mustCyl(math.P3(4.5, 0, 2), 1.5, 3), // circle spills past x=5
		"not on any face": mustCyl(math.P3(0, 0, 3), 1.5, 3),   // floats above the plate, base at z=3
	}
	for name, tool := range cases {
		if _, ok := JoinCylindricalBoss(plate, tool); ok {
			t.Errorf("%s: expected the boss union to defer (ok=false)", name)
		}
	}
}
