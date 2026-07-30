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

// TestObstacleCanalStationsAreOnTheEnvelope proves the closed form before any surface is built: every
// solved station's BOTH feet sit at exactly the ball radius from its centre, its wall foot lies ON the
// wall plane, and its rim foot is the supplied rim point unchanged. A station that fails this is a
// non-envelope station and geom.LoftCanalStations refuses to loft it.
func TestObstacleCanalStationsAreOnTheEnvelope(t *testing.T) {
	of := newT6Obstacle(t)
	c := t6ObstacleCanal(t, of, 41)
	_, wall, _, _ := t6ObstacleEnvelopeGeom(t)
	for j := range c.Centres {
		for _, f := range []struct {
			name string
			p    math.Point3
		}{{"rim", c.FeetRim[j]}, {"wall", c.FeetWall[j]}} {
			if d := stdmath.Abs(float64(f.p.DistanceTo(c.Centres[j])) - 6); d > 1e-12 {
				t.Fatalf("station %d %s foot is %.3e off the ball radius 6", j, f.name, d)
			}
		}
		if d := stdmath.Abs(float64(wall.Normal().Dot(wall.Origin.VectorTo(c.FeetWall[j])))); d > 1e-12 {
			t.Errorf("station %d wall foot is %.3e off the wall plane (tangency is not a tangency)", j, d)
		}
	}
}

// TestObstacleCanalWallFootLeavesThePlainSeam is the geometric heart of the change, asserted without any
// oracle: the ball is pushed OFF the plain fillet axis by the obstacle rim, so its wall tangency foot does
// NOT stay on the plain wall seam (z = -6) that the Coons model used as its c0 rail. At the two NODES the
// offset is algebraically zero (there the rim point IS the plain host contact); in between it is strictly
// positive and reaches its maximum at mid-span. A future refactor that quietly reinstates the straight
// seam fails here.
func TestObstacleCanalWallFootLeavesThePlainSeam(t *testing.T) {
	of := newT6Obstacle(t)
	c := t6ObstacleCanal(t, of, 41)
	last := len(c.FeetWall) - 1
	for _, j := range []int{0, last} {
		if d := stdmath.Abs(float64(c.FeetWall[j].Z) + 6); d > 1e-6 {
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

// TestObstacleCanalInteriorConvergesLikeTheCubicItIs is the guard on the K choice itself
// (obstacleCanalSubdiv): the interior residual must FALL as stations are added, and fall like the h^4 of
// a cubic v-interpolant. This is what makes the constant a measured value rather than a lucky one, and it
// is deliberately asserted on the INTERIOR condition — the patch's AREA is saturated across this whole
// range and would have said nothing (the U4 core-panel trap, u4-canal-report.md concern 3).
func TestObstacleCanalInteriorConvergesLikeTheCubicItIs(t *testing.T) {
	of := newT6Obstacle(t)
	res := ResolutionForSize(50)
	prev, prevArea := 0.0, 0.0
	for _, n := range []int{13, 25, 49} {
		c := t6ObstacleCanal(t, of, n)
		surf, err := geom.LoftCanalStations(c.Centres, c.FeetRim, c.FeetWall, c.Envelope.Radius, res.Weld())
		if err != nil {
			t.Fatalf("%d stations: loft: %v", n, err)
		}
		dev := maxBallDev(surf, &c.Envelope)
		area := surfacePatchArea(surf)
		if prev > 0 {
			if ratio := prev / dev; ratio < 8 {
				t.Errorf("%d stations: interior residual %.3e only %.1fx better than the half-density %.3e; a cubic interpolant must gain ~16x per doubling", n, dev, ratio, prev)
			}
			if rel := stdmath.Abs(area-prevArea) / prevArea; rel > 1e-4 {
				t.Errorf("%d stations: area moved %.3e relative — the premise that AREA is saturated (and therefore blind) no longer holds, so the constant's justification must be revisited", n, rel)
			}
		}
		t.Logf("%3d stations: MaxBallDev %.6e   area %.9f", n, dev, area)
		prev, prevArea = dev, area
	}
}

// surfacePatchArea integrates |S_u x S_v| over the patch domain on a Gauss-free uniform grid — enough to
// see whether the area MOVES between station densities, which is all the convergence gate needs.
func surfacePatchArea(s geom.BSplineSurface) float64 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	const n = 96
	du, dv := (u1-u0)/n, (v1-v0)/n
	total := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			total += float64(surfaceNormal(s, u0+(float64(i)+0.5)*du, v0+(float64(j)+0.5)*dv).Length()) * du * dv
		}
	}
	return total
}
