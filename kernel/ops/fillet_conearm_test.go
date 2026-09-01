// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The C2/C6/D1 cone-host fixtures are all pcone frustums/sharp cones, fillet r=10, apex on the +z axis,
// axis â=−ẑ (opening downward). These pin coneArmSurface's closed form and coneContactCircle to the
// DRAWEXE oracle tori (cone-host-corner-derivation.md §2 "Arm A"): each is an exact torus OCCT emits as
// a ToroidalSurface KPart, matched to ALL printed digits — so a major/centre/contact off by more than
// coneArmExactTol is a transcription bug, not imprecision.
const (
	coneArmR        = 10.0
	coneArmExactTol = 1e-9
)

// coneArmCase is a cone-host Arm A case: the host cone, its cap plane (⊥ axis) with material-outward
// normal, and the DRAWEXE-verified torus arm + host contact circle it must reproduce.
type coneArmCase struct {
	name           string
	apex           math.Point3
	tanAlpha       float64
	capZ           float64      // the cap-plane z (the circle edge's plane)
	capOutward     math.Vector3 // the cap plane's material-outward normal
	wantMajor      float64      // spine (major) radius R_s
	wantCenterZ    float64      // torus centre z (on the axis)
	wantContactZ   float64      // host contact circle z (on the axis)
	wantContactRad float64      // host contact circle radius
}

// coneAxisDown is the shared frustum/cone axis: −ẑ (all four fixtures open downward).
func coneAxisDown() math.Vector3 { return math.V3(0, 0, -1) }

// coneFor builds the host cone of a case: apex on the axis, axis −ẑ, half-angle atan(tanα).
func coneFor(t *testing.T, c coneArmCase) geom.Cone {
	t.Helper()
	co, err := geom.NewCone(c.apex, coneAxisDown(), stdmath.Atan(c.tanAlpha))
	if err != nil {
		t.Fatalf("%s cone: %v", c.name, err)
	}
	return co
}

// TestConeArmSurface_Oracle pins the torus arm of every cone-host cap (circle) edge to its DRAWEXE
// oracle spine circle (cone-host-corner-derivation.md §2): major radius R_s = tanα·h′ and centre O′ are
// exact closed forms of {r, cone, cap plane}, and minor = r always. A wrong material-side sign (A′ on the
// apex side) or a wrong cap-offset direction moves the major by tens of units — caught here to ≤1e-9.
func TestConeArmSurface_Oracle(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	for _, c := range coneArmOracleCases() {
		t.Run(c.name, func(t *testing.T) {
			co := coneFor(t, c)
			pl := planeOn(t, math.P3(0, 0, c.capZ), c.capOutward)
			n, _ := math.UnitVector3FromVector(c.capOutward)
			tor, reason := coneArmSurface(co, pl, n, +1, coneArmR, res)
			if reason != coneArmBuilt {
				t.Fatalf("%s: coneArmSurface declined a valid convex arm (reason %d)", c.name, reason)
			}
			assertConeArm(t, c, tor)
		})
	}
}

// assertConeArm checks one built torus arm's major, minor, and centre against the oracle to ≤1e-9.
func assertConeArm(t *testing.T, c coneArmCase, tor geom.Torus) {
	t.Helper()
	if stdmath.Abs(tor.MinorRadius-coneArmR) > coneArmExactTol {
		t.Fatalf("%s torus minor = %.12f, want %g", c.name, tor.MinorRadius, coneArmR)
	}
	if stdmath.Abs(tor.MajorRadius-c.wantMajor) > coneArmExactTol {
		t.Fatalf("%s torus major = %.12f, want %.12f (Δ %.3g)", c.name, tor.MajorRadius, c.wantMajor, tor.MajorRadius-c.wantMajor)
	}
	if d := float64(tor.Center.DistanceTo(math.P3(0, 0, c.wantCenterZ))); d > coneArmExactTol {
		t.Fatalf("%s torus centre = %v, want (0,0,%g) (off by %.3g)", c.name, tor.Center, c.wantCenterZ, d)
	}
}

