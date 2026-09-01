// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The mid-span obstacle patch used to be a single Coons FillSurface through four right rails. A Coons
// interpolant enforces the boundary and lets the interior go where the rail blend puts it; the true
// surface is the SURF-RST rolling-ball envelope — the ball tangent to the fillet WALL, passing THROUGH
// the obstacle rim (fillet_obstacle_canal.go). These are the hermetic, oracle-free gates on that swap.
//
// WHAT THEY MEASURE. maxBallDev (corner_ball_envelope.go) on a 12x12 strictly-interior parameter grid:
// for each sample it re-solves the ball centre in the sample's own section plane from the sample plus the
// declared tangency host, and reports how far the declared restriction curve then sits from being at
// radius. It is the only certificate field that says anything about a patch's INTERIOR, and it is exactly
// the condition a Coons fill through these rails cannot pass.

// t6ObstacleEnvelopeGeom is the synthetic T6 obstacle's own hosts and rim, in closed form: the fillet is
// r=6 on the box edge y=-13 ^ z=0 (so its plain ball-centre axis is y=-7, z=-6, spine +X), the notched
// HOST is z=0 carrying the base ellipse a=15/b=10, and the WALL is y=-13.
func t6ObstacleEnvelopeGeom(t *testing.T) (cyl geom.Cylinder, wall, host geom.Plane, rim geom.EllipseFull) {
	t.Helper()
	cyl, errC := geom.NewCylinder(math.P3(0, -7, -6), math.V3(1, 0, 0), 6)
	wall, errW := geom.NewPlane(math.P3(0, -13, 0), math.V3(0, 1, 0))
	host, errH := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	e, errE := geom.NewEllipseFull(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 15, 10)
	for _, err := range []error{errC, errW, errH, errE} {
		if err != nil {
			t.Fatalf("t6 obstacle envelope geometry: %v", err)
		}
	}
	return cyl, wall, host, e
}

// t6ObstacleCanal solves the surf-rst station triple for the synthetic T6 obstacle at `stations` rim feet
// spread over the dip, exactly as buildObstacleCanal does in the pipeline (obstacleCanalStations is the
// shared solver), and returns it with the envelope attached.
func t6ObstacleCanal(t *testing.T, of *ObstacleFeature, stations int) *obstacleCanal {
	t.Helper()
	cyl, wall, host, rim := t6ObstacleEnvelopeGeom(t)
	env := newRunoutEnvelope(cyl)
	feet := ellipseLowerArcSamples(of.Nodes[0], of.Nodes[1], 15, 10, stations)
	rows, ok := obstacleCanalStations(env, wall, host, feet, ResolutionForSize(50).Weld())
	if !ok {
		t.Fatalf("obstacleCanalStations declined on the T6 obstacle at %d stations", stations)
	}
	return &obstacleCanal{
		Centres: rows.centres, FeetRim: rows.feetRim, FeetWall: rows.feetWall,
		Envelope: BallEnvelope{
			Tangents: []geom.Surface{wall}, Through: []geom.Curve3{rim},
			Spine: env.spine, Radius: env.radius,
		},
	}
}

