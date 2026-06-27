// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// box4 returns a closed CCW (area>0) square loop of dedges from (x0,y0) of the given side, as the (u,v)
// boundary the grouping helpers consume.
func box4(x0, y0, side float64) []dedge {
	p := [4]math.Point2{
		math.P2(x0, y0), math.P2(x0+side, y0), math.P2(x0+side, y0+side), math.P2(x0, y0+side),
	}
	out := make([]dedge, 4)
	for i := range p {
		out[i] = dedge{a: p[i], b: p[(i+1)%4]}
	}
	return out
}

// reverseLoop flips a dedge loop's winding (a CCW outer becomes a CW hole, area<0).
func reverseDedgeLoop(loop []dedge) []dedge {
	out := make([]dedge, len(loop))
	for i, e := range loop {
		out[len(loop)-1-i] = dedge{a: e.b, b: e.a}
	}
	return out
}

// TestGroupLoopFacesContainment pins the disjoint-faces grouping (#1403): two separated outer loops with a
// hole nested in one become TWO faces, the hole attached to the outer that contains it — and the gate keeps
// the half-space/wrapping cases as a single face. This exercises groupLoopFaces, smallestContainingFace,
// dedgeLoopContains and loopPointInside, which the connected cone∩cone case (two outers, no hole) does not.
func TestGroupLoopFacesContainment(t *testing.T) {
	outerA := box4(0, 0, 10)                // contains the hole
	hole := reverseDedgeLoop(box4(3, 3, 4)) // CW hole inside outerA
	outerB := box4(20, 0, 10)               // disjoint second face
	loops := [][]dedge{outerA, hole, outerB}

	// Not multiFace, or wrapping → one face (the unchanged half-space convention).
	if g := groupLoopFaces(false, false, loops); len(g) != 1 {
		t.Errorf("non-multiface grouped into %d faces, want 1 (half-space convention)", len(g))
	}
	if g := groupLoopFaces(true, true, loops); len(g) != 1 {
		t.Errorf("wrapping band grouped into %d faces, want 1", len(g))
	}
	// multiFace, non-wrapping → two faces: {outerA, hole} and {outerB}.
	groups := groupLoopFaces(true, false, loops)
	if len(groups) != 2 {
		t.Fatalf("grouped into %d faces, want 2 (the two disjoint outers)", len(groups))
	}
	withHole := 0
	for _, g := range groups {
		if len(g) == 2 {
			withHole++ // the face that received the nested hole
		}
	}
	if withHole != 1 {
		t.Errorf("%d faces carry the hole, want exactly 1 (the containing outer)", withHole)
	}
}

// TestDedgeLoopContains pins the (u,v) point-in-polygon used to nest holes.
func TestDedgeLoopContains(t *testing.T) {
	sq := box4(0, 0, 10)
	if !dedgeLoopContains(sq, math.P2(5, 5)) {
		t.Error("center point reported outside the square")
	}
	if dedgeLoopContains(sq, math.P2(15, 5)) {
		t.Error("external point reported inside the square")
	}
}

// TestCurvedSolidMembershipDeclinesNonCone: the membership oracle handles a cone solid and declines a shape
// it has no analytic test for, so the general path defers rather than misclassify.
func TestCurvedSolidMembershipDeclinesNonCone(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "cone")
	if _, ok := curvedSolidMembership(cone); !ok {
		t.Error("cone solid membership should be available")
	}
	sph, _ := SolidSphere(math.P3(0, 0, 0), 3, "sph")
	if _, ok := curvedSolidMembership(sph); ok {
		t.Error("sphere solid membership is not wired yet; want ok=false so the caller defers")
	}
}

