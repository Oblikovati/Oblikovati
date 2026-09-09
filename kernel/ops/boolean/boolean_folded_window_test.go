// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The FOLDED ruled∩quadric window, end to end (ADR-0061 stage 5).
//
// A ball crossing a rod off its axis is the smallest pair whose section folds: the rod's ruling meets
// the ball over part of the sweep and misses it outside, so the two roots of the ruling quadratic meet
// at the two azimuths where the discriminant vanishes and the section is ONE closed loop rather than
// two full wraps. Every recognizer the kernel used to carry sidestepped this — the coaxial ball-and-rod
// builders split their operands by construction — and the general intersector refused it, so the
// boolean refused the pair by name.
//
// These rows certify the result against an INDEPENDENT oracle: the analytic membership of the two
// primitives integrated by Monte Carlo. A B-rep that measures right could still be the wrong shape, so
// the face census is asserted with it — an integral and a topology together, not a volume alone.

// spanSphereOnRod is the fixture: a rod (r=1, z ∈ [−2,2]) and a ball (r=0.5) centred 1.4 from its axis,
// so the ball straddles the wall — inside it at 0.9, outside at 1.9.
func spanSphereOnRod(t *testing.T) (*topo.Body, *topo.Body) {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(0, 0, -2), math.V3(0, 0, 1), 1, 4)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	ball, err := brep.SolidSphere(math.P3(1.4, 0, 0), 0.5, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	return rod, ball
}

// TestFoldedWindowBooleanMatchesTheMembershipIntegral drives all three operations and checks each
// result's analytic volume against the membership integral, and its face census against the shape the
// contact implies.
func TestFoldedWindowBooleanMatchesTheMembershipIntegral(t *testing.T) {
	t.Parallel()
	rod, ball := spanSphereOnRod(t)
	for _, c := range []struct {
		name             string
		op               ops.PartFeatureOperation
		want             float64 // the membership integral, sampled below
		tol              float64 // the integral's own error, not the boolean's
		wantCyl, wantSph int
		wantPlan, faces  int
	}{
		// The rod's wall breached by the ball's seam, its two caps whole, and the ball's cap outside.
		{"join", ops.Join, 13.077910, 2e-3, 1, 1, 2, 4},
		// The same wall with the seam, the two caps, and the ball's cap now facing into the dimple.
		{"cut", ops.Cut, 12.555898, 2e-3, 1, 1, 2, 4},
		// The lens: a patch of the wall and a patch of the ball, nothing planar.
		{"intersect", ops.Intersect, 0.012187, 5e-3, 1, 1, 0, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := ops.Boolean(c.op, rod, ball)
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
				t.Fatalf("%s: not a valid closed manifold solid: %+v", c.name, r)
			}
			cyl, sph, plan := analyticFaceCensus(res)
			if cyl != c.wantCyl || sph != c.wantSph || plan != c.wantPlan || len(res.Faces()) != c.faces {
				t.Errorf("%s: %d cylinder + %d sphere + %d plane of %d faces, want %d + %d + %d of %d",
					c.name, cyl, sph, plan, len(res.Faces()), c.wantCyl, c.wantSph, c.wantPlan, c.faces)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			if rel := stdmath.Abs(got-c.want) / c.want; rel > c.tol {
				t.Errorf("%s: volume %.6f, want %.6f from the membership integral — rel %.4g > %.4g",
					c.name, got, c.want, rel, c.tol)
			}
		})
	}
}

