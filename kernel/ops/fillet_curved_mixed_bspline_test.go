// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// n4TestArms builds the three N4 corner arms from the EXACT surfaces our kernel emits for
// tests/blend/simple/N4 (probed by instrumenting cornerArms), with the three shared host faces wired by
// pointer identity (vplane shared by ccyl+band, boss by ccyl+torus, tplane by torus+band). This lets the
// corner-point solver be tested against the DRAWEXE oracle without importing the STEP fixture.
func n4TestArms(t *testing.T) []edgeFillet {
	t.Helper()
	vplane := mustPlane(t, math.P3(98.480775301221, -17.36481776669, 0), math.V3(0.9848077530121899, -0.1736481776670335, 0))
	boss := mustCylinder(t, math.P3(115.84559306791, 81.115957534528, 0), math.V3(0, 0, 1), 20)
	tplane := mustPlane(t, math.P3(115.84559306791, 81.115957534528, 50), math.V3(0, 0, 1))
	vFace := minimalRoleFace(vplane, 10)
	bFace := minimalRoleFace(boss, 11)
	tFace := minimalRoleFace(tplane, 12)
	ccyl := mustCylinder(t, math.P3(116.51613753250129, 56.12495175002612, 0), math.V3(0, 0, 1), 5)
	torus := mustTorus(t, math.P3(115.84559306791, 81.115957534528, 45), math.V3(0, 0, 1), 15, 5)
	band := mustCylinder(t, math.P3(120.76963183297096, 80.24771664619283, 55), math.V3(-0.17364817768599966, -0.9848077530088457, 0), 5)
	return []edgeFillet{
		{armSurface: ccyl, armConcave: true, a: vFace, b: bFace},
		{armSurface: torus, a: tFace, b: bFace},
		{armSurface: band, flip: true, a: vFace, b: tFace},
	}
}

// TestSolveN4CornerMatchesOracle is the core validation: from OUR arm surfaces the corner-point solver
// reproduces the four DRAWEXE corner points A/B/C/D, and the coons4 patch builds + certifies (NoFold).
func TestSolveN4CornerMatchesOracle(t *testing.T) {
	arms, ok := classifyN4MixedArms(n4TestArms(t))
	if !ok {
		t.Fatal("classifyN4MixedArms rejected the N4 corner")
	}
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 60)})
	corner, ok := solveN4Corner(arms, 5, res)
	if !ok {
		t.Fatal("solveN4Corner declined the N4 corner")
	}
	want := map[string]math.Point3{
		"A": math.P3(113.39, 67.19, 55), "B": math.P3(118.31, 66.32, 50),
		"C": math.P3(116.38, 61.12, 45), "D": math.P3(111.59, 56.99, 45),
	}
	got := map[string]math.Point3{"A": corner.pts.a, "B": corner.pts.b, "C": corner.pts.c, "D": corner.pts.d}
	for k, w := range want {
		if d := float64(got[k].DistanceTo(w)); d > 0.05 {
			t.Errorf("corner %s = %v, want oracle %v (off by %.4f)", k, got[k], w, d)
		}
	}
	if _, isBSpline := corner.patch.Surface.(geom.BSplineSurface); !isBSpline {
		t.Fatalf("corner patch is %T, want a coons4 BSplineSurface", corner.patch.Surface)
	}
}

// TestOnSurfaceRailStaysOnHost checks a fitted contact rail lies on its host — the point-identity weld
// precondition: the torus rail B→C samples on the torus, the vplane rail D→A on the plane.
func TestOnSurfaceRailStaysOnHost(t *testing.T) {
	arms, _ := classifyN4MixedArms(n4TestArms(t))
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 60)})
	corner, ok := solveN4Corner(arms, 5, res)
	if !ok {
		t.Fatal("solveN4Corner declined")
	}
	torus := arms.torus.armSurface.(geom.Torus)
	// The plane rail is exact (a straight in-plane chord); the torus rail is a cubic fit whose between-
	// sample bow is a small fraction of r (2e-4·r), negligible for the point-identity weld.
	assertOnSurface(t, "railDA/vplane", corner.vplane, corner.railDA, 1e-6)
	assertOnSurface(t, "railBC/torus", torus, corner.railBC, 1e-3)
}

// assertOnSurface checks the curve's sampled points lie on surf within tol.
func assertOnSurface(t *testing.T, tag string, surf geom.Surface, c geom.BSplineCurve, tol float64) {
	t.Helper()
	for i := 0; i <= 8; i++ {
		p := c.PointAt(float64(i) / 8)
		_, _, foot := geom.ClosestPointOnSurface(surf, p)
		if d := float64(p.DistanceTo(foot)); d > tol {
			t.Errorf("%s: sample %d is %.2e off the host surface (tol %.0e)", tag, i, d, tol)
		}
	}
}

// TestClassifyN4DeclinesM8Roles is do-no-harm: the N4 classifier must REJECT M8's roles (convex-cyl +
// concave-cove-torus + planar) so the two mixed-corner paths never both fire.
func TestClassifyN4DeclinesM8Roles(t *testing.T) {
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	m8 := []edgeFillet{
		{armSurface: mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5)},                    // convex cyl (M8 pivot)
		{armSurface: mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 30, 5), armConcave: true}, // concave cove
		{flip: true, a: minimalRoleFace(pl, 20), b: minimalRoleFace(pl, 21)},                    // planar band
	}
	if _, ok := classifyN4MixedArms(m8); ok {
		t.Fatal("classifyN4MixedArms accepted M8's roles — the two mixed paths must be disjoint")
	}
}