// TestObstacleCanalStationsAreOnTheEnvelope checks every solved station against the surf-rst definition.
// Which assertions are EVIDENCE and which are construction invariants matters, so they are labelled:
//
//	|FeetRim − centre| = r   EVIDENCE. The only quantity surfRstCentre actually solves for; it is the
//	                        "passes THROUGH the rim point" half of surf-rst.
//	centre ⟂ its station    EVIDENCE. The centre must lie in the rim foot's OWN section plane. A station
//	                        solved at the wrong spine coordinate fails here and nowhere else.
//	centre on the near root EVIDENCE. surfRstCentre picks one of TWO roots on L_B, both at radius r from
//	                        the rim point, so no radius test can tell them apart. The wrong (far) root
//	                        lands about 2r away from the plain axis and on the far side of the host
//	                        plane; the physical ball is displaced only by the rim's intrusion, which is
//	                        bounded by r. This is the assertion a branch-sign regression trips.
//	FeetRim identity        EVIDENCE (cheap): the rim row must still BE the supplied rim points, in
//	                        order — the weld the patch's u=0 boundary rests on.
//	|FeetWall − centre| = r  CONSTRUCTION INVARIANT: FeetWall is defined as centre − r·n̂ with n̂ unit.
//	FeetWall on the wall     CONSTRUCTION INVARIANT (restated tangency): n̂ is the wall's own normal and
//	                        the centre sits on L_B, r from the wall, by construction. Kept because it is
//	                        what geom.LoftCanalStations itself re-checks, not because it can fail here.
func TestObstacleCanalStationsAreOnTheEnvelope(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	feet := ellipseLowerArcSamples(of.Nodes[0], of.Nodes[1], 15, 10, 41)
	c := t6ObstacleCanal(t, of, 41)
	cyl, wall, host, _ := t6ObstacleEnvelopeGeom(t)
	for j := range c.Centres {
		assertStationFeetAtRadius(t, c, j)
		assertStationInItsOwnSection(t, c, j, cyl)
		assertStationOnTheNearRoot(t, c, j, cyl, host)
		if c.FeetRim[j] != feet[j] {
			t.Errorf("station %d rim foot = %v, want the supplied rim point %v unchanged", j, c.FeetRim[j], feet[j])
		}
		if d := stdmath.Abs(float64(wall.Normal().Dot(wall.Origin.VectorTo(c.FeetWall[j])))); d > 1e-12 {
			t.Errorf("station %d wall foot is %.3e off the wall plane (tangency is not a tangency)", j, d)
		}
	}
}

// assertStationFeetAtRadius checks both feet sit at the ball radius from the centre. The RIM foot is the
// evidence (surfRstCentre solved it); the WALL foot is a construction invariant.
func assertStationFeetAtRadius(t *testing.T, c *obstacleCanal, j int) {
	t.Helper()
	for _, f := range []struct {
		name string
		p    math.Point3
	}{{"rim", c.FeetRim[j]}, {"wall", c.FeetWall[j]}} {
		if d := stdmath.Abs(float64(f.p.DistanceTo(c.Centres[j])) - 6); d > 1e-12 {
			t.Fatalf("station %d %s foot is %.3e off the ball radius 6", j, f.name, d)
		}
	}
}

// assertStationInItsOwnSection checks the centre lies in the section plane Π(s) of ITS OWN rim foot — the
// plane normal to the spine through that foot. This is what makes the emitted rim boundary the station's
// contact rather than some neighbour's.
func assertStationInItsOwnSection(t *testing.T, c *obstacleCanal, j int, cyl geom.Cylinder) {
	t.Helper()
	e := cyl.AxisDir.AsVector()
	if d := stdmath.Abs(float64(c.FeetRim[j].VectorTo(c.Centres[j]).Dot(e))); d > 1e-12 {
		t.Errorf("station %d centre is %.3e off its rim foot's own section plane — it was solved at the wrong spine station", j, d)
	}
}

// assertStationOnTheNearRoot rejects the far surf-rst root: the centre must stay within one ball radius
// of the plain fillet axis and on the SAME side of the host plane as that axis. The far root satisfies
// every radius/tangency test and violates both of these (it lands ~2r off the axis, across the host).
func assertStationOnTheNearRoot(t *testing.T, c *obstacleCanal, j int, cyl geom.Cylinder, host geom.Plane) {
	t.Helper()
	axis := axisPointNearest(cyl, c.Centres[j])
	if off := float64(axis.DistanceTo(c.Centres[j])); off > 6 {
		t.Errorf("station %d centre is %.6f from the plain fillet axis, past the ball radius 6 — this is the FAR surf-rst root", j, off)
	}
	sideC := float64(host.Normal().Dot(host.Origin.VectorTo(c.Centres[j])))
	sideA := float64(host.Normal().Dot(host.Origin.VectorTo(cyl.Origin)))
	if sideC*sideA <= 0 {
		t.Errorf("station %d centre sits %.6f from the host plane, on the opposite side from the plain axis (%.6f) — this is the FAR surf-rst root", j, sideC, sideA)
	}
}

// axisPointNearest projects p onto the plain fillet axis — the point A(s) the surf-rst offset t is
// measured from.
func axisPointNearest(cyl geom.Cylinder, p math.Point3) math.Point3 {
	e := cyl.AxisDir.AsVector()
	return cyl.Origin.TranslateBy(e.Scale(cyl.Origin.VectorTo(p).Dot(e)))
}

