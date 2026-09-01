// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// concaveBoreFixture wires a synthetic CONCAVE bore edge: the R=30 cylinder wall (axis +z) meets the
// plane y=0 along the vertical ruling at (30,0,z), with the cylinder face REVERSED so its material-
// outward normal points toward the axis (−r̂) — a BORE (material outside the wall, void inside). Bare
// two-face topology (each face carries the shared edge in a degenerate loop) is enough:
// concaveCylinderArmCandidates reads only the edge endpoints, the cyl face's outward normal (ε), and the
// two geometries. planeN is the plane's material-outward (into-void) normal +ŷ.
func concaveBoreFixture(t *testing.T) (*topo.Edge, geom.Cylinder, geom.Plane, math.UnitVector3) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "concave-bore", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(30, 0, 0), lin)
	hi := bld.AddVertex(math.P3(30, 0, 50), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(30, 0, 0), math.P3(30, 0, 50)), lo, hi, lin)
	cyl := cylAxis(0, 0, 1, 30)
	pl := planeWithNormal(0, 1, 0)
	bld.AddReversedFace(cyl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e))) // reversed ⇒ outward −r̂ (bore)
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cyl, pl, armOutward(0, 1, 0)
}

// TestConcaveCylinderArmCandidates_VoidSide pins the config-(ii) CONCAVE cylinder arm pair on the bore
// fixture: radius r, axis ∥ ẑ, BOTH candidate ball centres at ρ = R−r = 20 from the axis (the
// derivation's data-driven radial offset for ε=−1) and offset +r onto the plane's VOID side (signed
// plane distance +r, not the convex −r). The two rulings are the ±√disc·b mirror pair, so they share
// ρ and the +r plane offset — the void+foot gate, not this builder, later picks the physical one.
func TestConcaveCylinderArmCandidates_VoidSide(t *testing.T) {
	t.Parallel()
	e, cyl, pl, planeN := concaveBoreFixture(t)
	plus, minus, err := concaveCylinderArmCandidates(e, cyl, pl, planeN, 10, testArmResolution())
	if err != nil {
		t.Fatalf("concaveCylinderArmCandidates declined a valid bore edge: %v", err)
	}
	for _, arm := range []geom.Cylinder{plus, minus} {
		if !nearlyArm(arm.Radius, 10) || !nearlyArm(stdmath.Abs(arm.AxisDir.AsVector().Dot(math.V3(0, 0, 1))), 1) {
			t.Fatalf("bore concave arm = {radius %.6f, axis %v}, want {10, ∥ẑ}", arm.Radius, arm.AxisDir)
		}
		if rho := stdmath.Hypot(arm.Origin.X, arm.Origin.Y); !nearlyArm(rho, 20) { // dist(centre, axis)
			t.Fatalf("bore concave arm centre radial distance %.6f, want ρ = R−r = 20", rho)
		}
		if planeOff := arm.Origin.Y; !nearlyArm(planeOff, 10) { // signed dist to plane y=0 = +r (void side)
			t.Fatalf("bore concave arm plane offset %.6f, want +r = 10 (void side, not the convex −10)", planeOff)
		}
	}
}

// TestConcaveCylinderArmCandidates_Boss pins the boss branch (ε=+1): an unreversed cylinder face (material
// inside the wall, +r̂ outward) offsets both ball rulings to ρ = R+r = 40 — the derivation's data-driven
// radial sign, verified distinct from the bore's R−r.
func TestConcaveCylinderArmCandidates_Boss(t *testing.T) {
	t.Parallel()
	e, cyl, pl, planeN := concaveBossFixture(t)
	plus, minus, err := concaveCylinderArmCandidates(e, cyl, pl, planeN, 10, testArmResolution())
	if err != nil {
		t.Fatalf("concaveCylinderArmCandidates declined a valid boss edge: %v", err)
	}
	for _, arm := range []geom.Cylinder{plus, minus} {
		if rho := stdmath.Hypot(arm.Origin.X, arm.Origin.Y); !nearlyArm(rho, 40) {
			t.Fatalf("boss concave arm centre radial distance %.6f, want ρ = R+r = 40", rho)
		}
	}
}

// concaveBossFixture is concaveBoreFixture's boss dual: the cylinder face is NOT reversed, so its
// material-outward normal is +r̂ (material inside the wall) — ε=+1, ρ=R+r.
func concaveBossFixture(t *testing.T) (*topo.Edge, geom.Cylinder, geom.Plane, math.UnitVector3) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "concave-boss", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(30, 0, 0), lin)
	hi := bld.AddVertex(math.P3(30, 0, 50), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(30, 0, 0), math.P3(30, 0, 50)), lo, hi, lin)
	cyl := cylAxis(0, 0, 1, 30)
	pl := planeWithNormal(0, 1, 0)
	bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e))) // unreversed ⇒ outward +r̂ (boss)
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cyl, pl, armOutward(0, 1, 0)
}

