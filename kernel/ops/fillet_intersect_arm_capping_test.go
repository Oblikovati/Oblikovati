// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR2 — the intersectArmCapping port, oracle-pinned to DRAWEXE (far-runout-port-math.md). The workhorse is
// Torus∩Plane on D5's meridian arm: the closed-form feet and the spiric trim match OCCT's blend far edge
// to its own approximation floor, the trim is analytic-on-the-arm to machine eps, and the branch selection
// rejects the mirror oval. Cyl∩Plane is pinned to OCCT's exact ellipse. Honest-reject on invalid feet.

// d5CapZ is the D5 bottom-cap plane level z = −75√3 (latitude −60° of the ×10 sphere), the level every
// DRAWEXE meridian far-edge pole sits on (d5samples.txt, all 21 poles z=−129.90381056766577).
const d5CapZ = -129.90381056766577

// d5MeridianFarEdge is DRAWEXE's dump of D5's meridian arm far edge (edge result_3_1), 21 samples t=0..1
// (session scratchpad farrunout/d5samples.txt). t=0 is the sphere-spring foot, t=1 the plane-spring foot;
// z is d5CapZ throughout. It is the oracle the spiric trim is pinned against.
var d5MeridianFarEdge = [21][2]float64{
	{-74.230748813988725, 10.714285713525337}, {-74.282113871908933, 9.9677561625650259},
	{-74.217469085913237, 9.1990087884143499}, {-74.026977723449392, 8.4142110790065487},
	{-73.701926399784639, 7.6205798011617372}, {-73.235140343362104, 6.826254314688553},
	{-72.621381293988804, 6.0400980549783698}, {-71.857697476006948, 5.2714322555319635},
	{-70.94369110810085, 4.5297115780106676}, {-69.88165204778295, 3.8241500780036337},
	{-68.676582002648956, 3.163362976341725}, {-67.336056789651693, 2.5550139855509015},
	{-65.869968454508125, 2.00552532925856}, {-64.290163918722129, 1.5198632748523859},
	{-62.609986081252508, 1.1014070197033656}, {-60.843760105154175, 0.75191306186925966},
	{-59.006322605499463, 0.47158001322574306}, {-57.112561083615915, 0.25917936633205352},
	{-55.177001590486526, 0.1122408633770115}, {-53.213463879739734, 0.027273099846632141},
	{-51.234792870056268, 1.1102230246251565e-14},
}

// d5MeridianArm reconstructs D5's meridian torus arm and its hosts closed-form (far-runout-port-math §1/§5,
// verified against the DRAWEXE frame): host sphere R=150 at the origin; host longitude plane (the y=0
// meridian plane, normal ŷ); the fillet torus (centre (0,10,0), axis ŷ, R′=√19500, r=10, ref x̂ so the
// section lands in the DRAWEXE frame); and the bottom-cap capping plane z=d5CapZ (normal ẑ).
func d5MeridianArm(t *testing.T) (geom.Torus, geom.Sphere, geom.Plane, geom.Plane) {
	t.Helper()
	tor, err := geom.NewTorusWithRef(math.P3(0, 10, 0), math.V3(0, 1, 0), math.V3(1, 0, 0), stdmath.Sqrt(19500), 10)
	if err != nil {
		t.Fatalf("D5 meridian torus: %v", err)
	}
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 150)
	if err != nil {
		t.Fatalf("D5 host sphere: %v", err)
	}
	return tor, sphere, planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0)), planeOn(t, math.P3(0, 0, d5CapZ), math.V3(0, 0, 1))
}

// d5Feet computes the two runout feet closed-form via armSprings + springCapFoot, ordered [sphere, plane]
// like the DRAWEXE far edge. `near` is a material-side (−x) reference so springCapFoot keeps the −x roots.
func d5Feet(t *testing.T, tor geom.Torus, sphere geom.Sphere, lonPlane, cap geom.Plane) [2]math.Point3 {
	t.Helper()
	springs, ok := armSprings(tor, sphere, lonPlane, 10)
	if !ok {
		t.Fatal("armSprings declined the D5 sphere∧plane torus arm")
	}
	near := math.P3(-(tor.MajorRadius + tor.MinorRadius), 10, d5CapZ)
	res := ResolutionForSize(300)
	f0, ok0 := springCapFoot(springs[0], cap, near, res)
	f1, ok1 := springCapFoot(springs[1], cap, near, res)
	if !ok0 || !ok1 {
		t.Fatalf("springCapFoot declined: sphere=%v plane=%v", ok0, ok1)
	}
	return [2]math.Point3{f0, f1}
}