// TestPointInsideConeSolid pins the analytic frustum membership at the band edges and the rim.
func TestPointInsideConeSolid(t *testing.T) {
	cone, _ := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5) // tan~0.546
	if pointInsideConeSolid(cone, 2, 6, math.P3(0, 0, 1.5)) {
		t.Error("point below vMin reported inside")
	}
	if pointInsideConeSolid(cone, 2, 6, math.P3(0, 0, 7)) {
		t.Error("point above vMax reported inside")
	}
	if !pointInsideConeSolid(cone, 2, 6, math.P3(0, 0, 4)) {
		t.Error("on-axis mid-band point reported outside")
	}
	if pointInsideConeSolid(cone, 2, 6, math.P3(100, 0, 4)) {
		t.Error("far-off-axis point reported inside")
	}
}

// TestConeConeIntersectGeneralDeclines: the exported entry and the driver decline a cone+cylinder (the
// cone-cylinder case) and other non-frustum-crossing inputs, so kernel/ops keeps its fallback.
func TestConeConeIntersectGeneralDeclines(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeConeIntersectGeneral(cone, cyl, nil); ok {
		t.Error("cone∩cylinder should decline from the cone∩cone general path (ok=false)")
	}
}

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

// TestConeCylinderIntersectGeneral: cone∩cylinder through the general pipeline yields a watertight solid —
// the cone band inside the cylinder plus the two cylinder-wall lens caps — proving the two-sided recipe
// reuses the cylinder side (one cone + one cylinder), the second EPIC #1403 migration.
func TestConeCylinderIntersectGeneral(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	res, ok := coneCylinderIntersectGeneral(cone, cyl, nil)
	if !ok {
		t.Fatal("general cone∩cylinder declined; want the three-face intersection")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 1 || cyls != 2 || planes != 0 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 1 cone band + 2 cylinder lens caps", cones, cyls, planes)
	}
	// Order-independent + exported entry.
	if _, ok := ConeCylinderIntersectGeneral(cyl, cone, nil); !ok {
		t.Error("cone∩cylinder should resolve with the cylinder passed first too")
	}
}

// TestConeCylinderIntersectGeneralDeclines: two cylinders are not the cone∩cylinder case.
func TestConeCylinderIntersectGeneralDeclines(t *testing.T) {
	c1, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1, 12)
	c2, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeCylinderIntersectGeneral(c1, c2, nil); ok {
		t.Error("two cylinders should decline from the cone∩cylinder general path")
	}
}

// TestCrossingCylinderIntersectGeneral: two crossing cylinders through the general pipeline yield a
// watertight three-cylinder solid (rod band + two fat lens caps) — the simplest, fully symmetric pair
// (both sides cylinders), the third EPIC #1403 migration and the largest bespoke handler replaced.
func TestCrossingCylinderIntersectGeneral(t *testing.T) {
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	res, ok := crossingCylinderIntersectGeneral(rod, fat, nil)
	if !ok {
		t.Fatal("general crossing-cylinder declined; want the three-face intersection")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 0 || cyls != 3 || planes != 0 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 3 cylinders (rod band + 2 lens caps)", cones, cyls, planes)
	}
	if _, ok := CrossingCylinderIntersectGeneral(fat, rod, nil); !ok {
		t.Error("crossing cylinders should resolve with the fat passed first too")
	}
}

// TestCrossingCylinderIntersectGeneralDeclines: a cone and a cylinder are not the crossing-cylinder case.
func TestCrossingCylinderIntersectGeneralDeclines(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := CrossingCylinderIntersectGeneral(cone, cyl, nil); ok {
		t.Error("cone+cylinder should decline from the crossing-cylinder general path")
	}
}

// TestCrossingCylinderCutGeneral checks the STRUCTURE the general cut builder emits — its face composition and
// edge-USE-COUNT (every edge used twice). It does NOT assert orientation/region correctness: this builder is
// known broken (Oblikovati#1476, the OUTSIDE-keep region bug) and is NOT wired into kernel/ops — crossing-
// cylinder subtract stays on the bespoke handler. Orientation/volume are validated in ops (TestGeneral...
// IsAdopted covers the intersect path; cut/join join that guard once #1476 lands). Edge-count watertightness
// alone is exactly what masked the silent fallback, so this is intentionally scoped to structure (#1403).
func TestCrossingCylinderCutGeneral(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	res, ok := CrossingCylinderCutGeneral(fat, rod, nil)
	if !ok {
		t.Fatal("general crossing-cylinder cut declined; want the drilled solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 0 || cyls != 2 || planes != 2 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 2 cyl (breached fat + rod tunnel) + 2 plane (fat caps)", cones, cyls, planes)
	}
}