// TestObstacleCanalWallFootLeavesThePlainSeam is the geometric heart of the change, asserted without any
// oracle: the ball is pushed OFF the plain fillet axis by the obstacle rim, so its wall tangency foot does
// NOT stay on the plain wall seam (z = -6) that the Coons model used as its c0 rail. At the two NODES the
// offset is algebraically zero (there the rim point IS the plain host contact); in between it is strictly
// positive and reaches its maximum at mid-span. A future refactor that quietly reinstates the straight
// seam fails here.
func TestObstacleCanalWallFootLeavesThePlainSeam(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	c := t6ObstacleCanal(t, of, 41)
	last := len(c.FeetWall) - 1
	for _, j := range []int{0, last} {
		if stdmath.Abs(float64(c.FeetWall[j].Z)+6) > 1e-6 {
			t.Errorf("node station %d wall foot z = %.9f, want the plain seam -6 (offset must vanish at a node)", j, c.FeetWall[j].Z)
		}
	}
	worst := 0.0
	for j := 1; j < last; j++ {
		worst = stdmath.Max(worst, float64(c.FeetWall[j].Z)+6)
	}
	if worst < 0.5 {
		t.Errorf("the wall foot locus rises only %.6f above the plain seam; T6's bulge is ~0.79 — the surf-rst offset is not being applied", worst)
	}
}

// TestObstacleCanalCertifiesWhereTheCoonsFillCannot is the falsification, both directions, hermetic: over
// the SAME declared envelope the Coons fill through the same four rails reads a MaxBallDev three to four
// orders of magnitude over the model weld, while the canal loft reads inside it. The Coons number is
// asserted to be LARGE first, so the premise cannot silently evaporate under the test.
func TestObstacleCanalCertifiesWhereTheCoonsFillCannot(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	res := ResolutionForSize(50)
	c := t6ObstacleCanal(t, of, 145)

	coons, _, ok := bsplineObstacleProvider{}.Build(CornerBlendRequest{ObstacleFeature: of, Setback: res})
	if !ok {
		t.Fatal("the Coons obstacle fill must still build (this test compares the two models, not their builders)")
	}
	coonsSurf, isBS := coons.Surface.(geom.BSplineSurface)
	if !isBS {
		t.Fatalf("the Coons obstacle fill is %T, want geom.BSplineSurface", coons.Surface)
	}
	coonsDev := maxBallDev(coonsSurf, &c.Envelope)
	if coonsDev <= 100*res.Weld() {
		t.Fatalf("the Coons fill's interior residual is only %.3e (weld %.3e) — the defect this replaces is gone, so the comparison is meaningless", coonsDev, res.Weld())
	}

	of.Canal = c
	patch, cert, ok := obstacleCanalProvider{}.Build(CornerBlendRequest{ObstacleFeature: of, Setback: res})
	if !ok {
		t.Fatal("the obstacle canal must build on the T6 obstacle")
	}
	if patch.Kind != BlendKindObstacleCanal {
		t.Errorf("patch Kind = %q, want %q", patch.Kind, BlendKindObstacleCanal)
	}
	if !cert.Valid(res) {
		t.Errorf("the canal certificate must be valid: closed=%v nofold=%v dev=%.3e angle=%.3e ball=%.3e (weld %.3e)",
			cert.Closed, cert.NoFold, cert.MaxDev, cert.MaxAngleDev, cert.MaxBallDev, res.Weld())
	}
	if cert.MaxBallDev >= coonsDev/1000 {
		t.Errorf("canal MaxBallDev %.3e is not 1000x better than the Coons fill's %.3e", cert.MaxBallDev, coonsDev)
	}
	t.Logf("interior residual: coons4-style fill %.6e, surf-rst canal %.6e, weld %.3e (%.0fx apart)",
		coonsDev, cert.MaxBallDev, res.Weld(), coonsDev/cert.MaxBallDev)
}

