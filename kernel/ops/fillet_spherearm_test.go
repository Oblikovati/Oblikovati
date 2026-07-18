// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The D5/D9/E4 sphere-host fixtures are all psphere R=150, fillet r=10, sphere centre at the origin
// (the derivation established v−O = v). ρ = R−r = 140. These pin sphereArmSurface's closed form to
// the DRAWEXE oracle tori (sphere-host-corner-derivation.md §3): each is an exact torus that OCCT
// approximates as a rational BSpline, matched to ≤2.3e-4 — so a centre/major/minor off by more than
// nearlyArm is a transcription bug, not imprecision.
const sphereHostR = 150

// sphereArmTest is a Sphere∧Plane arm case: the host sphere, the plane (origin + material-outward
// normal), and the DRAWEXE-verified spine circle the torus arm must reproduce.
type sphereArmTest struct {
	name        string
	planeOrigin math.Point3
	outwardN    math.Vector3
	wantCenter  math.Point3
	wantAxis    math.Vector3
	wantMajor   float64
}

// d5r150 is the host ball of every fixture: centre origin, R=150 (psphere 15, tscale ×10).
func d5r150(t *testing.T) geom.Sphere {
	t.Helper()
	sp, err := geom.NewSphere(math.P3(0, 0, 0), sphereHostR)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	return sp
}

// planeOn builds a plane through origin with the given (material-outward) normal.
func planeOn(t *testing.T, o math.Point3, n math.Vector3) geom.Plane {
	t.Helper()
	pl, err := geom.NewPlane(o, n)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	return pl
}

// sqrt is a terse helper for the closed-form major radii in the expectations.
func sqrt(x float64) float64 { return stdmath.Sqrt(x) }

// TestSphereArmSurface_Oracle pins the torus arm of every sphere-host fixture edge to its DRAWEXE
// oracle spine circle (sphere-host-corner-derivation.md §3): centre, axis, and major radius are exact
// closed forms of {r, R, plane}, and minor = r always. A wrong material-side sign (ρ = R+r, or the
// centre foot on the wrong side) is caught here — the meridian/rim centres straddle the origin.
func TestSphereArmSurface_Oracle(t *testing.T) {
	cases := []sphereArmTest{
		// D5: longitude plane y=0 (outward −ŷ), cap plane z=75√3 (outward +ẑ).
		{"D5-meridian", math.P3(0, 0, 0), math.V3(0, -1, 0), math.P3(0, 10, 0), math.V3(0, 1, 0), sqrt(19500)},
		{"D5-rim", math.P3(0, 0, 129.9038105676658), math.V3(0, 0, 1), math.P3(0, 0, 119.9038105676658), math.V3(0, 0, 1), sqrt(5223.0761890)},
		// D9: longitude plane x=0 (outward +x̂), same cap plane.
		{"D9-meridian", math.P3(0, 0, 0), math.V3(1, 0, 0), math.P3(-10, 0, 0), math.V3(1, 0, 0), sqrt(19500)},
		{"D9-rim", math.P3(0, 0, 129.9038105676658), math.V3(0, 0, 1), math.P3(0, 0, 119.9038105676658), math.V3(0, 0, 1), sqrt(5223.0761890)},
		// E4 (the pole corner): the two longitude planes x=0 (outward +x̂) and y=0 (outward −ŷ).
		{"E4-meridian-x", math.P3(0, 0, 0), math.V3(1, 0, 0), math.P3(-10, 0, 0), math.V3(1, 0, 0), sqrt(19500)},
		{"E4-meridian-y", math.P3(0, 0, 0), math.V3(0, -1, 0), math.P3(0, 10, 0), math.V3(0, 1, 0), sqrt(19500)},
	}
	sp := d5r150(t)
	res := ResolutionForSize(300) // the sphere-body diagonal scale
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertSphereArm(t, sp, c, res)
		})
	}
}

// assertSphereArm builds one arm and checks its torus centre, axis, major and minor against the oracle.
func assertSphereArm(t *testing.T, sp geom.Sphere, c sphereArmTest, res Resolution) {
	t.Helper()
	pl := planeOn(t, c.planeOrigin, c.outwardN)
	n, _ := math.UnitVector3FromVector(c.outwardN)
	tor, reason := sphereArmSurface(sp, pl, n, 10, res)
	if reason != sphereArmBuilt {
		t.Fatalf("%s: sphereArmSurface declined a valid convex arm (reason %d)", c.name, reason)
	}
	if !nearlyArm(tor.MinorRadius, 10) || !nearlyArm(tor.MajorRadius, c.wantMajor) {
		t.Fatalf("%s torus radii = {major %.7f, minor %.7f}, want {%.7f, 10}", c.name, tor.MajorRadius, tor.MinorRadius, c.wantMajor)
	}
	if d := float64(tor.Center.DistanceTo(c.wantCenter)); d > 1e-6 {
		t.Fatalf("%s torus centre = %v, want %v (off by %.3g)", c.name, tor.Center, c.wantCenter, d)
	}
	wantAxis, _ := math.UnitVector3FromVector(c.wantAxis)
	if !nearlyArm(stdmath.Abs(tor.AxisDir.Dot(wantAxis)), 1) {
		t.Fatalf("%s torus axis = %v, want ∥ %v", c.name, tor.AxisDir, c.wantAxis)
	}
}

