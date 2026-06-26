// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone partial penetration (M2 Phase 2, Oblikovati/Oblikovati#1335). A cone (a tapered rod) that breaches one
// wall of the fatter cylinder and ENDS inside it — one entry imprint loop, the blind end disc interior. The
// plug, blind hole, one-sided stub, and join must weld into watertight analytic solids, the cone's blind end
// disc carrying the cone's radius at that apex distance. Volumes are checked through ops_test; here the
// concern is the watertight topology and that the surfaces stay analytic. A frustum from x=−6 (r=1) to x=0
// (r=1.75) ends at the fat axis, its end disc wholly inside the radius-3 cylinder.

func conePartialFrustum() *topo.Body {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(0, 0, 0), 1, 1.75, "cone")
	return cone
}

func conePartialFat() *topo.Body {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	return fat
}

// TestConePartialPlugThreeFaces: the cone ∩ fat plug is a watertight three-face solid — the fat-wall lens
// cap (cylinder), the cone stub band, and the cone's blind end cap (plane).
func TestConePartialPlugThreeFaces(t *testing.T) {
	res, ok := PartialPenetrationIntersect(conePartialFat(), conePartialFrustum(), nil)
	if !ok {
		t.Fatal("cone partial plug declined; want a three-face plug")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 1 || cyls != 1 || planes != 1 {
		t.Errorf("cone plug got %d cone + %d cyl + %d plane faces, want 1 (band) + 1 (lens cap) + 1 (blind cap)",
			cones, cyls, planes)
	}
}

// TestConePartialBlindHole: fat − cone is a watertight blind pocket — two fat caps, the holed wall, the cone
// tunnel band, and the cone's blind end cap as the pocket bottom (5 faces).
func TestConePartialBlindHole(t *testing.T) {
	res, ok := PartialPenetrationCut(conePartialFat(), conePartialFrustum(), nil)
	if !ok {
		t.Fatal("cone blind hole declined; want a five-face pocketed solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 1 || cyls != 1 || planes != 3 {
		t.Errorf("cone blind hole got %d cone + %d cyl + %d plane faces, want 1 (tunnel) + 1 (holed wall) + 3 (2 fat caps + blind bottom)",
			cones, cyls, planes)
	}
}

// TestConePartialConeMinusFatStub: cone − fat is the single tapered stub sticking out the entry side (one
// shell — a partial penetration sticks out one side only, unlike a full crossing's two stubs).
func TestConePartialConeMinusFatStub(t *testing.T) {
	res, ok := PartialPenetrationCut(conePartialFrustum(), conePartialFat(), nil)
	if !ok {
		t.Fatal("cone − fat (partial) declined; want a single stub")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 1 {
		t.Errorf("cone − fat (partial) has %d shells, want 1 (a single one-sided stub)", n)
	}
}

// TestConePartialJoin: fat ∪ cone is one connected solid — the fat with a single tapered stub sticking out
// the entry side (5 faces, one shell).
func TestConePartialJoin(t *testing.T) {
	res, ok := PartialPenetrationJoin(conePartialFat(), conePartialFrustum(), nil)
	if !ok {
		t.Fatal("fat ∪ cone (partial) declined; want the fat with one tapered stub")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 1 {
		t.Errorf("fat ∪ cone (partial) has %d shells, want 1 (one connected solid)", n)
	}
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 1 || cyls != 1 || planes != 3 {
		t.Errorf("fat ∪ cone got %d cone + %d cyl + %d plane faces, want 1 (stub band) + 1 (holed wall) + 3 (2 fat caps + stub end cap)",
			cones, cyls, planes)
	}
}

// TestConePartialOrderIndependent: the plug resolves whichever body is passed first.
func TestConePartialOrderIndependent(t *testing.T) {
	if _, ok := PartialPenetrationIntersect(conePartialFrustum(), conePartialFat(), nil); !ok {
		t.Error("cone partial plug should resolve with the cone passed first too")
	}
}