// TestObstacleCanalProviderDeclinesWithoutPayload pins the recognition signal: the tier keys on the
// ObstacleFeature.Canal payload alone, so a junction request and a payload-free obstacle both fall
// through untouched to the Coons tier.
func TestObstacleCanalProviderDeclinesWithoutPayload(t *testing.T) {
	t.Parallel()
	var p obstacleCanalProvider
	if p.Fits(CornerBlendRequest{}) {
		t.Error("the obstacle canal tier must not claim a junction request")
	}
	if p.Fits(CornerBlendRequest{ObstacleFeature: &ObstacleFeature{}}) {
		t.Error("the obstacle canal tier must not claim an obstacle with no surf-rst payload")
	}
	if got := p.Name(); got != BlendKindObstacleCanal {
		t.Errorf("Name() = %q, want %q", got, BlendKindObstacleCanal)
	}
}

// TestObstacleCanalRimFeetChecksItsPreconditionsBeforeSolving pins both of obstacleCanalRimFeet's
// PRECONDITION declines — too few rim samples to loft, and a dip that re-enters the band — and pins that
// each is checked BEFORE any interior rim solve runs: the zero-value obstacleDetection/edgeFillet handed
// in here carry no rim curve at all, so a single dipRimPointAtStation call would panic rather than return.
// Both must decline with a NIL row, like every other decline path in this file.
func TestObstacleCanalRimFeetChecksItsPreconditionsBeforeSolving(t *testing.T) {
	t.Parallel()
	cyl, _, _, _ := t6ObstacleEnvelopeGeom(t)
	env := newRunoutEnvelope(cyl)
	for _, tc := range []struct {
		what   string
		mangle func(*ObstacleFeature)
	}{
		{"a 3-sample dip (too coarse for the cubic v-interpolant)", func(of *ObstacleFeature) { of.RimArcPts = of.RimArcPts[:3] }},
		{"a dip that re-enters the band", func(of *ObstacleFeature) { of.RimArcPts[4] = of.RimArcPts[1] }},
	} {
		of := newT6Obstacle(t)
		tc.mangle(of)
		feet, ok := obstacleCanalRimFeet(env, of, obstacleDetection{}, edgeFillet{})
		if ok {
			t.Errorf("%s must decline", tc.what)
		}
		if feet != nil {
			t.Errorf("%s declined with a %d-point row, want nil (a decline must never hand back a partial payload)", tc.what, len(feet))
		}
	}
}

// TestObstacleCanalInteriorConvergesLikeTheCubicItIs is the guard on the K choice itself
// (obstacleCanalSubdiv), and it is driven BY that constant: the ladder is K/4 → K/2 → K, so changing the
// constant moves this test. It asserts four things, and each has a distinct job:
//
//   - the interior residual FALLS like the h^4 of a cubic v-interpolant (≥8x per doubling) — the
//     PROPERTY that makes K a measured value rather than a lucky one;
//   - the patch AREA does NOT move across the ladder — the premise that area is blind here, and would
//     have said nothing (the U4 core-panel trap, u4-canal-report.md concern 3);
//   - at K the residual is INSIDE the model weld — K is sufficient;
//   - at K/2 it is OUTSIDE — K is the FIRST value inside, which is the report's actual claim and what
//     makes an over-refined K (whose ~7 extra rim solves per gap per case are then unpaid-for) fail too.
//
// So the constant is gated from BOTH sides here: K=4 fails the sufficiency assertion, K=16 fails the
// first-inside assertion. Neither is a matter of taste — 4 leaves the interior uncertified on all five
// shipped cases, and 16 spends solves the weld does not ask for. If a future weld genuinely demands 16,
// this test is the place that must be re-measured, loudly, rather than drifting.
func TestObstacleCanalInteriorConvergesLikeTheCubicItIs(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	res := ResolutionForSize(50)
	devs := make([]float64, 0, 3)
	prev, prevArea := 0.0, 0.0
	for _, k := range obstacleCanalSubdivLadder(t) {
		n := k*(len(of.RimArcPts)-1) + 1 // the station count buildObstacleCanal produces at subdiv k
		dev, area := t6ObstacleCanalInterior(t, of, n, res)
		if prev > 0 {
			assertCanalCubicFalloff(t, n, dev, prev, area, prevArea)
		}
		t.Logf("subdiv %2d (%3d stations): MaxBallDev %.6e   area %.9f", k, n, dev, area)
		devs, prev, prevArea = append(devs, dev), dev, area
	}
	assertShippedSubdivIsFirstInsideTheWeld(t, devs, res.Weld())
}