// TestConcaveCylinderArmCandidates_Spindle is the bore existence guard: at r ≥ R the offset cylinder
// radius ρ = R−r collapses onto (or through) the axis, so the constructor honest-rejects with a message
// carrying the offending r and R (derivation §4 / the MINOR review finding).
func TestConcaveCylinderArmCandidates_Spindle(t *testing.T) {
	t.Parallel()
	e, cyl, pl, planeN := concaveBoreFixture(t)
	_, _, err := concaveCylinderArmCandidates(e, cyl, pl, planeN, 30, testArmResolution())
	if err == nil {
		t.Fatalf("concaveCylinderArmCandidates accepted r=R=30 (bore spindle ρ=R−r=0) — must reject")
	}
	for _, want := range []string{"r=30", "R=30"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("bore spindle reject %q must carry the offending %s", err.Error(), want)
		}
	}
}

// TestConcaveArmRulingBases_Clears is the P_r∩C_ρ existence guard: when the offset plane clears the offset
// cylinder (|m| > ρ, disc ≤ 0) there is no real ruling and the base solve declines (derivation §4).
func TestConcaveArmRulingBases_Clears(t *testing.T) {
	t.Parallel()
	_, cyl, _, planeN := concaveBoreFixture(t)
	far := planeAtY(200) // 200 ≫ ρ = 20 ⇒ m = 200, disc = ρ²−m² < 0
	if _, _, ok := concaveArmRulingBases(cyl, far, planeN, 20, 10); ok {
		t.Fatalf("concaveArmRulingBases accepted an offset plane that clears the offset cylinder")
	}
}

// planeAtY is a plane at y = d with outward normal +ŷ (the clearance-guard test plane).
func planeAtY(d float64) geom.Plane {
	p, err := geom.NewPlane(math.P3(0, d, 0), math.V3(0, 1, 0))
	if err != nil {
		panic(err)
	}
	return p
}

// concaveArmScene is one real corpus concave Cylinder∧Plane line-edge fixture resolved to the inputs the
// arm builder consumes: the imported body, the picked edge, its two host geometries, the plane host's
// material-outward normal, and the pick radius.
type concaveArmScene struct {
	body   *topo.Body
	edge   *topo.Edge
	cyl    geom.Cylinder
	pl     geom.Plane
	planeN math.UnitVector3
	r      float64
}

// concaveCorpusScene loads a Group-A corpus fixture and locates its concave Cylinder∧Plane line edge by
// the corpus pick midpoint — the same input path concaveCurvedArmFillet sees through the model harness.
func concaveCorpusScene(t *testing.T, name string, mid math.Point3, r float64) concaveArmScene {
	t.Helper()
	body := corpusFixture(t, "simple/"+name+".step")
	e := edgeNearestMidpoint(t, body, mid)
	cyl, pl, ok := cylinderPlaneEdge(e)
	if !ok {
		t.Fatalf("%s: edge nearest %v is not a Cylinder∧Plane edge", name, mid)
	}
	planeN, ok := planeHostNormal(e, pl)
	if !ok {
		t.Fatalf("%s: plane host normal unreadable", name)
	}
	return concaveArmScene{body, e, cyl, pl, planeN, r}
}

// TestConcaveArmRoot_PlaneFootGateDiscriminates is the regression for the IMPORTANT review finding: on
// each real Group-A body BOTH candidate rulings pass the void gate (brep.PointInside==false) AND are
// tangent-r to both infinite hosts, so the void+tangency filter ALONE accepts both — only the plane-foot
// -inside-trimmed-loop test picks the physical root. selectConcaveArmRoot must therefore return exactly
// the void-side arm without consulting nearest-to-midpoint. (Against the pre-fix code, concaveArmRootValid
// on the mirror ruling returned true, so this test's mirror-is-invalid assertion is the guard that fails.)
func TestConcaveArmRoot_PlaneFootGateDiscriminates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mid  math.Point3
		r    float64
	}{
		{"N3", math.P3(112.3726295, 61.41980247, 25), 5},
		{"M4", math.P3(50, 20, 25), 10},
		{"N9", math.P3(80, 10, 55), 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := concaveCorpusScene(t, c.name, c.mid, c.r)
			res := opstol.ForBody(s.body)
			plus, minus, err := concaveCylinderArmCandidates(s.edge, s.cyl, s.pl, s.planeN, s.r, res)
			if err != nil {
				t.Fatalf("%s: candidate arms declined: %v", c.name, err)
			}
			okPlus := concaveArmRootValid(s.body, s.edge, plus, s.cyl, s.pl, s.r, res)
			okMinus := concaveArmRootValid(s.body, s.edge, minus, s.cyl, s.pl, s.r, res)
			if okPlus == okMinus {
				t.Fatalf("%s: void+foot gate did not discriminate the rulings (plus=%v minus=%v)", c.name, okPlus, okMinus)
			}
			// The mirror root passes the void gate + BOTH tangency feet — proving the plane-foot test is
			// the load-bearing discriminator, not the void gate the finding assumed sufficed.
			mirror := plus
			if okPlus {
				mirror = minus
			}
			assertMirrorFailsOnlyPlaneFoot(t, s, mirror, res)
			arm, ok := selectConcaveArmRoot(s.body, s.edge, s.cyl, s.pl, plus, minus, s.r, res)
			if !ok {
				t.Fatalf("%s: selectConcaveArmRoot declined despite a sole valid root", c.name)
			}
			if (okPlus && arm != plus) || (okMinus && arm != minus) {
				t.Fatalf("%s: selectConcaveArmRoot did not return the void-side arm", c.name)
			}
		})
	}
}