// TestConeContactCircle_Oracle pins the torus↔cone contact circle (the retrim rail on the cone host) of
// every cone-host Arm A case to its DRAWEXE-verified z and radius (cone-host-corner-derivation.md §2:
// s* = h·cosα + R_s·sinα; radius s*·sinα centred at A + s*·cosα·â) to ≤1e-9, built from the real arm.
func TestConeContactCircle_Oracle(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	for _, c := range coneArmOracleCases() {
		t.Run(c.name, func(t *testing.T) {
			co := coneFor(t, c)
			pl := planeOn(t, math.P3(0, 0, c.capZ), c.capOutward)
			n, _ := math.UnitVector3FromVector(c.capOutward)
			tor, reason := coneArmSurface(co, pl, n, +1, coneArmR, res)
			if reason != coneArmBuilt {
				t.Fatalf("%s: arm not built (reason %d)", c.name, reason)
			}
			center, radius, ok := coneContactCircle(co, tor, res)
			if !ok {
				t.Fatalf("%s: coneContactCircle rejected the arm's own host cone", c.name)
			}
			assertContact(t, c, center, radius)
		})
	}
}

// assertContact checks the contact circle's on-axis z and radius against the oracle to ≤1e-9.
func assertContact(t *testing.T, c coneArmCase, center math.Point3, radius float64) {
	t.Helper()
	if stdmath.Abs(float64(center.Z)-c.wantContactZ) > coneArmExactTol {
		t.Fatalf("%s contact z = %.12f, want %.12f", c.name, float64(center.Z), c.wantContactZ)
	}
	if stdmath.Abs(float64(center.X)) > coneArmExactTol || stdmath.Abs(float64(center.Y)) > coneArmExactTol {
		t.Fatalf("%s contact centre off the axis: %v", c.name, center)
	}
	if stdmath.Abs(radius-c.wantContactRad) > coneArmExactTol {
		t.Fatalf("%s contact radius = %.12f, want %.12f", c.name, radius, c.wantContactRad)
	}
}

// coneArmOracleCases are the three cone-host cap (circle) edges of the corpus (C8 picks no circle edge):
// C2 bottom rim (frustum, tip away), C6 TOP rim (frustum, tip-toward-corner — the offset moves apex-ward,
// h reads on the apex side), D1 bottom rim (sharp cone). Numbers from cone-host-corner-derivation.md §2.
func coneArmOracleCases() []coneArmCase {
	return []coneArmCase{
		{"C2", math.P3(0, 0, 270), 1.0 / 3.0, 0, math.V3(0, 0, -1),
			76.1257411328, 10, 13.1622776602, 85.6125741133},
		{"C6", math.P3(0, 0, 270), 1.0 / 3.0, 150, math.V3(0, 0, 1),
			32.7924077994, 140, 143.1622776602, 42.2792407799},
		{"D1", math.P3(0, 0, 120), 5.0 / 12.0, 0, math.V3(0, 0, -1),
			35, 10, 13.8461538462, 44.2307692308},
	}
}

// TestConePlaneEdge recognizes an edge flanked by a cone face and a plane face, returning both surfaces
// and both faces; a plane∧plane edge (the third, straight corner edge) is not recognized.
func TestConePlaneEdge(t *testing.T) {
	t.Parallel()
	e, _, _ := conePlaneFixtureEdge(t, false)
	co, pl, cf, pf, ok := conePlaneEdge(e)
	if !ok {
		t.Fatalf("conePlaneEdge did not recognize a cone∧plane edge")
	}
	if cf == nil || pf == nil || pl.Origin.Z != 0 {
		t.Fatalf("conePlaneEdge returned {planeOrigin=%v, coneFace=%v, planeFace=%v}", pl.Origin, cf, pf)
	}
	if _, isCone := cf.Geometry().(geom.Cone); !isCone {
		t.Fatalf("conePlaneEdge returned a non-cone as the cone face: %T", cf.Geometry())
	}
	_ = co
	if _, _, _, _, ok := conePlaneEdge(planePlaneFixtureEdge(t)); ok {
		t.Fatalf("conePlaneEdge wrongly recognized a plane∧plane edge")
	}
}

