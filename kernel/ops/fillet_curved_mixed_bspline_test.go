// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
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

// n4TestCornerPts solves the N4 corner's four points, two terminating-arm arcs and two arm ball centres
// from the exact arm/host surfaces n4TestArms carries (boss = the r=20 wall both curved arms share). Shared
// by every N4 unit test that needs the corner's own derived INPUTS rather than the assembled patch.
func n4TestCornerPts(t *testing.T, arms n4MixedArms) n4CornerPts {
	t.Helper()
	pts, ok := n4CornerPoints(
		mustCylinder(t, math.P3(115.84559306791, 81.115957534528, 0), math.V3(0, 0, 1), 20),
		arms.torus.armSurface.(geom.Torus), arms.ccyl.armSurface.(geom.Cylinder),
		arms.band.armSurface.(geom.Cylinder), arms.band.a.Geometry().(geom.Plane),
		arms.torus.a.Geometry().(geom.Plane), 5)
	if !ok {
		t.Fatal("n4CornerPoints declined the N4 corner")
	}
	return pts
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
//
// It covers the two PINNED end stations as well as the 63 derived ones. n4CanalSurface replaces stations 0
// and N−1 with the corner points the terminating arms own (pts.ballBand / pts.ballCcyl with feet a,b / d,c),
// so those two — the only stations whose cross-sections become the patch's welded v=0 / v=1 boundaries —
// would otherwise be the two this test never reaches.
func TestN4BallPathRollsOnPlaneAndTorusArm(t *testing.T) {
	arms, _ := classifyN4MixedArms(n4TestArms(t))
	torus := arms.torus.armSurface.(geom.Torus)
	vplane := arms.band.a.Geometry().(geom.Plane)
	pts := n4TestCornerPts(t, arms)
	path, ok := n4CornerBallPath(torus, vplane, pts.ballBand, pts.ballCcyl, 1e-9)
	if !ok {
		t.Fatal("n4CornerBallPath declined the N4 corner")
	}
	for j, m := range path.centers {
		assertBallCentreRollsOnPlaneAndTube(t, fmt.Sprintf("derived station %d", j), torus, vplane, m, path.feetTorus[j])
	}
	assertBallCentreRollsOnPlaneAndTube(t, "PINNED band-arm end station", torus, vplane, pts.ballBand, pts.b)
	assertBallCentreRollsOnPlaneAndTube(t, "PINNED ccyl-arm end station", torus, vplane, pts.ballCcyl, pts.c)
	// The pinned stations' VPLANE feet are corner points A and D, the patch's two welded plane-side corners.
	for _, w := range []struct {
		tag          string
		centre, foot math.Point3
	}{{"A on the band-arm station", pts.ballBand, pts.a}, {"D on the ccyl-arm station", pts.ballCcyl, pts.d}} {
		if d := stdmath.Abs(float64(w.foot.DistanceTo(w.centre)) - 5); d > 1e-9 {
			t.Errorf("PINNED vplane foot %s is %.2e off ball radius 5", w.tag, d)
		}
	}
}

// assertBallCentreRollsOnPlaneAndTube checks one ball-centre station against the canal reading: exactly r
// from the vertical plane, exactly 2r from the lateral torus arm's spine circle (the ball riding that tube's
// outside), and its torus foot at ball radius r.
func assertBallCentreRollsOnPlaneAndTube(t *testing.T, tag string, torus geom.Torus, vplane geom.Plane, centre, footTorus math.Point3) {
	t.Helper()
	_, _, foot := geom.ClosestPointOnSurface(vplane, centre)
	if d := stdmath.Abs(float64(foot.DistanceTo(centre)) - 5); d > 1e-9 {
		t.Errorf("%s centre is %.2e off distance r=5 from the vplane", tag, d)
	}
	if d := stdmath.Abs(torusTubeMembership(torus, 10, centre)); d > 1e-9 {
		t.Errorf("%s centre is %.2e off distance 2r=10 from the torus arm's spine circle", tag, d)
	}
	if d := stdmath.Abs(float64(footTorus.DistanceTo(centre)) - 5); d > 1e-9 {
		t.Errorf("%s torus foot is %.2e off ball radius 5", tag, d)
	}
}

// n4TestCanalCert certifies the real N4 canal patch against its received 4-cycle, applying `mutate` to that
// 4-cycle first so a test can perturb what the certificate is measured AGAINST while leaving the surface
// untouched. Returns the certificate and the Resolution its thresholds are read at.
func n4TestCanalCert(t *testing.T, mutate func(*RailLoop)) (Certificate, Resolution) {
	t.Helper()
	arms, _ := classifyN4MixedArms(n4TestArms(t))
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 60)})
	corner, ok := solveN4Corner(arms, 5, res)
	if !ok {
		t.Fatal("solveN4Corner declined the N4 corner")
	}
	loop := RailLoop{Sides: []Side{
		{Curve: corner.pts.arcAB, Adjacent: arms.band.armSurface, Cont: G1},
		{Curve: corner.railBC, Adjacent: arms.torus.armSurface, Cont: G1},
		{Curve: corner.pts.arcCD, Adjacent: arms.ccyl.armSurface, Cont: G1},
		{Curve: corner.railDA, Adjacent: corner.vplane, Cont: G1},
	}}
	mutate(&loop)
	hosts := []geom.Surface{corner.vplane, arms.torus.armSurface}
	return certifyN4CanalPatch(corner.patch.Surface.(geom.BSplineSurface), loop, hosts, res), res
}