// signedDistTorus is the exact signed distance of p to the torus surface (|dist-to-tube-axis-circle − r|),
// used to certify the trim is ON the arm (analytic-on-arm) and to measure the oracle's own on-torus floor.
func signedDistTorus(tr geom.Torus, p math.Point3) float64 {
	axis := tr.AxisDir.AsVector()
	q := tr.Center.VectorTo(p)
	za := float64(q.Dot(axis))
	radial := float64(q.Sub(axis.Scale(math.Scalar(za))).Length())
	return stdmath.Abs(stdmath.Hypot(radial-tr.MajorRadius, za) - tr.MinorRadius)
}

// TestIntersectArmCapping_D5MeridianSpiric is the FR2 gate. The closed-form feet match DRAWEXE (sphere
// 8.2e-8, plane 3.9e-5 = OCCT's blend edge tol); the spiric trim is analytic-on-the-arm (on the exact
// torus AND the exact plane to machine eps) and matches DRAWEXE's meridian far edge to OCCT's own
// approximation floor.
func TestIntersectArmCapping_D5MeridianSpiric(t *testing.T) {
	tor, sphere, lonPlane, cap := d5MeridianArm(t)
	feet := d5Feet(t, tor, sphere, lonPlane, cap)
	assertFeetMatchOracle(t, feet)

	curve, ok := intersectArmCapping(tor, cap, feet, 10, ResolutionForSize(300))
	if !ok || curve == nil {
		t.Fatal("intersectArmCapping declined D5's Torus∩Plane runout")
	}
	assertAnalyticOnArm(t, tor, curve)
	assertMatchesOracleFarEdge(t, tor, curve)
}

// assertFeetMatchOracle pins the closed-form feet to the DRAWEXE far-edge endpoints.
func assertFeetMatchOracle(t *testing.T, feet [2]math.Point3) {
	t.Helper()
	sphereOracle := math.P3(math.Scalar(d5MeridianFarEdge[0][0]), math.Scalar(d5MeridianFarEdge[0][1]), d5CapZ)
	planeOracle := math.P3(math.Scalar(d5MeridianFarEdge[20][0]), math.Scalar(d5MeridianFarEdge[20][1]), d5CapZ)
	ds := float64(feet[0].DistanceTo(sphereOracle))
	dp := float64(feet[1].DistanceTo(planeOracle))
	t.Logf("feet vs DRAWEXE: sphere-spring %.3e, plane-spring %.3e (OCCT blend edge tol 1e-4)", ds, dp)
	if ds > 1e-6 {
		t.Fatalf("sphere-spring foot %v off oracle %v by %.3e (>1e-6)", feet[0], sphereOracle, ds)
	}
	if dp > 1e-4 {
		t.Fatalf("plane-spring foot %v off oracle %v by %.3e (>1e-4 OCCT blend tol)", feet[1], planeOracle, dp)
	}
}

// assertAnalyticOnArm certifies every sampled trim point is on the exact torus AND the exact cap plane to
// machine eps — the shared-edge-identity guarantee (the trim is the analytic spiric, never a polyline).
func assertAnalyticOnArm(t *testing.T, tor geom.Torus, curve geom.Curve3) {
	t.Helper()
	worstTor, worstPl := 0.0, 0.0
	for k := 0; k <= 20; k++ {
		p := curve.PointAt(float64(k) / 20)
		worstTor = stdmath.Max(worstTor, signedDistTorus(tor, p))
		worstPl = stdmath.Max(worstPl, stdmath.Abs(float64(p.Z)-d5CapZ))
	}
	t.Logf("analytic-on-arm: worst off-torus %.3e, worst off-plane %.3e", worstTor, worstPl)
	if worstTor > 1e-9 || worstPl > 1e-9 {
		t.Fatalf("trim not analytic-on-arm: off-torus %.3e off-plane %.3e (want machine eps)", worstTor, worstPl)
	}
}

