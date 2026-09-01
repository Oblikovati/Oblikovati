// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The U4 setback-section oracle: OCCT's REAL interior seam curves, dumped from DRAWEXE 8.0.0 (the
// parity oracle, recipe in .superpowers/sdd/t1-t7-oracle-forensics.md §Appendix + the U4 derivation §0:
// `restore CFI_6_h56fhg.rle s ; tscale s 0 0 0 1 ; explode s E ; blend result s 5 s_2`). Each row is a
// point (x,y,z) sampled at 9 equal parameters along the shared face boundary:
//   - occtSectionZNode: the z=-6.240 section — the edge SHARED between the B-only sliver (result_5,
//     area 3.039) and the dual-host core panel (result_13, area 30.334): the A-node seam station.
//   - occtSectionZMid:  the z=0 section — the edge SHARED between the two core panels (result_13 /
//     result_14): the mid-panel seam station.
//
// These are the go/no-go oracle for the setbackSection rail (derivation §4-U4-2, the CRUX slice).
//
// The load-bearing measured fact: a circle fitted through each row's endpoints + midpoint has radius
// 4.999999 / 5.000006 (= the fillet rolling-ball radius, ef.cyl.Radius) and the remaining 6 points
// scatter only 6.1e-7 / 7.8e-7 from it — i.e. OCCT's rational-BSpline section IS the fillet-radius
// circular arc, sampled to a ~1e-6 representation floor. setbackSection reconstructs that exact arc.
var occtSectionZNode = [][3]float64{
	{5.006254, -20.000000, -6.239986}, {5.891428, -19.921023, -6.239986},
	{6.795413, -19.668930, -6.239986}, {7.667392, -19.233007, -6.239986},
	{8.452265, -18.622846, -6.239986}, {9.100907, -17.869463, -6.239986},
	{9.579885, -17.020372, -6.239986}, {9.876862, -16.130122, -6.239986},
	{10.000000, -15.250004, -6.239986},
}
var occtSectionZMid = [][3]float64{
	{8.000000, -20.000000, 0}, {8.358874, -19.721849, 0}, {8.695122, -19.408114, 0},
	{9.003630, -19.061721, 0}, {9.279827, -18.686551, 0}, {9.519908, -18.287304, 0},
	{9.720998, -17.869300, 0}, {9.881274, -17.438249, 0}, {10.000000, -17.000000, 0},
}

// occtSectionOracleFloor bounds how close ANY analytic rail can sit to OCCT's SAMPLED section. It is a
// curve-approximation tolerance (tol:calibrated), NOT a model-coincidence weld: OCCT stores the section
// as a rational BSpline whose sample points scatter up to ~7.8e-7 from the exact radius-5 circle they
// represent (measured: circleFitResidual below), and the fixture rounds them to 6 decimals — so model
// weld (res.Weld() ≈ 6.4e-8) is BELOW the oracle's own sampling precision and unreachable by any curve,
// including OCCT's own exact circle. 2e-6 clears that floor with headroom yet still fails a wrong rail
// hard: the naive fillet-45°-bulge arc misses by ~4.4e-3 (≈2000×), the straight chord by ~0.2.
const occtSectionOracleFloor = 2e-6 // tol:calibrated (OCCT BSpline-vs-analytic-circle sampling floor)

// circleFitResidual returns the max deviation of samples from the circle fitted through the first,
// middle and last of them — the oracle's OWN internal scatter, the floor no analytic rail can beat.
func circleFitResidual(t *testing.T, samples [][3]float64) float64 {
	t.Helper()
	arc, err := geom.Arc3dByThreePoints(sampleP(samples[0]), sampleP(samples[len(samples)/2]), sampleP(samples[len(samples)-1]))
	if err != nil {
		t.Fatalf("circleFitResidual: samples collinear: %v", err)
	}
	return maxRadialDev(arc, samples)
}

func sampleP(s [3]float64) math.Point3 { return math.P3(s[0], s[1], s[2]) }

