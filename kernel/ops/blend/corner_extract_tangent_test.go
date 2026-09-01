// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The T-N7.2 tangent-degenerate corner test bed. The N7 corner (wall cylinder R=50 about (50,50),
// plane x=50 through the wall axis, plane z=10, rolling radius r=5) is the case where the three offset
// spines do NOT concur: the corner ball is rooted at C=(45,√-partition,15) on s_4's spine, and its
// reflections C′=(55,·,15), C″=(55,·,5) sit on s_10's and s_5's spines. Every assertion is against the
// DRAWEXE oracle result_5 (blend of N7.step edges 7/4/9, r=5): area 90.194 and the four vertices
// V0=(55.556,0.310,5), V1=(55,5.279,10), V2=(50,5.279,15), V3=(44.444,0.310,15).

// n7CornerFill builds the N7 corner as extractCurvedCorner consumes it: the rooted corner ball centre
// C + radius, and the three curved arms wired to their two HOST faces (wall cylinder + the two planes).
// solveCurvedCorner cannot produce this weld (C is off two of the three spines — the very degeneracy),
// so the fixture supplies w directly; only w.center, w.radius and the arms' hosts/surfaces are read.
func n7CornerFill(t *testing.T) (cornerWeld, []edgeFillet, opstol.Resolution) {
	t.Helper()
	y := 50 - stdmath.Sqrt(2000) // 5.27864… — s_4 spine y where x=45 meets the R−r=45 offset circle
	bld := topo.NewBuilder(true, topo.Lineage{})
	// The three host faces carry real OUTER loops (W2): the cylinder-arm contact rulings (s_4 vertical,
	// s_10 axial) exit these loops via armHostContactRail's ruling terminator, and each cylinder arm's
	// two host rails terminate at ONE far station (wall/fx50 both z=-60, fx50/fz10 both y=-60) so its far
	// cross-section arc is a clean constant-station section. The s_5 torus arm reads its hosts by curved
	// geometry (curvedHostArc), so it needs no loop.
	fWall := n7WallHost(t, bld)
	fx50 := n7PlaneHost(t, bld, mustPlane(t, math.P3(50, 0, 0), math.V3(1, 0, 0)),
		[]math.Point3{math.P3(50, -60, -60), math.P3(50, 70, -60), math.P3(50, 70, 70), math.P3(50, -60, 70)})
	fz10 := n7PlaneHost(t, bld, mustPlane(t, math.P3(0, 0, 10), math.V3(0, 0, 1)),
		[]math.Point3{math.P3(0, -60, 10), math.P3(110, -60, 10), math.P3(110, 70, 10), math.P3(0, 70, 10)})
	s4 := mustCylinder(t, math.P3(45, y, 0), math.V3(0, 0, 1), 5)   // wall ∧ x=50 straight arm
	s10 := mustCylinder(t, math.P3(55, 0, 15), math.V3(0, 1, 0), 5) // x=50 ∧ z=10 straight arm
	s5, err := geom.NewTorusWithRef(math.P3(50, 50, 5), math.V3(0, 0, 1), math.V3(1, 0, 0), 45, 5)
	if err != nil {
		t.Fatalf("build s_5 torus arm: %v", err)
	}
	arms := []edgeFillet{
		{a: fWall, b: fx50, armSurface: s4},
		{a: fx50, b: fz10, armSurface: s10},
		{a: fWall, b: fz10, armSurface: s5},
	}
	return cornerWeld{center: math.P3(45, y, 15), radius: 5}, arms, geom.ResolutionForSize(150)
}

// n7WallHost builds the N7 wall cylinder (R=50 about (50,50), axis z) with an outer-loop BAND the s_4
// arm's vertical contact ruling exits: two z-rims (z=-60 far, z=60 near) over the corner azimuth
// [-140°,-40°], closed by two vertical edges. The s_4 ruling exits the FAR rim at z=-60 — matched to
// fx50's z=-60 edge so the arm's far cross-section arc is a clean constant-z section.
func n7WallHost(t *testing.T, bld *topo.Builder) *topo.Face {
	t.Helper()
	lin := topo.Lineage{}
	axis, ref := math.V3(0, 0, 1), math.V3(1, 0, 0)
	deg := stdmath.Pi / 180
	bot := mustArcRef(t, math.P3(50, 50, -60), axis, ref, 50, -140*deg, 100*deg) // z=-60, θ:-140°→-40°
	top := mustArcRef(t, math.P3(50, 50, 60), axis, ref, 50, -40*deg, -100*deg)  // z=60, θ:-40°→-140°
	a0, a1 := bld.AddVertex(bot.PointAt(0), lin), bld.AddVertex(bot.PointAt(1), lin)
	a2, a3 := bld.AddVertex(top.PointAt(0), lin), bld.AddVertex(top.PointAt(1), lin)
	be := bld.AddEdge(bot, a0, a1, lin)
	right := bld.AddEdge(geom.NewLineSegment(a1.Point(), a2.Point()), a1, a2, lin)
	te := bld.AddEdge(top, a2, a3, lin)
	left := bld.AddEdge(geom.NewLineSegment(a3.Point(), a0.Point()), a3, a0, lin)
	cyl := mustCylinder(t, math.P3(50, 50, 0), math.V3(0, 0, 1), 50)
	return bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(be), topo.Fwd(right), topo.Fwd(te), topo.Fwd(left)))
}

