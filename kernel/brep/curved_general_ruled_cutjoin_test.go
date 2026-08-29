// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// coneCutJoinPair builds the test cone pair: a radius-0.8→1.5 rod frustum (axis x) crossing a radius-2→4 fat
// frustum (axis z).
func coneCutJoinPair() (fat, rod *topo.Body) {
	fat, _ = SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	rod, _ = SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "rod")
	return fat, rod
}

// TestRuledConeCrossingJoinGeneral: fat ∪ rod (two crossing frustums) through the general pipeline is a watertight
// solid — the fat's keyhole-bridged holed wall, the two tapered rod stubs, and all four caps (#1403).
func TestRuledConeCrossingJoinGeneral(t *testing.T) {
	fat, rod := coneCutJoinPair()
	res, ok := RuledConeCrossingJoinGeneral(fat, rod, nil)
	if !ok {
		t.Fatal("general cone∪cone declined; want the welded solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	// 3 cone: the fat's holed wall + the two rod stubs. 4 plane: 2 fat caps + 2 rod caps.
	if cones != 3 || cyls != 0 || planes != 4 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 3 cone (holed fat + 2 stubs) + 4 plane caps", cones, cyls, planes)
	}
}

// TestRuledConeCrossingCutGeneral: fat − rod (drilling the fat frustum) through the general pipeline is a watertight
// solid — the breached fat wall, its whole caps, and the reversed rod tunnel (#1403).
func TestRuledConeCrossingCutGeneral(t *testing.T) {
	fat, rod := coneCutJoinPair()
	res, ok := RuledConeCrossingCutGeneral(fat, rod, nil)
	if !ok {
		t.Fatal("general cone−cone declined; want the drilled solid")
	}
	assertWatertight(t, res)
	cones, _, planes := faceTypeCounts(t, res)
	// 2 cone: the breached fat wall + the rod tunnel. 2 plane: the fat's two whole caps.
	if cones != 2 || planes != 2 {
		t.Errorf("got %d cone + %d plane faces, want 2 cone (breached fat + tunnel) + 2 plane (fat caps)", cones, planes)
	}
}

// TestConeCylinderCutJoinGeneralResolveOrder: the mixed cone/cylinder pair resolves each operand by type
// (ruledOperandOf) regardless of which argument is the cone, so both argument orders are accepted (#1403).
func TestConeCylinderCutJoinGeneralResolveOrder(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	if _, ok := RuledConeCrossingJoinGeneral(cyl, cone, nil); !ok {
		t.Error("cone∪cylinder declined with cylinder first")
	}
	if _, ok := RuledConeCrossingJoinGeneral(cone, cyl, nil); !ok {
		t.Error("cone∪cylinder declined with cone first")
	}
}

// TestConePairDecline: a non-crossing pair (far apart, no imprint) declines so kernel/ops keeps its fallback.
func TestConePairDecline(t *testing.T) {
	a, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "a")
	b, _ := SolidCylinderCone(math.P3(40, 0, 0), math.P3(52, 0, 0), 0.8, 1.5, "b") // far apart, no intersection
	if _, ok := RuledConeCrossingJoinGeneral(a, b, nil); ok {
		t.Error("non-intersecting frustums should decline from the join general path")
	}
	if _, ok := RuledConeCrossingCutGeneral(a, b, nil); ok {
		t.Error("non-intersecting frustums should decline from the cut general path")
	}
}

// TestRuledOperandOfResolvesType: ruledOperandOf builds a cone operand from a frustum body and a cylinder
// operand from a cylinder body, carrying that surface forward for the split (#1403).
func TestRuledOperandOfResolvesType(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	oc, okc := ruledOperandOf(cone)
	oy, oky := ruledOperandOf(cyl)
	if !okc || !oky {
		t.Fatalf("ruledOperandOf failed: cone=%v cylinder=%v", okc, oky)
	}
	if _, isCone := oc.surface.(geom.Cone); !isCone {
		t.Errorf("cone operand carries %T, want geom.Cone", oc.surface)
	}
	if _, isCyl := oy.surface.(geom.Cylinder); !isCyl {
		t.Errorf("cylinder operand carries %T, want geom.Cylinder", oy.surface)
	}
}

// partialPair builds the partial-penetration test pair: a radius-1.5 rod on x ending at the centre (x=0) of a
// radius-3 fat cylinder on z (the rod's blind end sits inside the fat).
func partialPair() (fat, stub *topo.Body) {
	fat, _ = SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	stub, _ = SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6)
	return fat, stub
}

// TestPartialImprintOneLoop: the partial imprint is exactly ONE loop (the single entry breach), traced
// rod-first whichever argument order is given (#1403).
func TestPartialImprintOneLoop(t *testing.T) {
	fat, stub := partialPair()
	for _, order := range []struct {
		name string
		a, b *topo.Body
	}{{"fat,stub", fat, stub}, {"stub,fat", stub, fat}} {
		loops, ok := partialImprint(order.a, order.b, nil)
		if !ok || len(loops) != 1 {
			t.Errorf("partialImprint(%s) ok=%v loops=%d, want ok=true loops=1", order.name, ok, len(loops))
		}
	}
}