// obstacleCanalSubdivLadder is the shipped constant's own halving ladder K/4, K/2, K — so a mutated
// constant measures a different ladder. K < 4 cannot be measured this way at all and is rejected here:
// two doublings are the minimum that can show an h^4 falloff.
func obstacleCanalSubdivLadder(t *testing.T) []int {
	t.Helper()
	if obstacleCanalSubdiv < 4 {
		t.Fatalf("obstacleCanalSubdiv = %d: too small for a two-doubling convergence ladder, so the constant cannot be justified by measurement at all", obstacleCanalSubdiv)
	}
	return []int{obstacleCanalSubdiv / 4, obstacleCanalSubdiv / 2, obstacleCanalSubdiv}
}

// t6ObstacleCanalInterior lofts the synthetic T6 canal at n stations and returns its interior residual
// and its patch area — the two quantities the K ladder compares.
func t6ObstacleCanalInterior(t *testing.T, of *ObstacleFeature, n int, res Resolution) (float64, float64) {
	t.Helper()
	c := t6ObstacleCanal(t, of, n)
	surf, err := geom.LoftCanalStations(c.Centres, c.FeetRim, c.FeetWall, c.Envelope.Radius, res.Weld())
	if err != nil {
		t.Fatalf("%d stations: loft: %v", n, err)
	}
	return maxBallDev(surf, &c.Envelope), surfacePatchArea(surf)
}

// assertCanalCubicFalloff is the anti-K=9-trap pair: the residual must gain like a cubic interpolant's
// h^4 while the area stays put, so the ladder is read on the INTERIOR and never on area.
func assertCanalCubicFalloff(t *testing.T, n int, dev, prev, area, prevArea float64) {
	t.Helper()
	if ratio := prev / dev; ratio < 8 {
		t.Errorf("%d stations: interior residual %.3e only %.1fx better than the half-density %.3e; a cubic interpolant must gain ~16x per doubling", n, dev, ratio, prev)
	}
	if rel := stdmath.Abs(area-prevArea) / prevArea; rel > 1e-4 {
		t.Errorf("%d stations: area moved %.3e relative — the premise that AREA is saturated (and therefore blind) no longer holds, so the constant's justification must be revisited", n, rel)
	}
}

// assertShippedSubdivIsFirstInsideTheWeld pins the constant's VALUE, not just its convergence: the
// shipped K must certify the interior and K/2 must not.
func assertShippedSubdivIsFirstInsideTheWeld(t *testing.T, devs []float64, weld float64) {
	t.Helper()
	shipped, half := devs[len(devs)-1], devs[len(devs)-2]
	if shipped > weld {
		t.Errorf("obstacleCanalSubdiv = %d leaves the interior residual at %.3e, OUTSIDE the %.3e model weld — the patch would not certify", obstacleCanalSubdiv, shipped, weld)
	}
	if half <= weld {
		t.Errorf("obstacleCanalSubdiv = %d is over-refined: half of it already reads %.3e, inside the %.3e model weld, so the extra %d rim solves per dip gap buy nothing",
			obstacleCanalSubdiv, half, weld, obstacleCanalSubdiv/2)
	}
	t.Logf("model weld %.3e: subdiv %d reads %.6e (%.1fx the weld), subdiv %d reads %.6e (%.1fx)",
		weld, obstacleCanalSubdiv, shipped, shipped/weld, obstacleCanalSubdiv/2, half, half/weld)
}

// surfacePatchArea integrates |S_u x S_v| over the patch domain on a Gauss-free uniform grid — enough to
// see whether the area MOVES between station densities, which is all the convergence gate needs.
func surfacePatchArea(s geom.BSplineSurface) float64 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	const n = 96
	du, dv := (u1-u0)/n, (v1-v0)/n
	total := 0.0
	for i := range n {
		for j := range n {
			total += float64(surfaceNormal(s, u0+(float64(i)+0.5)*du, v0+(float64(j)+0.5)*dv).Length()) * du * dv
		}
	}
	return total
}