// TestConeArmEdge_Dispatches is the integration guard: computeEdgeFillet's cone branch HANDLES a
// cone∧plane cap edge (handled=true, arm built) so it returns before curvedAdjacentError; a plane∧plane
// edge is NOT handled here (handled=false), keeping the existing planar path byte-identical.
func TestConeArmEdge_Dispatches(t *testing.T) {
	t.Parallel()
	e, _, _ := conePlaneFixtureEdge(t, false)
	ef, handled, err := coneArmEdge(nil, e, filletPick{edge: e, r0: 10, r1: 10})
	if !handled || err != nil || ef.armSurface == nil {
		t.Fatalf("cone∧plane cap edge: want handled+no-error+arm, got handled=%v err=%v arm=%v", handled, err, ef.armSurface)
	}
	if _, ok := ef.armSurface.(geom.Torus); !ok {
		t.Fatalf("cone∧plane cap arm is %T, want geom.Torus", ef.armSurface)
	}
	pp := planePlaneFixtureEdge(t)
	if _, handled, _ := coneArmEdge(nil, pp, filletPick{edge: pp, r0: 10, r1: 10}); handled {
		t.Fatalf("coneArmEdge wrongly handled a plane∧plane edge (want handled=false, keep the planar path)")
	}
}

// TestConeArm_ConcaveBoreBuilds is the material-side dispatch gate: a conical BORE rim — the cone face is
// Reversed (material OUTSIDE the cone) — is a genuinely CONVEX edge (do-no-harm comment above
// coneArmFillet), so coneHostMaterialSign (s≤0) routes it to coneArmSurface with the apex shift flipped
// to s=−1 (A′ = A − r/sinα·â) but the SAME plane-into-material offset the boss (s=+1) case uses — NOT the
// closed-rim concaveConeArmSurface, which is for a different, genuinely edge-CONCAVE construction (a bump
// meeting a plate, ball in the void, both offsets flipped). Pinned against OCCT's own I1 oracle number
// (DRAWEXE `blend result s 10 s_4` on the bore rim: apex(-200,0,-200) â=(0,0,1) tanα=1, r=10) — the
// verified-correct torus is centre(-200,0,10), major 224.142135623731 = 200+10(1+√2), minor 10; a naive
// BOTH-flipped (concaveConeArmSurface) construction gives centre(-200,0,-10), major 204.142 instead,
// 20 (=2r) short — the mutation witness below.
func TestConeArm_ConcaveBoreBuilds(t *testing.T) {
	t.Parallel()
	e, co, pl := conePlaneFixtureEdge(t, true) // reversed cone face → concave bore
	coneFace, planeFace := boreFaces(e)
	res := ResolutionForSize(300)
	nOut, _ := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	ef, reason := coneArmFillet(e, co, pl, coneFace, planeFace, filletPick{edge: e, r0: 10, r1: 10}, res)
	if reason != coneArmBuilt {
		t.Fatalf("concave bore was rejected: reason %d, want built (I1's follow-on)", reason)
	}
	if ef.armConcave {
		t.Fatalf("concave bore arm must NOT be armConcave — the edge is convex, the ball rolls in the " +
			"material, and the ordinary (non-void) single-arm runout weld must handle it")
	}
	tor, ok := ef.armSurface.(geom.Torus)
	if !ok {
		t.Fatalf("concave bore arm is %T, want geom.Torus", ef.armSurface)
	}
	// Mutation witness: the genuinely-edge-concave (both-flipped) construction gives a DIFFERENT torus —
	// proof coneArmFilletConcave must call coneArmSurface(s=-1), not concaveConeArmSurface.
	if wrongTor, wrongReason := concaveConeArmSurface(co, pl, nOut, 10, res); wrongReason == coneArmBuilt &&
		tor.Center == wrongTor.Center && tor.MajorRadius == wrongTor.MajorRadius {
		t.Fatalf("concave bore arm %+v matches the both-flipped concaveConeArmSurface construction — "+
			"coneArmFilletConcave must use coneArmSurface(s=-1), the material-side plane offset", tor)
	}
	wantTor, wantReason := coneArmSurface(co, pl, nOut, -1, 10, res)
	if wantReason != coneArmBuilt || tor.Center != wantTor.Center || tor.MajorRadius != wantTor.MajorRadius {
		t.Fatalf("coneArmFillet's concave-bore arm %+v does not match coneArmSurface(s=-1) directly %+v", tor, wantTor)
	}
	if _, _, err := coneArmEdge(nil, e, filletPick{edge: e, r0: 10, r1: 10}); err != nil {
		t.Fatalf("concave bore should build without error via coneArmEdge, got: %v", err)
	}
}

