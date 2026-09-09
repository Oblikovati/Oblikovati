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
	t.Parallel()
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
	t.Parallel()
	sq := box4(0, 0, 10)
	if !dedgeLoopContains(sq, math.P2(5, 5)) {
		t.Error("center point reported outside the square")
	}
	if dedgeLoopContains(sq, math.P2(15, 5)) {
		t.Error("external point reported inside the square")
	}
}

// TestPointInsideConeSolid pins the analytic frustum membership at the band edges and the rim.
func TestPointInsideConeSolid(t *testing.T) {
	t.Parallel()
	cone, _ := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5) // tan~0.546
	if pointInsideConeSolid(cone, 2, 6, math.P3(0, 0, 1.5), false) {
		t.Error("point below vMin reported inside")
	}
	if pointInsideConeSolid(cone, 2, 6, math.P3(0, 0, 7), false) {
		t.Error("point above vMax reported inside")
	}
	if !pointInsideConeSolid(cone, 2, 6, math.P3(0, 0, 4), false) {
		t.Error("on-axis mid-band point reported outside")
	}
	if pointInsideConeSolid(cone, 2, 6, math.P3(100, 0, 4), false) {
		t.Error("far-off-axis point reported inside")
	}
}

// Ruled-crossing corpus (ADR-0058 phase 3, ADR-0061 stage 4). Every ruled∩ruled crossing — cone∩cone,
// cone∩cylinder, cylinder∩cylinder including the near-pinch — is built by the ONE general pipeline
// Boolean routes to: SSI → trimByImprint → solid membership → curvedStitch, with no per-pair loop→body
// constructor and no recognizer in front of it. These rows used to call the bespoke drivers
// (ruledCrossingIntersect, RuledCrossing{Cut,Join}General) directly; they now drive Boolean, so the face
// composition they pin is the composition the KERNEL ships, not one an unreachable driver could still
// produce. Each case must be a watertight solid whose analytic faces are the ones the pair implies.