// assertMatchesOracleFarEdge measures the closest-point distance from every DRAWEXE far-edge sample to the
// trim curve. The exact spiric must lie within OCCT's own approximation floor of each sample; the oracle
// samples' own max deviation from the exact torus (measured here) is that floor (~4.7e-7).
func assertMatchesOracleFarEdge(t *testing.T, tor geom.Torus, curve geom.Curve3) {
	t.Helper()
	oracleFloor, worst := 0.0, 0.0
	for _, s := range d5MeridianFarEdge {
		p := math.P3(math.Scalar(s[0]), math.Scalar(s[1]), d5CapZ)
		oracleFloor = stdmath.Max(oracleFloor, signedDistTorus(tor, p))
		worst = stdmath.Max(worst, closestOnCurve(curve, p))
	}
	t.Logf("trim vs DRAWEXE far edge: worst closest-point %.3e (OCCT on-torus floor %.3e)", worst, oracleFloor)
	if worst > 6.83e-7 { // far-runout-port-math §1: OCCT's blend approximation error, not our trim's
		t.Fatalf("spiric trim off DRAWEXE far edge by %.3e (>6.83e-7 OCCT floor)", worst)
	}
}

// closestOnCurve returns the minimum distance from p to the curve — a coarse scan then a ternary refine
// (a test-only probe; the curve itself is analytic, this only compares against the approximate oracle).
func closestOnCurve(curve geom.Curve3, p math.Point3) float64 {
	step, bestT := 1.0/2000, 0.0
	best := stdmath.Inf(1)
	for i := 0; i <= 2000; i++ {
		if d := float64(curve.PointAt(float64(i) * step).DistanceTo(p)); d < best {
			best, bestT = d, float64(i)*step
		}
	}
	lo, hi := stdmath.Max(0, bestT-step), stdmath.Min(1, bestT+step)
	for k := 0; k < 100; k++ {
		m1, m2 := lo+(hi-lo)/3, hi-(hi-lo)/3
		if curve.PointAt(m1).DistanceTo(p) < curve.PointAt(m2).DistanceTo(p) {
			hi = m2
		} else {
			lo = m1
		}
	}
	return float64(curve.PointAt((lo + hi) / 2).DistanceTo(p))
}

// TestIntersectArmCapping_D5BranchMutation is the mirror-oval kill (far-runout-port-math §6.1). The port's
// selected spiric branch hits BOTH feet; the OPPOSITE branch (the mirror oval) misses them by ~1e2 — 8+
// orders larger — so the endpoint certificate can never return it. Forcing the mirror is caught here.
func TestIntersectArmCapping_D5BranchMutation(t *testing.T) {
	tor, sphere, lonPlane, cap := d5MeridianArm(t)
	feet := d5Feet(t, tor, sphere, lonPlane, cap)
	curve, ok := intersectArmCapping(tor, cap, feet, 10, ResolutionForSize(300))
	if !ok {
		t.Fatal("precondition: the port must build the D5 trim")
	}
	sa, isSpiric := curve.(geom.SpiricArc)
	if !isSpiric {
		t.Fatalf("D5 trim is %T, want geom.SpiricArc", curve)
	}
	good0 := float64(sa.PointAt(0).DistanceTo(feet[0]))
	good1 := float64(sa.PointAt(1).DistanceTo(feet[1]))
	mirror := sa
	mirror.Branch = -sa.Branch
	bad0 := float64(mirror.PointAt(0).DistanceTo(feet[0]))
	bad1 := float64(mirror.PointAt(1).DistanceTo(feet[1]))
	t.Logf("branch certificate: selected feet gaps (%.3e, %.3e) ; mirror oval gaps (%.3e, %.3e)", good0, good1, bad0, bad1)
	if good0 > 1e-6 || good1 > 1e-6 {
		t.Fatalf("selected branch %v misses the feet (%.3e, %.3e)", sa.Branch, good0, good1)
	}
	if bad0 < 1 && bad1 < 1 {
		t.Fatalf("mirror oval (branch %v) is not separated from the feet (%.3e, %.3e) — certificate would not reject it", mirror.Branch, bad0, bad1)
	}
}

// TestCylinderPlaneTrim_Ellipse pins the oblique Cyl∩Plane section to OCCT's exact ellipse
// (far-runout-port-math §3: cylinder r=15 axis ẑ ∩ plane 3 2 4 / 0.5 0.2 0.8 → Center (0,0,6.375), radii
// 18.0818451768618 / 15, major dir (−0.77023,−0.30809,0.55842)), then trims between two on-ellipse feet
// and certifies the trim is analytic-on-the-arm (on the exact cylinder AND the exact plane).
func TestCylinderPlaneTrim_Ellipse(t *testing.T) {
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 15)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	pl := planeOn(t, math.P3(3, 2, 4), math.V3(0.5, 0.2, 0.8))
	res := ResolutionForSize(60)
	curves, ok := geom.IntersectSurfacesAnalytic(pl, cyl, res)
	if !ok || len(curves) != 1 {
		t.Fatalf("plane∩cylinder: handled=%v n=%d, want 1 ellipse", ok, len(curves))
	}
	ell := curves[0].(geom.EllipseFull)
	assertEllipseMatchesOCCT(t, ell)

	feet := [2]math.Point3{ell.PointAt(0.12), ell.PointAt(0.4)}
	trim, ok := intersectArmCapping(cyl, pl, feet, 15, res)
	if !ok || trim == nil {
		t.Fatal("intersectArmCapping declined the oblique Cyl∩Plane runout")
	}
	assertOnCylinderAndPlane(t, cyl, pl, trim)
}