// assertMirrorFailsOnlyPlaneFoot verifies the spurious mirror ruling sits OUTSIDE the body (void gate
// passes) and is tangent-r to both infinite hosts (both feet pass) — so it fails concaveArmRootValid
// ONLY on the plane-foot-inside-trimmed-loop test.
func assertMirrorFailsOnlyPlaneFoot(t *testing.T, s concaveArmScene, mirror geom.Cylinder, res opstol.Resolution) {
	t.Helper()
	centre, _ := armBallCenter(mirror, edgeMidpoint(s.edge))
	if brep.PointInside(s.body, centre) {
		t.Fatalf("mirror centre unexpectedly inside the body — void gate already discriminates")
	}
	cylFace, planeFace := concaveHostFaces(s.edge, s.cyl, s.pl)
	tol := res.Weld() * s.r
	if _, ok := armRunoutFoot(cylFace, centre, s.r, tol); !ok {
		t.Fatalf("mirror not tangent to the cylinder host — tangency already discriminates")
	}
	footP, ok := armRunoutFoot(planeFace, centre, s.r, tol)
	if !ok {
		t.Fatalf("mirror not tangent to the plane host — tangency already discriminates")
	}
	if planeFootOnTrimmedFace(planeFace, s.pl, footP) {
		t.Fatalf("mirror plane foot landed INSIDE the trimmed loop — the discriminator failed to separate the roots")
	}
}

// TestSelectConcaveArmRoot_PicksFartherGatePasser proves selection is by the void+foot gate, NOT by
// nearest-to-edge-midpoint: it feeds selectConcaveArmRoot the physical (void-side) arm together with a
// DECOY arm centred ON the wall — the nearest possible ruling to the edge midpoint, which fails tangency.
// The gate-passing arm is FARTHER from the midpoint than the decoy, yet must still be selected. A
// nearest-midpoint heuristic would have wrongly chosen the decoy.
func TestSelectConcaveArmRoot_PicksFartherGatePasser(t *testing.T) {
	t.Parallel()
	s := concaveCorpusScene(t, "N3", math.P3(112.3726295, 61.41980247, 25), 5)
	res := opstol.ForBody(s.body)
	plus, minus, err := concaveCylinderArmCandidates(s.edge, s.cyl, s.pl, s.planeN, s.r, res)
	if err != nil {
		t.Fatalf("candidate arms declined: %v", err)
	}
	physical, ok := selectConcaveArmRoot(s.body, s.edge, s.cyl, s.pl, plus, minus, s.r, res)
	if !ok {
		t.Fatalf("selectConcaveArmRoot declined on a valid N3 edge")
	}
	mid := edgeMidpoint(s.edge)
	decoy, derr := geom.NewCylinderWithRef(mid, s.cyl.AxisDir.AsVector(), s.planeN.AsVector(), s.r) // on the wall
	if derr != nil {
		t.Fatalf("decoy arm build failed: %v", derr)
	}
	if float64(mid.DistanceTo(decoy.Origin)) >= float64(mid.DistanceTo(physical.Origin)) {
		t.Fatalf("decoy is not nearer the edge midpoint than the physical arm — the test is not exercising the farther-root path")
	}
	got, ok := selectConcaveArmRoot(s.body, s.edge, s.cyl, s.pl, decoy, physical, s.r, res)
	if !ok || got != physical {
		t.Fatalf("selectConcaveArmRoot chose the nearer decoy over the farther gate-passing root (ok=%v)", ok)
	}
}

