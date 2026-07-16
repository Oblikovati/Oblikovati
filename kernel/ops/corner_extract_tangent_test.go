// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
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
func n7CornerFill(t *testing.T) (cornerWeld, []edgeFillet, Resolution) {
	t.Helper()
	y := 50 - stdmath.Sqrt(2000) // 5.27864… — s_4 spine y where x=45 meets the R−r=45 offset circle
	bld := topo.NewBuilder(true, topo.Lineage{})
	fWall := bld.AddFace(mustCylinder(t, math.P3(50, 50, 0), math.V3(0, 0, 1), 50), topo.Lineage{})
	fx50 := bld.AddFace(mustPlane(t, math.P3(50, 0, 0), math.V3(1, 0, 0)), topo.Lineage{})
	fz10 := bld.AddFace(mustPlane(t, math.P3(0, 0, 10), math.V3(0, 0, 1)), topo.Lineage{})
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

// TestExtractTangentCorner_IsValence4Coons4 is the slice's core gate: the N7 corner must extract as a
// CLOSED valence-4 RailLoop that resolveBlend fills with coons4, whose area matches the DRAWEXE oracle
// 90.194 within 1%.
func TestExtractTangentCorner_IsValence4Coons4(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 {
		t.Fatalf("extractCurvedCorner: want a closed valence-4 N7 loop; ok=%v valence=%d", ok, loop.Valence())
	}
	if !loop.Closed(res.Weld() * 50) {
		t.Fatalf("N7 RailLoop is not closed")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCoons4 {
		t.Fatalf("N7 must resolve to coons4; ok=%v kind=%q", ok, patch.Kind)
	}
	area := geom.SurfaceArea(patch.Surface)
	if e := stdmath.Abs(area-90.194) / 90.194; e > 0.01 {
		t.Fatalf("N7 corner fill area = %.5f, want 90.194 within 1%% (off %.3f%%)", area, 100*e)
	}
}

// TestExtractTangentCorner_ReproducesOracleVertices asserts the four rail corners reproduce OCCT's
// result_5 vertices V0..V3 exactly (to res.Weld·r), the proof that the reflected-family rails are the
// right rails (not the octant great arcs, which would miss two of the four).
func TestExtractTangentCorner_ReproducesOracleVertices(t *testing.T) {
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