// TestConeArmSurface_NearCylinderRejects is the α→0 existence guard: a near-zero half-angle makes the
// apex shift r/sinα blow up (a true cylinder host, handled by M5). The guard rejects with the SPECIFIC
// near-cylinder reason; deleting it lets the huge apex shift fall through to a coneArmClears reject
// (h′ < 0) — the reason-flip mutation proof that this guard is load-bearing.
func TestConeArmSurface_NearCylinderRejects(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	apex, capZ := math.P3(0, 0, 270), 0.0
	aband := coneAlphaBandCoef * res.Weld() / float64(apex.DistanceTo(math.P3(0, 0, capZ)))
	tiny, _ := geom.NewCone(apex, coneAxisDown(), stdmath.Asin(0.5*aband)) // sinα = 0.5·band < band
	pl := planeOn(t, math.P3(0, 0, capZ), math.V3(0, 0, -1))
	n, _ := math.UnitVector3FromVector(math.V3(0, 0, -1))
	if _, reason := coneArmSurface(tiny, pl, n, +1, 10, res); reason != coneArmNearCylinder {
		t.Fatalf("near-cylinder cone (sinα below band) not rejected as near-cylinder: reason %d", reason)
	}
	ok, _ := geom.NewCone(apex, coneAxisDown(), stdmath.Atan(1.0/3.0)) // control: a real cone builds
	if _, reason := coneArmSurface(ok, pl, n, +1, 10, res); reason != coneArmBuilt {
		t.Fatalf("control cone (α=atan⅓) should build, got reason %d", reason)
	}
}

// TestConeArmSurface_NearPlaneRejects is the α→π/2 existence guard: a near-plane cone (cosα below band)
// is rejected with the SPECIFIC near-plane reason; deleting it lets tanα·h′ build a giant-major torus —
// the reason-flip mutation proof.
func TestConeArmSurface_NearPlaneRejects(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	apex, capZ := math.P3(0, 0, 270), 0.0
	aband := coneAlphaBandCoef * res.Weld() / float64(apex.DistanceTo(math.P3(0, 0, capZ)))
	flat, _ := geom.NewCone(apex, coneAxisDown(), stdmath.Pi/2-stdmath.Asin(0.5*aband)) // cosα = 0.5·band < band
	pl := planeOn(t, math.P3(0, 0, capZ), math.V3(0, 0, -1))
	n, _ := math.UnitVector3FromVector(math.V3(0, 0, -1))
	if _, reason := coneArmSurface(flat, pl, n, +1, 10, res); reason != coneArmNearPlane {
		t.Fatalf("near-plane cone (cosα below band) not rejected as near-plane: reason %d", reason)
	}
}

// TestConeArmSurface_GrazingRejects is the R_s→0 existence guard: a cap plane placed so the spine radius
// R_s = tanα·h′ falls below the model band collapses the torus to a point (a grazing/tangent cap). The
// guard rejects with the SPECIFIC grazing reason (h′ still > 0, so it passes coneArmClears); a control
// cap a hair further out builds — a straddle proof the band is real, not vacuous.
func TestConeArmSurface_GrazingRejects(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	co, _ := geom.NewCone(math.P3(0, 0, 270), coneAxisDown(), stdmath.Atan(1.0/3.0))
	sinA := stdmath.Sin(co.HalfAngle)
	apexPrimeZ := 270 - 10/sinA // A′z = 270 − r/sinα (â = −ẑ)
	band := armSpindleBand * res.Weld()
	graze := grazeCapPlane(t, apexPrimeZ, 2*band) // h′ = 2·band → R_s = tanα·2band < band
	build := grazeCapPlane(t, apexPrimeZ, 100)    // h′ = 100 → R_s ≈ 33 ≫ band
	n, _ := math.UnitVector3FromVector(math.V3(0, 0, -1))
	if _, reason := coneArmSurface(co, graze, n, +1, 10, res); reason != coneArmGrazing {
		t.Fatalf("grazing cap (R_s below band) not rejected as grazing: reason %d", reason)
	}
	if _, reason := coneArmSurface(co, build, n, +1, 10, res); reason != coneArmBuilt {
		t.Fatalf("control cap (h′=100) should build, got reason %d", reason)
	}
}