// TestConcaveCurvedArmFillet_TorusArmH7 is the concave-torus-arm-derivation.md wiring regression: on
// H7's real corpus circle edge (R=50 host, cap z=−50, r=10) concaveCurvedArmFillet now dispatches
// armTorus to concaveTorusArmSurface instead of declining "later slice" — the UNIQUE R+r arm (no
// plus/minus root selection, derivation §3), major = R+r = 60, centre = cap projection + r·outwardN =
// (0,0,−40), matching DRAWEXE exactly (derivation §5, H7 row: OCCT arm torus (0,0,−40); ẑ; 60,10).
func TestConcaveCurvedArmFillet_TorusArmH7(t *testing.T) {
	t.Parallel()
	body := corpusFixture(t, "simple/H7.step")
	mid := math.P3(-35.35533906, 35.35533906, -50)
	e := edgeNearestMidpoint(t, body, mid)
	cyl, pl, ok := cylinderPlaneEdge(e)
	if !ok {
		t.Fatalf("H7: edge nearest %v is not a Cylinder∧Plane edge", mid)
	}
	res := opstol.ForBody(body)
	if kind := classifyCurvedArm(cyl, pl, res); kind != armTorus {
		t.Fatalf("H7 arm edge classified %d, want armTorus", kind)
	}
	if conv := ClassifyEdgeConvexity(e); conv != EdgeConcave {
		t.Fatalf("H7 arm edge convexity = %v, want concave", conv)
	}
	p := filletPick{edge: e, r0: 10, r1: 10}
	ef, built, err := concaveCurvedArmFillet(body, e, cyl, pl, p, res, FillConcaveOutward)
	if err != nil || !built {
		t.Fatalf("concaveCurvedArmFillet declined H7's torus arm edge: built=%v err=%v", built, err)
	}
	tor, ok := ef.armSurface.(geom.Torus)
	if !ok {
		t.Fatalf("H7 arm surface = %T, want geom.Torus", ef.armSurface)
	}
	if !ef.armConcave {
		t.Fatalf("H7 arm edgeFillet.armConcave = false, want true (void-side cove)")
	}
	if !nearlyArm(tor.MajorRadius, 60) || !nearlyArm(tor.MinorRadius, 10) {
		t.Fatalf("H7 arm torus radii = (%g, %g), want (60, 10) = R+r, r", tor.MajorRadius, tor.MinorRadius)
	}
	want := math.P3(0, 0, -40)
	if d := float64(tor.Center.DistanceTo(want)); d > 1e-6 {
		t.Fatalf("H7 arm torus centre = %v, want %v (dist %.9g)", tor.Center, want, d)
	}
}

// TestConcaveCurvedArmFillet_LineArmByteIdentical is the do-no-harm regression for pulling the former
// inline armCylinder body out into concaveCylinderArmEdge (to make room for the armTorus branch above
// without pushing concaveCurvedArmFillet over funlen): it proves the dispatcher reproduces EXACTLY the
// value the original inline sequence — planeHostNormal → concaveCylinderArmCandidates →
// selectConcaveArmRoot — produces on N3's concave LINE arm edge (Group A).
func TestConcaveCurvedArmFillet_LineArmByteIdentical(t *testing.T) {
	t.Parallel()
	s := concaveCorpusScene(t, "N3", math.P3(112.3726295, 61.41980247, 25), 5)
	res := opstol.ForBody(s.body)
	if kind := classifyCurvedArm(s.cyl, s.pl, res); kind != armCylinder {
		t.Fatalf("N3 arm edge classified %d, want armCylinder", kind)
	}
	plus, minus, err := concaveCylinderArmCandidates(s.edge, s.cyl, s.pl, s.planeN, s.r, res)
	if err != nil {
		t.Fatalf("N3 candidate arms declined: %v", err)
	}
	want, ok := selectConcaveArmRoot(s.body, s.edge, s.cyl, s.pl, plus, minus, s.r, res)
	if !ok {
		t.Fatalf("N3 selectConcaveArmRoot declined despite a valid pair")
	}
	p := filletPick{edge: s.edge, r0: s.r, r1: s.r}
	ef, built, err := concaveCurvedArmFillet(s.body, s.edge, s.cyl, s.pl, p, res, FillConcaveOutward)
	if err != nil || !built {
		t.Fatalf("concaveCurvedArmFillet declined N3's line arm edge: built=%v err=%v", built, err)
	}
	got, ok := ef.armSurface.(geom.Cylinder)
	if !ok {
		t.Fatalf("N3 arm surface = %T, want geom.Cylinder", ef.armSurface)
	}
	if got != want {
		t.Fatalf("N3 line arm via the dispatcher = %+v, want byte-identical %+v (the original inline candidates+select)", got, want)
	}
	if !ef.armConcave {
		t.Fatalf("N3 arm edgeFillet.armConcave = false, want true")
	}
}