// TestPartialJoinGeneralStructure: fat ∪ rod (a partial penetration) is a watertight solid — the holed fat
// wall, its two whole caps, the single rod stub, and the rod's entry cap; the rod's blind cap (inside the fat)
// is dropped (#1403).
func TestPartialJoinGeneralStructure(t *testing.T) {
	fat, stub := partialPair()
	res, ok := PartialPenetrationJoinGeneral(fat, stub, nil)
	if !ok {
		t.Fatal("partial join declined; want the stub solid")
	}
	assertWatertight(t, res)
	_, cyls, planes := faceTypeCounts(t, res)
	// 2 cyl: holed fat wall + the single rod stub. 3 plane: 2 fat caps + the rod's entry cap (blind cap dropped).
	if cyls != 2 || planes != 3 {
		t.Errorf("got %d cyl + %d plane faces, want 2 cyl (holed fat + stub) + 3 plane (2 fat caps + entry cap)", cyls, planes)
	}
}

// TestPartialCutGeneralBlindHole: fat − rod is a watertight blind-hole solid — the holed fat wall, its two
// caps, the reversed rod tunnel, and the rod's blind cap as the pocket bottom (#1403).
func TestPartialCutGeneralBlindHole(t *testing.T) {
	fat, stub := partialPair()
	res, ok := PartialPenetrationCutGeneral(fat, stub, nil)
	if !ok {
		t.Fatal("partial cut declined; want the blind hole")
	}
	assertWatertight(t, res)
	_, cyls, planes := faceTypeCounts(t, res)
	// 2 cyl: holed fat wall + the rod tunnel. 3 plane: 2 fat caps + the rod's blind cap (pocket bottom).
	if cyls != 2 || planes != 3 {
		t.Errorf("got %d cyl + %d plane faces, want 2 cyl (holed fat + tunnel) + 3 plane (2 fat caps + blind cap)", cyls, planes)
	}
}

// TestCapsInsideOutsidePartition: for the partial rod, its blind cap (centre inside the fat) is capsInside and
// its entry cap (centre outside) is capsOutside; the partition is complete (#1403).
func TestCapsInsideOutsidePartition(t *testing.T) {
	fat, stub := partialPair()
	fatInside, _ := curvedSolidMembership(fat)
	in := capsInside(stub, fatInside)
	out := capsOutside(stub, fatInside)
	if len(in) != 1 || len(out) != 1 {
		t.Errorf("rod caps partition = %d inside + %d outside, want 1 + 1 (blind inside, entry outside)", len(in), len(out))
	}
}

// TestPartialIntersectGeneralPlug: rod ∩ fat (a partial penetration) is a watertight plug — the fat-wall lens
// cap, the rod-wall band, and the rod's blind end cap (the interior-ending planar disc) (#1403).
func TestPartialIntersectGeneralPlug(t *testing.T) {
	fat, stub := partialPair()
	res, ok := PartialPenetrationIntersectGeneral(fat, stub, nil)
	if !ok {
		t.Fatal("partial intersect declined; want the plug")
	}
	assertWatertight(t, res)
	_, cyls, planes := faceTypeCounts(t, res)
	// 2 cyl: the fat-wall lens cap (a trimmed cylinder patch) + the rod-wall band. 1 plane: the rod's blind cap.
	if cyls != 2 || planes != 1 {
		t.Errorf("got %d cyl + %d plane faces, want 2 cyl (lens cap + rod band) + 1 plane (blind cap)", cyls, planes)
	}
}

// TestPartialPairDecline: a FULL crossing (the rod passes right through) is not a partial penetration — the
// imprint is two loops either way — so the partial drivers decline and kernel/ops uses the crossing path.
func TestPartialPairDecline(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	through, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12) // passes fully through
	if _, ok := PartialPenetrationJoinGeneral(fat, through, nil); ok {
		t.Error("a full crossing should decline from the partial join path")
	}
}

// TestCurvedBooleanWatertightAcrossScales is the #1602 scale sweep: the same curved booleans run at
// 1 cm–200 cm extents must all stitch watertight. The seam points fed to the stitch carry SSI noise
// proportional to the extent, so a weld grid that does not scale with the model (the retired
// absolute 1e-6) lets seams tear as parts grow.
func TestCurvedBooleanWatertightAcrossScales(t *testing.T) {
	for _, s := range []float64{1, 10, 50, 200} {
		fat, _ := SolidCylinder(math.P3(0, 0, -1.2*s), math.V3(0, 0, 1), 0.6*s, 2.4*s)
		rod, _ := SolidCylinder(math.P3(-1.2*s, 0, 0), math.V3(1, 0, 0), 0.3*s, 2.4*s)
		if res, ok := CrossingCylinderCutGeneral(fat, rod, nil); ok {
			assertWatertight(t, res)
		} else {
			t.Errorf("scale %g: crossing-cylinder cut declined", s)
		}

		fatC, _ := SolidCylinderCone(math.P3(0, 0, -1.2*s), math.P3(0, 0, 1.2*s), 0.4*s, 0.8*s, "fat")
		rodC, _ := SolidCylinderCone(math.P3(-1.2*s, 0, 0), math.P3(1.2*s, 0, 0), 0.16*s, 0.3*s, "rod")
		if res, ok := RuledConeCrossingCutGeneral(fatC, rodC, nil); ok {
			assertWatertight(t, res)
		} else {
			t.Errorf("scale %g: cone-cone cut declined", s)
		}

		a, _ := SolidCylinder(math.P3(-1.2*s, 0, 0), math.V3(1, 0, 0), 0.6*s, 2.4*s)
		b, _ := SolidCylinder(math.P3(0, 0, -1.2*s), math.V3(0, 0, 1), 0.6*s, 2.4*s)
		if res, ok := SteinmetzCutGeneral(a, b, nil); ok {
			assertWatertight(t, res)
		} else {
			t.Errorf("scale %g: Steinmetz cut declined", s)
		}
	}
}