// TestSphereArmSurface_Spindle is the existence guard: r ≥ R engulfs the host (ρ = R−r ≤ 0), a
// self-intersecting spindle torus, which must honest-reject rather than emit degenerate geometry.
func TestSphereArmSurface_Spindle(t *testing.T) {
	sp, err := geom.NewSphere(math.P3(0, 0, 0), 8)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	pl := planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	n, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	if _, reason := sphereArmSurface(sp, pl, n, 10, ResolutionForSize(50)); reason != sphereArmSpindle {
		t.Fatalf("sphereArmSurface accepted r=10 ≥ R=8: want a spindle reject, got reason %d", reason)
	}
}

// TestSphereArmSurface_Clears is the existence guard: a plane whose offset clears the offset sphere
// (|h| ≥ ρ) leaves no spine circle — the ball cannot touch both surfaces — so the arm rejects.
func TestSphereArmSurface_Clears(t *testing.T) {
	sp := d5r150(t)
	n, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	far := planeOn(t, math.P3(0, 0, 200), math.V3(0, 0, 1)) // offset plane at z=190 > ρ=140: |h| ≥ ρ
	if _, reason := sphereArmSurface(sp, far, n, 10, ResolutionForSize(300)); reason != sphereArmClears {
		t.Fatalf("sphereArmSurface accepted a plane that clears the offset sphere (|h|=190 > ρ=140), reason %d", reason)
	}
}

// TestSpherePlaneEdge recognizes an edge flanked by a sphere face and a plane face, returning both
// surfaces and the plane's face; a plane∧plane edge (the third, straight edge) is not recognized.
func TestSpherePlaneEdge(t *testing.T) {
	spEdge, planeGeom := spherePlaneFixtureEdge(t)
	sp, pl, sf, pf, ok := spherePlaneEdge(spEdge)
	if !ok {
		t.Fatalf("spherePlaneEdge did not recognize a sphere∧plane edge")
	}
	if sp.Radius != sphereHostR || pl.Origin != planeGeom.Origin || sf == nil || pf == nil {
		t.Fatalf("spherePlaneEdge returned {R=%g, planeOrigin=%v, sphereFace=%v, planeFace=%v}, want {150, %v, non-nil, non-nil}",
			sp.Radius, pl.Origin, sf, pf, planeGeom.Origin)
	}
	if _, ok := sf.Geometry().(geom.Sphere); !ok {
		t.Fatalf("spherePlaneEdge returned a non-sphere as the sphere face: %T", sf.Geometry())
	}
	if _, _, _, _, ok := spherePlaneEdge(planePlaneFixtureEdge(t)); ok {
		t.Fatalf("spherePlaneEdge wrongly recognized a plane∧plane edge")
	}
}

// TestSphereArmEdge_Dispatches is the integration guard: computeEdgeFillet's sphere branch HANDLES a
// sphere∧plane edge (handled=true) so it returns before reaching curvedAdjacentError — the old
// "cannot round an edge bordering a curved (sphere) face" reject is gone for these edges. A plane∧plane
// edge is NOT handled here (handled=false), so it still flows to the existing planar path.
func TestSphereArmEdge_Dispatches(t *testing.T) {
	spEdge, _ := spherePlaneFixtureEdge(t)
	if _, handled, _ := sphereArmEdge(nil, spEdge, filletPick{edge: spEdge, r0: 10, r1: 10}); !handled {
		t.Fatalf("sphereArmEdge left a sphere∧plane edge to curvedAdjacentError (want handled=true)")
	}
	pp := planePlaneFixtureEdge(t)
	if _, handled, _ := sphereArmEdge(nil, pp, filletPick{edge: pp, r0: 10, r1: 10}); handled {
		t.Fatalf("sphereArmEdge wrongly handled a plane∧plane edge (want handled=false, keep the planar path)")
	}
}