// TestN4CertificateMeasuresGeometryItDoesNotOwn falsifies certifyN4CanalPatch's G0 measure, which is the
// only way to know it is a guard rather than a claim. MaxDev used to be maxLoopSurfaceDev of the patch's OWN
// boundary isoparms — the surface measured against itself, reading ~4.4e-13 whatever the surface is — so a
// regression in the end-pinning or in LoftCanalStations' parametrisation could lift the boundary clean off
// pts.arcAB and still certify, shipping a cracked weld. It now measures the RECEIVED arm arcs against the
// surface and the foot-loci against the two HOSTS, so lifting an arm arc off the surface must REJECT.
func TestN4CertificateMeasuresGeometryItDoesNotOwn(t *testing.T) {
	base, res := n4TestCanalCert(t, func(*RailLoop) {})
	if !base.Valid(res) {
		t.Fatalf("the unperturbed N4 canal patch does not certify: %+v (weld %.3e)", base, res.Weld())
	}
	// The self-referential floor is ~4.4e-13; the informative residuals are the foot-loci's ~1.8e-8. A MaxDev
	// down at the floor means the certificate is reading the surface against itself again.
	if base.MaxDev < 1e-11 {
		t.Fatalf("MaxDev %.3e sits at the self-referential floor (~4.4e-13): the certificate is measuring the "+
			"patch against its own boundary isoparms and so certifies nothing", base.MaxDev)
	}
	for _, lift := range []float64{1e-3, 1e-6} {
		lifted, res := n4TestCanalCert(t, func(l *RailLoop) { l.Sides[0].Curve = bulgeArcOffSurface(t, l.Sides[0].Curve, lift) })
		if lifted.MaxDev < lift/2 {
			t.Fatalf("lifting the received band arc %.0e off the surface left MaxDev at %.3e — the G0 measure "+
				"does not see the boundary come off the arc the arm face trims to", lift, lifted.MaxDev)
		}
		// MaxDev must be the SOLE catcher, or this proves nothing about MaxDev: the mutation keeps the arc's
		// endpoints, so the 4-cycle still closes, and a mid-span radial bulge barely tilts the tangent plane.
		if !lifted.Closed || lifted.MaxAngleDev > seamAngularTol {
			t.Fatalf("the %.0e bulge also tripped Closed=%v / MaxAngleDev=%.3e, so it does not isolate the G0 "+
				"axis — pick a mutation only MaxDev can see", lift, lifted.Closed, lifted.MaxAngleDev)
		}
		if lifted.Valid(res) {
			t.Fatalf("a patch whose boundary is %.0e off the received band arc still certifies "+
				"(MaxDev %.3e vs weld %.3e) — the corner would ship a cracked weld", lift, lifted.MaxDev, res.Weld())
		}
	}
}

// bulgeArcOffSurface returns an arc through the SAME two endpoints whose midpoint is pushed d radially
// outward — so it diverges from the canal's end cross-section by d mid-span while still meeting it at both
// corner points. That endpoint preservation is the point: it leaves Closed and MaxAngleDev untouched, so a
// rejection can only have come from the G0 measure under test.
//
// Two more obvious mutations are traps. Translating the arc rigidly along its own normal slides it ALONG the
// canal (that normal is the spine direction, i.e. the surface's own v-tangent) and measures ~0. Offsetting
// the RADIUS moves the endpoints too, which breaks Closed and lets the certificate reject for a reason that
// has nothing to do with MaxDev.
func bulgeArcOffSurface(t *testing.T, c geom.Curve3, d float64) geom.Curve3 {
	t.Helper()
	arc, ok := c.(geom.Arc3d)
	if !ok {
		t.Fatalf("side curve is %T, want geom.Arc3d (the terminating-arm cross-section)", c)
	}
	lo, hi := arc.Domain()
	mid := arc.PointAt((lo + hi) / 2)
	out := arc.Center.VectorTo(mid)
	bulged, err := geom.Arc3dByThreePoints(arc.PointAt(lo),
		mid.TranslateBy(out.Scale(math.Scalar(d/float64(out.Length())))), arc.PointAt(hi))
	if err != nil {
		t.Fatalf("bulging the band arc by %g gave no arc through the three points: %v", d, err)
	}
	return bulged
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