// TestCrossingCylinderJoinGeneral: target ∪ tool (fat side-breached by a crossing rod) through the general
// pipeline yields the correct welded solid — the fat's holed wall (a keyhole-bridged tube), the two disjoint
// rod stubs (split by connected band), and BOTH bodies' whole caps. The OUTSIDE-keep wrapping-band emission
// (Oblikovati#1476) is what makes this mesh the right region; its volume is checked against OCC in ops
// (TestCurvedBooleanVolumesMatchOCC, crossing ∪). Here we assert the watertight face structure (#1403/#1476).
func TestCrossingCylinderJoinGeneral(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	res, ok := CrossingCylinderJoinGeneral(fat, rod, nil)
	if !ok {
		t.Fatal("general crossing-cylinder join declined; want the welded solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	// 3 cyl: the fat's holed wall (one keyhole-bridged tube with 2 lens holes) + the two rod stubs (the
	// connected-band split separates them, unlike a single 4-loop face). 4 plane: 2 fat caps + 2 rod caps.
	if cones != 0 || cyls != 3 || planes != 4 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 3 cyl (holed fat tube + 2 rod stubs) + 4 plane (2 fat + 2 rod caps)", cones, cyls, planes)
	}
}

// TestCrossingCylinderJoinGeneralDeclines: a non-crossing pair (far apart, no imprint) declines so kernel/ops
// keeps its fallback.
func TestCrossingCylinderJoinGeneralDeclines(t *testing.T) {
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 12)
	b, _ := SolidCylinder(math.P3(20, 0, 0), math.V3(0, 0, 1), 1.5, 12) // far apart, no intersection
	if _, ok := CrossingCylinderJoinGeneral(a, b, nil); ok {
		t.Error("non-intersecting cylinders should decline from the join general path")
	}
}

// TestJoinFacesAssembly: the union assembly keeps both walls outward (no reversal) and contributes both
// bodies' caps — distinct from the cut, which reverses the tool wall and drops the tool's caps.
func TestJoinFacesAssembly(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	wallA := []curvedFace{{reversed: false}}
	wallB := []curvedFace{{reversed: false}}
	faces := joinFaces(wallA, fat, wallB, rod)
	for i, f := range faces {
		if f.reversed {
			t.Errorf("joinFaces[%d] reversed=true, want all walls/caps outward (union keeps outward sense)", i)
		}
	}
	// 1 wallA + 2 fat caps + 1 wallB + 2 rod caps = 6 faces.
	if len(faces) != 6 {
		t.Errorf("joinFaces produced %d faces, want 6 (1 wallA + 2 fat caps + 1 wallB + 2 rod caps)", len(faces))
	}
}

// TestReverseCurvedFaces: reversing flips the face sense (the cut wall faces into the cavity).
func TestReverseCurvedFaces(t *testing.T) {
	in := []curvedFace{{reversed: false}, {reversed: true}}
	out := reverseCurvedFaces(in)
	if !out[0].reversed || out[1].reversed {
		t.Errorf("reverseCurvedFaces sense = {%v,%v}, want {true,false}", out[0].reversed, out[1].reversed)
	}
}

// TestCrossingCylinderCutGeneralDeclines: a non-crossing pair (parallel, no imprint) declines so kernel/ops
// keeps its fallback.
func TestCrossingCylinderCutGeneralDeclines(t *testing.T) {
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 12)
	b, _ := SolidCylinder(math.P3(20, 0, 0), math.V3(0, 0, 1), 1.5, 12) // far apart, no intersection
	if _, ok := CrossingCylinderCutGeneral(a, b, nil); ok {
		t.Error("non-intersecting cylinders should decline from the cut general path")
	}
}
