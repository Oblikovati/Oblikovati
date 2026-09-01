// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// n1VerticalWallEdge is N1's picked Cyl∧Plane LINE edge: the vertical ruling where the y=0 plane meets
// the notch wall, at (80,0,z). A bare edge (no faces) is enough — cylinderArmSurface reads only its
// endpoints to select which of the two P_r∩C_ρ rulings the fillet sits on (nearerRuling).
func n1VerticalWallEdge(t *testing.T) *topo.Edge {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "n1wall", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(80, 0, 0), lin)
	hi := bld.AddVertex(math.P3(80, 0, 100), lin)
	return bld.AddEdge(geom.NewLineSegment(math.P3(80, 0, 0), math.P3(80, 0, 100)), lo, hi, lin)
}

// The R+r BORE/NOTCH-wall corner foundation (corner-blend-weld). The pre-foundation curved-corner
// engine hard-coded R−r (a convex boss cap), so on a concave/notch wall — material OUTSIDE the
// cylinder — it mirrored the whole corner into the removed void and the weld declined. These tests
// pin the ε=−1 arm surfaces and corner sphere to the OCCT 8.0.0 DRAWEXE values for N1
// (box − r20 cylinder notch at a corner, r=5 fillet on 3 edges), so the foundation math cannot
// silently regress before the downstream weld pieces green the end-to-end case. The paired ε=+1
// assertions are the do-no-harm guard: the SAME geometry on the boss sense must still give R−r.

// n1NotchWall is N1's cut cylinder (radius 20 at (100,0), axis ẑ) — the concave notch wall whose
// material lies OUTSIDE the cylinder, so the rolling ball is tangent to it from the material side at
// R+r (the DRAWEXE oracle: result_3 torus Radii 25 5, result_10 cyl axis (75.5051,5)).
func n1NotchWall(t *testing.T) geom.Cylinder {
	t.Helper()
	c, err := geom.NewCylinder(math.P3(100, 0, -10), math.V3(0, 0, 1), 20)
	if err != nil {
		t.Fatalf("n1NotchWall: %v", err)
	}
	return c
}

// TestTorusArmSurface_N1Bore pins the ε=−1 torus arm to N1's OCCT torus (`Radii 25 5`, origin
// (100,0,95)): a notch wall gives major R+r = 25 (contacting the wall at its INNER equator
// major−minor = 20 = R), minor r = 5, centre r below the z=100 cap. ε=+1 on the SAME wall must give
// the boss major R−r = 15 (the do-no-harm flip guard).
func TestTorusArmSurface_N1Bore(t *testing.T) {
	t.Parallel()
	res := testArmResolution()
	wall := n1NotchWall(t)
	tor, ok := torusArmSurface(wall, planeAtZ(100), armOutward(0, 0, 1), 5, -1, res)
	if !ok {
		t.Fatalf("torusArmSurface (bore ε=−1) declined N1's notch rim")
	}
	if !nearlyArm(tor.MajorRadius, 25) || !nearlyArm(tor.MinorRadius, 5) ||
		!nearlyArm(float64(tor.Center.X), 100) || !nearlyArm(float64(tor.Center.Z), 95) {
		t.Fatalf("N1 bore torus = {major %.6f, minor %.6f, centre %v}, want {25,5,(100,0,95)} (OCCT Radii 25 5)",
			tor.MajorRadius, tor.MinorRadius, tor.Center)
	}
	boss, _ := torusArmSurface(wall, planeAtZ(100), armOutward(0, 0, 1), 5, 1, res)
	if !nearlyArm(boss.MajorRadius, 15) {
		t.Fatalf("do-no-harm: ε=+1 on the same wall = major %.6f, want R−r = 15", boss.MajorRadius)
	}
}

// TestCylinderArmSurface_N1Bore pins the ε=−1 cylinder arm (the vertical y=0∧wall fillet) to N1's OCCT
// cylinder (result_10 axis (75.5051,5), radial R+r = 25 from the wall axis (100,0)): a radius-5
// cylinder whose axis is ∥ ẑ and sits 25 from the wall axis. ε=+1 must place it at R−r = 15 (the void
// side the pre-foundation code wrongly built).
func TestCylinderArmSurface_N1Bore(t *testing.T) {
	t.Parallel()
	wall := n1NotchWall(t)
	edge := n1VerticalWallEdge(t)
	y0 := planeWithNormal(0, -1, 0) // y=0 plane, material-outward normal −ŷ (material at y>0)
	arm, ok := cylinderArmSurface(edge, wall, y0, armOutward(0, -1, 0), 5, -1)
	if !ok || !nearlyArm(arm.Radius, 5) {
		t.Fatalf("N1 bore cylinder arm radius = %.6f (ok=%v), want 5", arm.Radius, ok)
	}
	if !nearlyArm(stdmath.Abs(arm.AxisDir.AsVector().Dot(math.V3(0, 0, 1))), 1) {
		t.Fatalf("N1 bore cylinder arm axis = %v, want ∥ ẑ", arm.AxisDir)
	}
	if got := axisRadialFromWall(arm, wall); !nearlyArm(got, 25) {
		t.Fatalf("N1 bore cylinder arm axis radial = %.6f from the wall axis, want R+r = 25", got)
	}
	boss, _ := cylinderArmSurface(edge, wall, y0, armOutward(0, -1, 0), 5, 1)
	if got := axisRadialFromWall(boss, wall); !nearlyArm(got, 15) {
		t.Fatalf("do-no-harm: ε=+1 arm axis radial = %.6f, want R−r = 15", got)
	}
}

// TestCurvedCornerCenter_N1Bore drives the REAL corner solve on the N1 corpus fixture: the notch
// wall must be detected as a bore (cornerWallRadialSign = −1) and the corner sphere centre must land
// on the material side at (75.505,5,95) — R+r = 25 from the wall axis (100,0) — matching OCCT's
// corner KPart, NOT the pre-foundation void-side mirror (85.857,5,95) at R−r = 15.
func TestCurvedCornerCenter_N1Bore(t *testing.T) {
	t.Parallel()
	cyl, planes, v, r := cornerHostInputs(t, "simple/N1", math.P3(80, 0, 100), 5)
	if eps := cornerWallRadialSign(facesAtVertex(v), cyl, v.Point()); eps != -1 {
		t.Fatalf("cornerWallRadialSign on N1's notch = %v, want −1 (a bore/notch wall)", eps)
	}
	res := curvedCornerResolution(v, cyl, planes)
	c, ok := curvedCornerCenter(cyl, planes, r, -1, v, res)
	if !ok || !nearlyPt(c, math.P3(75.5051025721682, 5, 95)) {
		t.Fatalf("N1 bore corner centre = %v (ok=%v), want (75.5051,5,95) = R+r material side, NOT the void mirror (85.857,5,95)", c, ok)
	}
}

// axisRadialFromWall is the perpendicular distance from arm's axis line to the wall axis (both ∥ ẑ):
// the arm's radial offset, R∓r depending on the material side.
func axisRadialFromWall(arm, wall geom.Cylinder) float64 {
	a := wall.AxisDir.AsVector()
	w := wall.Origin.VectorTo(arm.Origin)
	return float64(w.Sub(a.Scale(w.Dot(a))).Length())
}
