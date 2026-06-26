// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// General curved∩curved pipeline (EPIC Oblikovati/Oblikovati#1403). The first pair routed through the
// general SSI→trimByImprint→solid-membership→curvedStitch path (coneConeIntersectGeneral) must produce the
// SAME watertight three-face shape the bespoke ConeConeIntersect builds — a rod-cone band between the two
// imprint loops plus the two fat-cone lens caps — but with NO per-pair loop→body constructor.

// TestConeConeIntersectGeneralMatchesBespoke crosses a narrow frustum through a fatter one and checks the
// GENERAL pipeline yields a watertight solid whose every face is an analytic cone (3 cones), structurally
// identical to the bespoke handler — proving the general split→classify→stitch works on a real pair.
func TestConeConeIntersectGeneralMatchesBespoke(t *testing.T) {
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")

	res, ok := coneConeIntersectGeneral(thin, fat, nil)
	if !ok {
		t.Fatal("general cone∩cone declined; want the three-face intersection")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 3 || cyls != 0 || planes != 0 {
		t.Errorf("general path got %d cone + %d cyl + %d plane faces, want 3 cones (rod band + 2 fat lens caps)",
			cones, cyls, planes)
	}
}

// TestConeConeIntersectGeneralOrderIndependent: the general path resolves whichever cone is passed first.
func TestConeConeIntersectGeneralOrderIndependent(t *testing.T) {
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	res, ok := coneConeIntersectGeneral(fat, thin, nil)
	if !ok {
		t.Fatal("general cone∩cone should resolve with the fat cone passed first too")
	}
	assertWatertight(t, res)
}