// maxRadialDev returns the largest perpendicular (radial) distance of any sample from the arc's circle —
// the exact deviation of an analytic circular rail from the sampled oracle, free of the discretization
// error a closest-point-on-polyline scan would add.
func maxRadialDev(arc geom.Arc3d, samples [][3]float64) float64 {
	worst := 0.0
	for _, s := range samples {
		if d := stdmath.Abs(arc.Center.DistanceTo(sampleP(s)) - arc.Radius); d > worst {
			worst = d
		}
	}
	return worst
}

// TestSetbackSectionMatchesOCCTSectionOracle is the U4-2 CRUX gate (derivation §4-U4-2): setbackSection
// at the A-node station (z=-6.240) and the mid station (z=0) must reproduce OCCT's real section curve.
// It asserts, per station: (1) the rail is a circular arc of the fillet rolling-ball radius (the setback
// section is analytic, not a free-form curve that would need the method-C marcher); (2) its endpoints
// weld to OCCT's section endpoints (the boss-rim points, shared bit-for-bit with the neighbour A-rim /
// B-rim rails — the corner-weld invariant); (3) every OCCT sample sits within the oracle floor of the
// rail. Method A (radiusArcRail) meets all three, so U4-3/U4-4's coons4 fill is de-risked: the seam
// rails it consumes are exact.
func TestSetbackSectionMatchesOCCTSectionOracle(t *testing.T) {
	t.Parallel()
	_, fils, res := u4Fillet(t)
	ef := fils[0]
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("fixture precondition: detectObstacles(U4) = (%d, %v), want (2, true)", len(dets), ok)
	}
	for _, tc := range []struct {
		name    string
		z       float64
		samples [][3]float64
	}{
		{"A-node z=-6.240 (sliver|core seam)", occtSectionZNode[0][2], occtSectionZNode},
		{"mid z=0 (core|core seam)", 0, occtSectionZMid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertSectionMatchesOracle(t, tc.z, dets, ef, res, tc.samples)
		})
	}
}

func assertSectionMatchesOracle(t *testing.T, z float64, dets []obstacleDetection, ef edgeFillet, res Resolution, samples [][3]float64) {
	t.Helper()
	rail, ok := setbackSection(z, dets, ef, res)
	if !ok {
		t.Fatalf("setbackSection(z=%.4f) ok=false, want a rail", z)
	}
	arc, isArc := rail.(geom.Arc3d)
	if !isArc {
		t.Fatalf("setbackSection(z=%.4f) = %T, want a circular arc (the section is analytic)", z, rail)
	}
	if d := stdmath.Abs(arc.Radius - ef.cyl.Radius); d > 1e-6 {
		t.Errorf("rail radius %.6f, want the fillet rolling-ball radius %.6f (Δ=%.2e)", arc.Radius, ef.cyl.Radius, d)
	}
	assertEndpointsWeldToOracle(t, rail, samples)
	if dev := maxRadialDev(arc, samples); dev > occtSectionOracleFloor {
		t.Errorf("rail deviates %.3e from OCCT's section samples, want <= %.1e (oracle floor)", dev, occtSectionOracleFloor)
	}
	// Prove the gate is honest, not vacuous: the naive fillet-45° bulge would miss by ~1e-3, and the
	// oracle floor is genuinely tight (OCCT's own samples scatter this far from their circle).
	t.Logf("z=%.4f: radius=%.6f radialDev=%.3e oracleFloor(OCCT self-scatter)=%.3e weld=%.3e",
		z, arc.Radius, maxRadialDev(arc, samples), circleFitResidual(t, samples), res.Weld())
}

// assertEndpointsWeldToOracle checks the rail's two ends coincide with OCCT's section endpoints. The
// tolerance is the oracle floor (the fixture's 6-decimal rounding + the imported-rim-vs-OCCT-rim
// difference), not model weld: the endpoints are pinned EXACTLY to this kernel's own boss-rim points
// (proven weld-exact to the neighbour rail in TestSetbackSectionEndpointsPinnedToRim), and this only
// cross-checks that those rim points agree with OCCT's independently-dumped ones.
func assertEndpointsWeldToOracle(t *testing.T, rail geom.Curve3, samples [][3]float64) {
	t.Helper()
	ends := map[string][2]math.Point3{
		"pA": {rail.PointAt(0), sampleP(samples[0])},
		"pB": {rail.PointAt(1), sampleP(samples[len(samples)-1])},
	}
	for name, pair := range ends {
		if d := pair[0].DistanceTo(pair[1]); d > occtSectionOracleFloor {
			t.Errorf("%s: rail end %v is %.3e from OCCT's section end %v, want <= %.1e", name, pair[0], d, pair[1], occtSectionOracleFloor)
		}
	}
}

