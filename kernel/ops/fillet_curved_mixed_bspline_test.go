// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
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
// reproduces the four DRAWEXE corner points A/B/C/D, and the canal patch builds + certifies (NoFold).
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
		t.Fatalf("corner patch is %T, want the rolling-ball canal BSplineSurface", corner.patch.Surface)
	}
}

// TestN4RailsAreBallContactLoci checks both rails lie on their host AND that the vplane rail is the ball's
// CONTACT LOCUS rather than the straight chord the superseded onSurfaceRail produced. The second half is
// the regression guard for the shape defect this construction fixed: projecting a chord onto a PLANE
// returns the chord, so railDA came out exactly linear, and the fill then interpolated a boundary ~21% of
// r away from the oracle. OCCT's own rail bows 3.00 units off its 14.39 chord; the true contact locus is a
// rigid −r·n̂ offset of the ball-centre curve, so it must bow by the same amount.
func TestN4RailsAreBallContactLoci(t *testing.T) {
	arms, _ := classifyN4MixedArms(n4TestArms(t))
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 60)})
	corner, ok := solveN4Corner(arms, 5, res)
	if !ok {
		t.Fatal("solveN4Corner declined")
	}
	torus := arms.torus.armSurface.(geom.Torus)
	// Both rails are the canal's own boundary isoparms. The vplane locus is EXACT (every pole lies in the
	// plane, so the whole interpolant does); the torus locus leaves the torus only by the loft's
	// between-station interpolation error.
	assertOnSurface(t, "railDA/vplane", corner.vplane, corner.railDA, 1e-9)
	assertOnSurface(t, "railBC/torus", torus, corner.railBC, 1e-4)
	if bow := maxChordBow(corner.railDA); bow < 2.5 || bow > 3.5 {
		t.Fatalf("railDA bows %.3f off its chord, want OCCT's 3.00 (a straight chord — bow 0 — is the "+
			"superseded chord-projection defect)", bow)
	}
}

// maxChordBow is a curve's largest sampled deviation from the straight chord between its endpoints.
func maxChordBow(c geom.Curve3) float64 {
	lo, hi := c.Domain()
	p0, p1 := c.PointAt(lo), c.PointAt(hi)
	chord := p0.VectorTo(p1)
	span := float64(chord.Length())
	worst := 0.0
	for i := 0; i <= 32; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/32)
		t := float64(p0.VectorTo(p).Dot(chord)) / (span * span)
		foot := p0.TranslateBy(chord.Scale(math.Scalar(t)))
		worst = stdmath.Max(worst, float64(p.DistanceTo(foot)))
	}
	return worst
}

// assertOnSurface checks the curve's sampled points lie on surf within tol.
func assertOnSurface(t *testing.T, tag string, surf geom.Surface, c geom.Curve3, tol float64) {
	t.Helper()
	lo, hi := c.Domain()
	for i := 0; i <= 16; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/16)
		_, _, foot := geom.ClosestPointOnSurface(surf, p)
		if d := float64(p.DistanceTo(foot)); d > tol {
			t.Errorf("%s: sample %d is %.2e off the host surface (tol %.0e)", tag, i, d, tol)
		}
	}
}

// TestN4BallPathRollsOnPlaneAndTorusArm proves the derived ball-centre curve is what the canal reading
// says it is: every station centre sits exactly r from the vertical plane and exactly 2r from the lateral
// torus arm's spine circle (the ball rolling on that tube), and both host feet land at ball distance r.
func TestN4BallPathRollsOnPlaneAndTorusArm(t *testing.T) {
	arms, _ := classifyN4MixedArms(n4TestArms(t))
	torus := arms.torus.armSurface.(geom.Torus)
	vplane := arms.band.a.Geometry().(geom.Plane)
	pts, ok := n4CornerPoints(
		mustCylinder(t, math.P3(115.84559306791, 81.115957534528, 0), math.V3(0, 0, 1), 20), torus,
		arms.ccyl.armSurface.(geom.Cylinder), arms.band.armSurface.(geom.Cylinder),
		vplane, arms.torus.a.Geometry().(geom.Plane), 5)
	if !ok {
		t.Fatal("n4CornerPoints declined")
	}
	path, ok := n4CornerBallPath(torus, vplane, pts.ballBand, pts.ballCcyl, 1e-9)
	if !ok {
		t.Fatal("n4CornerBallPath declined the N4 corner")
	}
	for j, m := range path.centers {
		_, _, foot := geom.ClosestPointOnSurface(vplane, m)
		if d := stdmath.Abs(float64(foot.DistanceTo(m)) - 5); d > 1e-9 {
			t.Fatalf("station %d centre is %.2e off distance r=5 from the vplane", j, d)
		}
		if d := stdmath.Abs(spineCircleDistance(torus, m) - 10); d > 1e-9 {
			t.Fatalf("station %d centre is %.2e off distance 2r=10 from the torus arm's spine circle", j, d)
		}
		if d := stdmath.Abs(float64(path.feetTorus[j].DistanceTo(m)) - 5); d > 1e-9 {
			t.Fatalf("station %d torus foot is %.2e off ball radius 5", j, d)
		}
	}
}

// spineCircleDistance is the distance from p to a torus's spine circle (its tube axis).
func spineCircleDistance(tor geom.Torus, p math.Point3) float64 {
	k := tor.AxisDir.AsVector()
	rel := tor.Center.VectorTo(p)
	h := float64(rel.Dot(k))
	rho := float64(rel.Sub(k.Scale(math.Scalar(h))).Length())
	return stdmath.Hypot(rho-tor.MajorRadius, h)
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