// grazeCapPlane returns a cap plane (outward −ẑ) placed so the offset-plane height above A′ is exactly
// hPrime: P_off z = A′z − hPrime, and P_off = pl.Origin + r·ẑ ⇒ pl.Origin z = A′z − hPrime − r.
func grazeCapPlane(t *testing.T, apexPrimeZ, hPrime float64) geom.Plane {
	t.Helper()
	return planeOn(t, math.P3(0, 0, apexPrimeZ-hPrime-10), math.V3(0, 0, -1))
}

// TestClassifyConeArm_RulingRejects: the ruling edge (a plane CONTAINING the axis, n̂⊥â) is the canal
// arm (CN2), which classifyConeArm reports as coneClassRuling and coneArmFillet honest-rejects — it must
// NOT half-build a torus. An oblique plane reports coneClassOblique.
func TestClassifyConeArm_RulingRejects(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(300)
	co, _ := geom.NewCone(math.P3(0, 0, 270), coneAxisDown(), stdmath.Atan(1.0/3.0))
	cap := planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, -1))    // ⊥ axis
	ruling := planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0))  // contains the axis (n̂ ⊥ â)
	oblique := planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 1)) // neither
	rEdge := 90.0                                             // the C2 bottom-rim radius (the model-relative band divisor)
	if got := classifyConeArm(co, cap, rEdge, res); got != coneClassTorus {
		t.Fatalf("cap plane ⊥ axis classified %d, want coneClassTorus", got)
	}
	if got := classifyConeArm(co, ruling, rEdge, res); got != coneClassRuling {
		t.Fatalf("ruling plane ∥ axis classified %d, want coneClassRuling", got)
	}
	if got := classifyConeArm(co, oblique, rEdge, res); got != coneClassOblique {
		t.Fatalf("oblique plane classified %d, want coneClassOblique", got)
	}
}

// conePlaneFixtureEdge builds a minimal Cone∧Plane cap (circle) edge: host cone (apex (0,0,270), axis
// −ẑ, tanα=1/3), cap plane z=0, on the bottom rim circle radius 90. When reversed is true the cone face
// is Reversed (material OUTSIDE the cone — a conical bore) for the concave-bore gate test.
func conePlaneFixtureEdge(t *testing.T, reversed bool) (*topo.Edge, geom.Cone, geom.Plane) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "cone-cap-edge", 0))
	bld := topo.NewBuilder(true, lin)
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 90, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("rim arc: %v", err)
	}
	e := bld.AddEdge(arc, bld.AddVertex(arc.PointAt(0), lin), bld.AddVertex(arc.PointAt(stdmath.Pi/2), lin), lin)
	co, err := geom.NewCone(math.P3(0, 0, 270), coneAxisDown(), stdmath.Atan(1.0/3.0))
	if err != nil {
		t.Fatalf("cone: %v", err)
	}
	if reversed {
		bld.AddReversedFace(co, lin, topo.OuterLoop(topo.Fwd(e))) // material OUTSIDE the cone (bore)
	} else {
		bld.AddFace(co, lin, topo.OuterLoop(topo.Fwd(e)))
	}
	pl := planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, -1))
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Rev(e)))
	return e, co, pl
}

// boreFaces returns the (cone, plane) faces of a conePlaneFixtureEdge in a stable order.
func boreFaces(e *topo.Edge) (coneFace, planeFace *topo.Face) {
	for _, f := range e.Faces() {
		if _, ok := f.Geometry().(geom.Cone); ok {
			coneFace = f
		} else {
			planeFace = f
		}
	}
	return coneFace, planeFace
}