// assertEllipseMatchesOCCT checks the section ellipse against OCCT's `intersect` dump (§3).
func assertEllipseMatchesOCCT(t *testing.T, ell geom.EllipseFull) {
	t.Helper()
	if d := float64(ell.Center.DistanceTo(math.P3(0, 0, 6.375))); d > 1e-9 {
		t.Fatalf("ellipse centre %v off OCCT (0,0,6.375) by %.3e", ell.Center, d)
	}
	if stdmath.Abs(ell.MajorRadius-18.0818451768618) > 1e-9 || stdmath.Abs(ell.MinorRadius-15) > 1e-9 {
		t.Fatalf("ellipse radii (%.10f, %.10f), want OCCT (18.0818451768618, 15)", ell.MajorRadius, ell.MinorRadius)
	}
	majorDot := stdmath.Abs(float64(ell.MajorAxis.AsVector().Dot(math.V3(-0.77023, -0.30809, 0.55842))))
	if majorDot < 1-1e-4 {
		t.Fatalf("ellipse major axis %v not aligned with OCCT XAxis (|dot|=%.6f)", ell.MajorAxis, majorDot)
	}
}

// assertOnCylinderAndPlane certifies every sampled trim point is on the exact cylinder and cap plane.
func assertOnCylinderAndPlane(t *testing.T, cyl geom.Cylinder, pl geom.Plane, trim geom.Curve3) {
	t.Helper()
	n := pl.Normal()
	worstCyl, worstPl := 0.0, 0.0
	for k := 0; k <= 20; k++ {
		p := trim.PointAt(float64(k) / 20)
		q := cyl.Origin.VectorTo(p)
		axial := q.Dot(cyl.AxisDir.AsVector())
		radial := float64(q.Sub(cyl.AxisDir.AsVector().Scale(axial)).Length())
		worstCyl = stdmath.Max(worstCyl, stdmath.Abs(radial-cyl.Radius))
		worstPl = stdmath.Max(worstPl, stdmath.Abs(float64(n.Dot(pl.Origin.VectorTo(p)))))
	}
	t.Logf("cyl∩plane trim analytic-on-arm: worst off-cylinder %.3e, off-plane %.3e", worstCyl, worstPl)
	if worstCyl > 1e-9 || worstPl > 1e-9 {
		t.Fatalf("cyl∩plane trim not analytic-on-arm: off-cyl %.3e off-plane %.3e", worstCyl, worstPl)
	}
}

// TestIntersectArmCapping_HonestReject: invalid inputs floor to (nil,false), never a garbage curve — a foot
// off the arm (§0 inversion certificate) and a cap plane ⊥ the torus axis (the M→0 latitude-circle
// follow-on) both decline.
func TestIntersectArmCapping_HonestReject(t *testing.T) {
	tor, _, _, cap := d5MeridianArm(t)
	res := ResolutionForSize(300)
	offArm := [2]math.Point3{math.P3(1000, 0, 0), math.P3(0, 0, d5CapZ)}
	if c, ok := intersectArmCapping(tor, cap, offArm, 10, res); ok || c != nil {
		t.Fatalf("a foot off the arm must decline; got (%v,%v)", c, ok)
	}
	axisCap := planeOn(t, math.P3(0, 20, 0), math.V3(0, 1, 0)) // ⊥ the torus axis ŷ → M→0
	feet := [2]math.Point3{tor.PointAt(0.3, 0.3), tor.PointAt(0.3, -0.3)}
	if c, ok := intersectArmCapping(tor, axisCap, feet, 10, res); ok || c != nil {
		t.Fatalf("a cap plane ⊥ the torus axis (M→0) must decline (latitude-circle follow-on); got (%v,%v)", c, ok)
	}
}