// TestFoldedWindowSeamIsExact reads the seam itself: the edge where the ball meets the rod's wall must
// be a single closed curve reported EXACT (a closed form carries no chord deviation), and every point
// of it must sit on both surfaces.
func TestFoldedWindowSeamIsExact(t *testing.T) {
	t.Parallel()
	rod, ball := spanSphereOnRod(t)
	res, err := ops.Boolean(ops.Join, rod, ball)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if tol := res.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("the folded-window join reports AchievedBoundaryTolerance %g, want 0 — the section is a closed form", tol)
	}
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	sph, _ := geom.NewSphere(math.P3(1.4, 0, 0), 0.5)
	seams := 0
	for _, e := range res.Edges() {
		if _, isLine := e.Geometry().(geom.Circle); isLine {
			continue // the rod's two rim circles
		}
		seams++
		lo, hi := e.Geometry().Domain()
		for k := 0; k <= 128; k++ {
			p := e.Geometry().PointAt(lo + (hi-lo)*float64(k)/128)
			offCyl := stdmath.Abs(float64(geom.SignedDistanceToSurface(cyl, p)))
			offSph := stdmath.Abs(float64(geom.SignedDistanceToSurface(sph, p)))
			if offCyl > 1e-9 || offSph > 1e-9 { // tol:weld — the seam lies on both surfaces to rounding
				t.Fatalf("seam point %v sits %.3e off the rod and %.3e off the ball", p, offCyl, offSph)
			}
		}
	}
	if seams != 1 {
		t.Errorf("the join carries %d non-circular edges, want 1 (the ball's seam on the wall)", seams)
	}
}

// analyticFaceCensus tallies a body's faces by analytic surface kind.
func analyticFaceCensus(b *topo.Body) (cylinders, spheres, planes int) {
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cylinders++
		case geom.Sphere:
			spheres++
		case geom.Plane:
			planes++
		}
	}
	return cylinders, spheres, planes
}

// TestFoldedWindowMembershipIntegral is the oracle's own regression: it re-derives the three volumes
// the rows above assert, so a change to the fixture cannot silently drift the numbers they compare
// against. It is a Monte Carlo over the two primitives' analytic membership — deliberately not the
// kernel's own classifier, so the oracle is independent of what it gates.
func TestFoldedWindowMembershipIntegral(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		op   ops.PartFeatureOperation
		want float64
		tol  float64
	}{
		{"join", ops.Join, 13.077910, 2e-3},
		{"cut", ops.Cut, 12.555898, 2e-3},
		{"intersect", ops.Intersect, 0.012187, 5e-3},
	} {
		got := rodBallMembershipVolume(c.op)
		if rel := stdmath.Abs(got-c.want) / c.want; rel > c.tol {
			t.Errorf("%s: the membership integral is %.6f, the row asserts %.6f — rel %.4g", c.name, got, c.want, rel)
		}
	}
}

// rodBallMembershipVolume integrates "inside the rod op inside the ball" over a box that contains the
// result, with a fixed seed so the number is the same on every run and platform.
func rodBallMembershipVolume(op ops.PartFeatureOperation) float64 {
	lo, hi := math.P3(-1, -1, -2), math.P3(1.9, 1, 2)
	if op == ops.Intersect {
		lo, hi = math.P3(0.85, -0.5, -0.5), math.P3(1, 0.5, 0.5) // the lens alone, so the sampling is dense on it
	}
	rng := rand.New(rand.NewSource(7))
	span := hi.AsVector().Sub(lo.AsVector())
	boxVol := float64(span.X * span.Y * span.Z)
	hits := 0
	for range membershipSamples {
		x := float64(lo.X) + rng.Float64()*float64(span.X)
		y := float64(lo.Y) + rng.Float64()*float64(span.Y)
		z := float64(lo.Z) + rng.Float64()*float64(span.Z)
		inRod := x*x+y*y <= 1 && z >= -2 && z <= 2
		dx := x - 1.4
		inBall := dx*dx+y*y+z*z <= 0.25
		if keepsUnderOp(op, inRod, inBall) {
			hits++
		}
	}
	return boxVol * float64(hits) / membershipSamples
}

// keepsUnderOp is the Requicha membership rule for the three operations.
func keepsUnderOp(op ops.PartFeatureOperation, inTarget, inTool bool) bool {
	switch op {
	case ops.Join:
		return inTarget || inTool
	case ops.Cut:
		return inTarget && !inTool
	default:
		return inTarget && inTool
	}
}

// membershipSamples is the Monte Carlo's sample count: enough that its own relative error is a few
// parts in ten thousand on the two large volumes and a few in a thousand on the lens, which is what the
// rows' tolerances allow for. The kernel's answer is exact; the tolerance is the ORACLE's.
const membershipSamples = 20_000_000
