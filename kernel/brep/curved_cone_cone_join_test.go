// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Cone–cone join assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). A rod cone passing through a fatter cone
// must weld into one watertight solid — the fat cone with a tapered stub each side — the same shape as the
// cone–cylinder join, only the fat is a cone too. Volume is checked through ops_test; here the concern is the
// watertight topology and that the surfaces stay analytic.

// TestConeConeJoinFatWithStubs joins a radius-2→4 fat cone with a crossing radius-0.8→1.5 rod cone and checks
// the result is a watertight seven-face solid: two fat-cone caps, the holed fat-cone wall, and two rod-cone
// stubs each capped by a disc.
func TestConeConeJoinFatWithStubs(t *testing.T) {
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")

	res, ok := ConeConeJoin(fat, thin)
	if !ok {
		t.Fatal("cone ∪ cone declined; want the fat cone with two tapered stubs")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 1 {
		t.Errorf("cone ∪ fat has %d shells, want 1 (one connected solid)", n)
	}
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 3 || cyls != 0 || planes != 4 {
		t.Errorf("cone ∪ fat got %d cone + %d cyl + %d plane faces, want 3 (2 stub bands + holed wall) + 4 (2 fat caps + 2 stub end caps)",
			cones, cyls, planes)
	}
}

// TestConeConeJoinOrderIndependent: joining works whichever cone is passed first.
func TestConeConeJoinOrderIndependent(t *testing.T) {
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	if _, ok := ConeConeJoin(thin, fat); !ok {
		t.Error("cone ∪ cone should resolve with the rod cone passed first too")
	}
}