// TestRuledCrossingIntersectConeCone crosses a narrow frustum through a fatter one: a watertight solid of 3
// analytic cones (the rod band between the two imprint loops + the two fat-cone lens caps).
func TestRuledCrossingIntersectConeCone(t *testing.T) {
	t.Parallel()
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")

	res, err := Boolean(Intersection, thin, fat)
	if err != nil {
		t.Fatalf("cone∩cone: %v", err)
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 3 || cyls != 0 || planes != 0 {
		t.Errorf("cone∩cone got %d cone + %d cyl + %d plane faces, want 3 cones (rod band + 2 fat lens caps)",
			cones, cyls, planes)
	}
	if _, err := Boolean(Intersection, fat, thin); err != nil { // order-independent
		t.Errorf("cone∩cone with the fat cone first: %v", err)
	}
}

// TestRuledCrossingIntersectConeCylinder crosses a cone through a cylinder: 1 cone band inside the cylinder +
// the two cylinder-wall lens caps, watertight, and order-independent.
func TestRuledCrossingIntersectConeCylinder(t *testing.T) {
	t.Parallel()
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	res, err := Boolean(Intersection, cone, cyl)
	if err != nil {
		t.Fatalf("cone∩cylinder: %v", err)
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 1 || cyls != 2 || planes != 0 {
		t.Errorf("cone∩cylinder got %d cone + %d cyl + %d plane faces, want 1 cone band + 2 cylinder lens caps", cones, cyls, planes)
	}
	if _, err := Boolean(Intersection, cyl, cone); err != nil {
		t.Errorf("cone∩cylinder with the cylinder first: %v", err)
	}
}

// TestRuledCrossingIntersectCylinderCylinder crosses two cylinders: a watertight 3-cylinder solid (rod band +
// two fat lens caps).
func TestRuledCrossingIntersectCylinderCylinder(t *testing.T) {
	t.Parallel()
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	res, err := Boolean(Intersection, rod, fat)
	if err != nil {
		t.Fatalf("cylinder∩cylinder: %v", err)
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 0 || cyls != 3 || planes != 0 {
		t.Errorf("cylinder∩cylinder got %d cone + %d cyl + %d plane faces, want 3 cylinders (rod band + 2 lens caps)", cones, cyls, planes)
	}
	if _, err := Boolean(Intersection, fat, rod); err != nil {
		t.Errorf("crossing cylinders with the fat first: %v", err)
	}
}

// TestRuledCrossingIntersectPlanarPairStaysPlanar: two planar blocks carry no ruled side, so the ruled
// machinery must not claim them — the planar half of the same pipeline returns the 6-face box overlap.
// The driver this replaced answered ok=false here so kernel/ops could fall back; there is no fallback
// now, so the assertion is the ANSWER rather than the decline (ADR-0061 stage 4).
func TestRuledCrossingIntersectPlanarPairStaysPlanar(t *testing.T) {
	t.Parallel()
	x, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "x")
	y, _ := SolidBlock(math.P3(1, 1, 1), math.P3(3, 3, 3), "y")
	res, err := Boolean(Intersection, x, y)
	if err != nil {
		t.Fatalf("block∩block: %v", err)
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 0 || cyls != 0 || planes != 6 {
		t.Errorf("block∩block got %d cone + %d cyl + %d plane faces, want the 6-plane overlap box", cones, cyls, planes)
	}
}

// TestCrossingCylinderCut drills a crossing rod through a fat cylinder: the breached fat wall + the rod
// tunnel (2 cylinders) and the fat cylinder's two caps.
func TestCrossingCylinderCut(t *testing.T) {
	t.Parallel()
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	res, err := Boolean(Difference, fat, rod)
	if err != nil {
		t.Fatalf("fat − rod: %v", err)
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 0 || cyls != 2 || planes != 2 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 2 cyl (breached fat + rod tunnel) + 2 plane (fat caps)", cones, cyls, planes)
	}
}

// TestCrossingCylinderJoin: target ∪ tool (a fat cylinder side-breached by a crossing rod) welds into the
// fat's holed wall (a keyhole-bridged tube), the two disjoint rod stubs split by the connected band, and
// both bodies' whole caps.
func TestCrossingCylinderJoin(t *testing.T) {
	t.Parallel()
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	rod, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	res, err := Boolean(Union, fat, rod)
	if err != nil {
		t.Fatalf("fat ∪ rod: %v", err)
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	// 3 cyl: the fat's holed wall (one keyhole-bridged tube with 2 lens holes) + the two rod stubs (the
	// connected-band split separates them, unlike a single 4-loop face). 4 plane: 2 fat caps + 2 rod caps.
	if cones != 0 || cyls != 3 || planes != 4 {
		t.Errorf("got %d cone + %d cyl + %d plane faces, want 3 cyl (holed fat tube + 2 rod stubs) + 4 plane (2 fat + 2 rod caps)", cones, cyls, planes)
	}
}

// TestDisjointCylindersBooleanByComponent: two cylinders far apart share no imprint. The union is both
// bodies whole (2 walls + 4 caps) and the difference is the target untouched — the answers the ruled
// drivers used to decline so a fallback could produce them.
func TestDisjointCylindersBooleanByComponent(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 12)
	b, _ := SolidCylinder(math.P3(20, 0, 0), math.V3(0, 0, 1), 1.5, 12) // far apart, no intersection

	union, err := Boolean(Union, a, b)
	if err != nil {
		t.Fatalf("disjoint ∪: %v", err)
	}
	if cones, cyls, planes := faceTypeCounts(t, union); cones != 0 || cyls != 2 || planes != 4 {
		t.Errorf("disjoint ∪ got %d cone + %d cyl + %d plane faces, want both cylinders whole (2 walls + 4 caps)", cones, cyls, planes)
	}
	diff, err := Boolean(Difference, a, b)
	if err != nil {
		t.Fatalf("disjoint −: %v", err)
	}
	if cones, cyls, planes := faceTypeCounts(t, diff); cones != 0 || cyls != 1 || planes != 2 {
		t.Errorf("disjoint − got %d cone + %d cyl + %d plane faces, want the target untouched (1 wall + 2 caps)", cones, cyls, planes)
	}
}

// TestReverseCurvedFaces: reversing flips the face sense (the cut wall faces into the cavity).
func TestReverseCurvedFaces(t *testing.T) {
	t.Parallel()
	in := []curvedFace{{reversed: false}, {reversed: true}}
	out := reverseCurvedFaces(in)
	if !out[0].reversed || out[1].reversed {
		t.Errorf("reverseCurvedFaces sense = {%v,%v}, want {true,false}", out[0].reversed, out[1].reversed)
	}
}

// TestCurvedBooleanWatertightAcrossScales sweeps one crossing-cylinder cut, one cone-pair cut and one
// equal-radius Steinmetz cut over four decades of model size. Everything the trim reads is
// model-relative (ADR-0042), so the answer must not depend on how big the part is; when it did, a
// tolerance somewhere was absolute. Carried over from the deleted drivers' own scale sweep.
func TestCurvedBooleanWatertightAcrossScales(t *testing.T) {
	t.Parallel()
	for _, s := range []float64{1, 10, 50, 200} {
		fat, _ := SolidCylinder(math.P3(0, 0, math.Scalar(-1.2*s)), math.V3(0, 0, 1), math.Scalar(0.6*s), math.Scalar(2.4*s))
		rod, _ := SolidCylinder(math.P3(math.Scalar(-1.2*s), 0, 0), math.V3(1, 0, 0), math.Scalar(0.3*s), math.Scalar(2.4*s))
		res, err := Boolean(Difference, fat, rod)
		if err != nil {
			t.Errorf("scale %g: crossing-cylinder cut: %v", s, err)
		} else {
			assertWatertight(t, res)
		}

		fatC, _ := SolidCylinderCone(math.P3(0, 0, math.Scalar(-1.2*s)), math.P3(0, 0, math.Scalar(1.2*s)), 0.4*s, 0.8*s, "fat")
		rodC, _ := SolidCylinderCone(math.P3(math.Scalar(-1.2*s), 0, 0), math.P3(math.Scalar(1.2*s), 0, 0), 0.16*s, 0.3*s, "rod")
		resC, err := Boolean(Difference, fatC, rodC)
		if err != nil {
			t.Errorf("scale %g: cone-cone cut: %v", s, err)
		} else {
			assertWatertight(t, resC)
		}

		a, _ := SolidCylinder(math.P3(math.Scalar(-1.2*s), 0, 0), math.V3(1, 0, 0), math.Scalar(0.6*s), math.Scalar(2.4*s))
		b, _ := SolidCylinder(math.P3(0, 0, math.Scalar(-1.2*s)), math.V3(0, 0, 1), math.Scalar(0.6*s), math.Scalar(2.4*s))
		resS, err := Boolean(Difference, a, b)
		if err != nil {
			t.Errorf("scale %g: Steinmetz cut: %v", s, err)
		} else {
			assertWatertight(t, resS)
		}
	}
}