// n7PlaneHost builds a plane host face with a straight-edged rectangular outer loop through corners (in
// order) — a real loop for the cylinder-arm ruling terminator to cross.
func n7PlaneHost(t *testing.T, bld *topo.Builder, pl geom.Plane, corners []math.Point3) *topo.Face {
	t.Helper()
	lin := topo.Lineage{}
	verts := make([]*topo.Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(corners))
	for i := range corners {
		a, b := verts[i], verts[(i+1)%len(corners)]
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, lin))
	}
	return bld.AddFace(pl, lin, topo.OuterLoop(uses...))
}

// mustArcRef builds an Arc3d with an explicit centre/axis/ref (unlike mustArc's z-constant convention),
// or fails the test.
func mustArcRef(t *testing.T, center math.Point3, axis, ref math.Vector3, r, start, sweep float64) geom.Arc3d {
	t.Helper()
	arc, err := geom.NewArc3d(center, axis, ref, r, start, sweep)
	if err != nil {
		t.Fatalf("build arc: %v", err)
	}
	return arc
}

// TestExtractTangentCorner_IsValence4Canal is the slice's core gate (M6' C3): the N7 corner must
// extract as a CLOSED valence-4 RailLoop that resolveBlend now fills with the rolling-ball CANAL tier
// (BlendKindCanal, Canal-payload-marked), whose EMERGENT area matches the DRAWEXE oracle 90.194 within
// the C2-justified 0.05% (the canal is the RIGHT surface for this tangent-degenerate corner; coons4
// was the do-no-harm fallback that held it to 1% before the canal solve landed).
func TestExtractTangentCorner_IsValence4Canal(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 {
		t.Fatalf("extractCurvedCorner: want a closed valence-4 N7 loop; ok=%v valence=%d", ok, loop.Valence())
	}
	if !loop.Closed(res.Weld() * 50) {
		t.Fatalf("N7 RailLoop is not closed")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCanal {
		t.Fatalf("N7 must resolve to the canal tier; ok=%v kind=%q", ok, patch.Kind)
	}
	area := geom.SurfaceArea(patch.Surface)
	if e := stdmath.Abs(area-90.194) / 90.194; e > 5e-4 {
		t.Fatalf("N7 canal fill area = %.5f, want 90.194 within 0.05%% (off %.4f%%)", area, 100*e)
	}
}

// TestExtractTangentCorner_ReproducesOracleVertices asserts the four rail corners reproduce OCCT's
// result_5 vertices V0..V3 exactly (to res.Weld·r), the proof that the reflected-family rails are the
// right rails (not the octant great arcs, which would miss two of the four).
func TestExtractTangentCorner_ReproducesOracleVertices(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok {
		t.Fatalf("extractCurvedCorner declined the N7 corner")
	}
	want := []math.Point3{
		math.P3(55.5555556, 0.3096005, 5), math.P3(55, 5.2786405, 10),
		math.P3(50, 5.2786405, 15), math.P3(44.4444444, 0.3096005, 15),
	}
	tol := res.Weld() * w.radius
	for _, wv := range want {
		if !loopHasCorner(loop, wv, tol) {
			t.Fatalf("no rail corner within %.2e of oracle vertex %v", tol, wv)
		}
	}
}

// TestExtractTangentCorner_BridgeIsOnWall is the INDEPENDENT on-wall check the area gate cannot give:
// coons4's ParamAt silently re-projects an off-wall rail, so watertightness with the retrimmed wall
// host (T-N7.3) depends on E2 being genuinely on the wall. Every sample of E2 (the one BSpline rail)
// must sit within res.Weld·R of radius 50 about the wall axis.
func TestExtractTangentCorner_BridgeIsOnWall(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok {
		t.Fatalf("extractCurvedCorner declined the N7 corner")
	}
	e2, ok := wallBridgeCurve(loop)
	if !ok {
		t.Fatalf("no on-wall bridge (BSpline) rail found in the N7 loop")
	}
	axis, origin := math.V3(0, 0, 1), math.P3(50, 50, 0)
	tol := res.Weld() * 50
	maxOff := 0.0
	for i := 0; i <= 400; i++ {
		p := e2.PointAt(float64(i) / 400)
		radial := origin.VectorTo(p)
		d := float64(radial.Sub(axis.Scale(radial.Dot(axis))).Length())
		maxOff = stdmath.Max(maxOff, stdmath.Abs(d-50))
	}
	if maxOff > tol {
		t.Fatalf("E2 off-wall by %.3e, want ≤ %.3e (res.Weld·R) — the bridge is NOT on the wall", maxOff, tol)
	}
	t.Logf("E2 max off-wall = %.3e (tol %.3e)", maxOff, tol)
}

// loopHasCorner reports whether any rail start point is within tol of q.
func loopHasCorner(loop RailLoop, q math.Point3, tol float64) bool {
	for _, s := range loop.Sides {
		if float64(curveStart(s.Curve).DistanceTo(q)) <= tol {
			return true
		}
	}
	return false
}

// wallBridgeCurve returns the loop's on-wall bridge — the single BSpline rail (the arcs are Arc3d).
func wallBridgeCurve(loop RailLoop) (geom.BSplineCurve, bool) {
	for _, s := range loop.Sides {
		if c, ok := s.Curve.(geom.BSplineCurve); ok {
			return c, true
		}
	}
	return geom.BSplineCurve{}, false
}