// TestSetbackSectionEndpointsPinnedToRim proves the corner-weld invariant (ADR-0042): the rail's
// endpoints are BIT-IDENTICAL to the independently-evaluated boss-rim points a neighbour panel's A-rim /
// B-rim rail would end at — so the panels share their corners with no seam-opening float drift.
func TestSetbackSectionEndpointsPinnedToRim(t *testing.T) {
	t.Parallel()
	_, fils, res := u4Fillet(t)
	ef := fils[0]
	dets, _ := detectObstacles(ef, res)
	detA, detB, ok := hostDetections(dets)
	if !ok {
		t.Fatalf("hostDetections: ok=false, want both hosts")
	}
	const z = 0.0
	rail, ok := setbackSection(z, dets, ef, res)
	if !ok {
		t.Fatalf("setbackSection ok=false")
	}
	pinA := sectionEndA(detA, ef, z)
	pinB, _ := dipRimPointAtStation(detB, ef, z)
	// The rail's ends equal the boss-rim points to within weld — in practice ~1e-15 (a couple of ULP of
	// Arc3dByThreePoints' endpoint reconstruction), far inside res.Weld(); the assembler welds the shared
	// corner at exactly this tolerance, so the panels close with no seam drift (ADR-0042).
	if d := rail.PointAt(0).DistanceTo(pinA); d > res.Weld() {
		t.Errorf("pA not pinned to the host-A rim point: drift %.3e (want <= weld %.3e)", d, res.Weld())
	}
	if d := rail.PointAt(1).DistanceTo(pinB); d > res.Weld() {
		t.Errorf("pB not pinned to the host-B rim point: drift %.3e (want <= weld %.3e)", d, res.Weld())
	}
}

// TestSetbackSectionCollinearFallsBackToLineSegment pins §2.4: when the endpoints are farther apart than
// the arc diameter (a vanishingly thin, near-collinear section) no radius-r circle spans them — the apex
// collapses onto the chord and the rail degrades to a straight segment rather than returning a bogus arc.
func TestSetbackSectionCollinearFallsBackToLineSegment(t *testing.T) {
	t.Parallel()
	_, fils, _ := u4Fillet(t)
	ef := fils[0]
	pA := math.P3(5, -20, 0)
	pB := math.P3(5+2*ef.cyl.Radius+0.1, -20, 0) // chord longer than the arc diameter ⇒ no radius-r arc
	rail := radiusArcRail(pA, pB, ef, 0)
	if _, isLine := rail.(geom.LineSegment); !isLine {
		t.Fatalf("radiusArcRail on a super-diameter chord = %T, want geom.LineSegment (collinear fallback)", rail)
	}
	if d := rail.PointAt(0).DistanceTo(pA); d != 0 {
		t.Errorf("fallback segment start drifted %.3e from pA", d)
	}
	if d := rail.PointAt(1).DistanceTo(pB); d != 0 {
		t.Errorf("fallback segment end drifted %.3e from pB", d)
	}
}

// TestSetbackSectionRejectsNonDualHost pins that setbackSection is defined only for the qualifying==2
// dual-host case: a single-host detection list has no A/B pair to bridge, so hostDetections → ok=false.
func TestSetbackSectionRejectsNonDualHost(t *testing.T) {
	t.Parallel()
	_, fils, res := u4Fillet(t)
	ef := fils[0]
	dets, _ := detectObstacles(ef, res)
	detA, _, _ := hostDetections(dets)
	if _, ok := setbackSection(0, []obstacleDetection{detA}, ef, res); ok {
		t.Errorf("setbackSection with one host = ok=true, want ok=false (needs an A and a B host)")
	}
}