// TestSphereArm_ConcaveDimpleHostRejects is the SP1-review regression: a spherical DIMPLE rim — the
// sphere face is Reversed (material OUTSIDE the sphere, a deburred countersink meeting a plate top) —
// is a genuinely CONVEX edge, so the old naked convexity guard PASSED it and sphereArmSurface silently
// built a wrong-side R−r torus (major √(140²−110²) = 86.6025 for O=(0,0,100) R=150 plane z=0 r=10,
// whereas the true concave host needs ρ = R+r → major 116.1895, Δ 29.59). The host material-side gate
// (sphereHostMaterialSign) must HONEST-REJECT it (s ≤ 0), naming the concave host — no torus built.
func TestSphereArm_ConcaveDimpleHostRejects(t *testing.T) {
	e := dimpleRimFixtureEdge(t)
	if ClassifyEdgeConvexity(e) != EdgeConvex {
		t.Fatalf("dimple rim must classify CONVEX (the whole point of the regression), got %v", ClassifyEdgeConvexity(e))
	}
	// Mutation witness: the ungated surface builder WOULD build the wrong-side major 86.6025 torus.
	sp, pl, sphereFace, planeFace, _ := spherePlaneEdge(e)
	n, _ := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if tor, reason := sphereArmSurface(sp, pl, n, 10, ResolutionForBody(nil)); reason != sphereArmBuilt || !nearlyArm(tor.MajorRadius, sqrt(7500)) {
		t.Fatalf("mutation witness broken: ungated surface gave reason %d major %.4f, want built major 86.6025", reason, tor.MajorRadius)
	}
	// The gate rejects with the concave-host reason — no arm is built.
	if _, reason := sphereArmFillet(e, sp, pl, sphereFace, planeFace, filletPick{edge: e, r0: 10, r1: 10}, ResolutionForBody(nil)); reason != sphereArmConcaveHost {
		t.Fatalf("concave dimple host was NOT rejected as concave: reason %d (a wrong-side torus would be built)", reason)
	}
	ef, handled, err := sphereArmEdge(nil, e, filletPick{edge: e, r0: 10, r1: 10})
	if !handled || err == nil || ef.armSurface != nil {
		t.Fatalf("dimple rim: want handled+error+no-arm, got handled=%v err=%v armSurface=%v", handled, err, ef.armSurface)
	}
	if !strings.Contains(err.Error(), "CONCAVE") {
		t.Fatalf("reject message does not name the concave host: %q", err.Error())
	}
}

// dimpleRimFixtureEdge builds the concave-dimple rim edge: host sphere O=(0,0,100) R=150 with the
// sphere face REVERSED (material outside), plane z=0, on the rim circle x²+y²=12500 (radius √12500).
// The edge is a short arc about the rim point (√12500,0,0), oriented so the dihedral reads CONVEX.
func dimpleRimFixtureEdge(t *testing.T) *topo.Edge {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "dimple-rim", 0))
	bld := topo.NewBuilder(true, lin)
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), sqrt(12500), -0.2, 0.4)
	if err != nil {
		t.Fatalf("rim arc: %v", err)
	}
	e := bld.AddEdge(arc, bld.AddVertex(arc.PointAt(-0.2), lin), bld.AddVertex(arc.PointAt(0.2), lin), lin)
	sp, err := geom.NewSphere(math.P3(0, 0, 100), sphereHostR)
	if err != nil {
		t.Fatalf("dimple sphere: %v", err)
	}
	bld.AddReversedFace(sp, lin, topo.OuterLoop(topo.Fwd(e))) // material OUTSIDE the sphere
	bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, 1)), lin, topo.OuterLoop(topo.Rev(e)))
	return e
}

// spherePlaneFixtureEdge builds a minimal edge shared by a sphere face and a plane face (enough for
// the recognizer, which reads only the two flanking faces' geometry).
func spherePlaneFixtureEdge(t *testing.T) (*topo.Edge, geom.Plane) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "sp-edge", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(0, 0, 0), lin)
	hi := bld.AddVertex(math.P3(1, 0, 0), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), lo, hi, lin)
	sp := d5r150(t)
	pl := planeOn(t, math.P3(0, 0, 129.9038105676658), math.V3(0, 0, 1))
	bld.AddFace(sp, lin, topo.OuterLoop(topo.Fwd(e)))
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Rev(e)))
	return e, pl
}

// planePlaneFixtureEdge builds an edge shared by two plane faces — the third (straight) corner edge,
// which must keep the existing cylinderPlaneEdge path and not be recognized as a sphere arm.
func planePlaneFixtureEdge(t *testing.T) *topo.Edge {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "pp-edge", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(0, 0, 0), lin)
	hi := bld.AddVertex(math.P3(1, 0, 0), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), lo, hi, lin)
	bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, 1)), lin, topo.OuterLoop(topo.Fwd(e)))
	bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0)), lin, topo.OuterLoop(topo.Rev(e)))
	return e
}
