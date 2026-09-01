// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR2 — the closed-form spring/foot primitives (fillet_arm_springs.go). The D5 sphere+plane torus springs
// and their feet are pinned by the arm-capping tests; these pin the decline/branch corners of the primitives
// the sphere slice does not exercise but whose "never fabricate" promise must hold for the future torus
// family: the coaxial-only torus∩sphere spring and the near-parallel ruling∩plane foot.

// springTol builds the same scale-local weld torusSphereSpring is called with (points that bound the arm).
func springTol(pts ...math.Point3) float64 {
	return geom.ResolutionForPoints(pts).Weld()
}

// TestTorusSphereSpring_NonCoaxialDeclines pins the coaxial-only guard: an OFF-AXIS host sphere has no
// full latitude-circle of tangency (far-runout-port-math §2 — the |P−O|² gains a cos(u−Ψ) term), so the
// spring must decline even when the coaxial tangency test |rhs|≈amp is coincidentally satisfied. Here the
// sphere is offset A⊥=30 perpendicular to the torus axis yet |rhs|=amp exactly, so WITHOUT the A⊥ guard the
// old code would fabricate a wrong spring. MUTATION PROOF: delete the `aPerp > tol` guard and this fails.
func TestTorusSphereSpring_NonCoaxialDeclines(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 1, 0), math.V3(1, 0, 0), 50, 10)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	sp, err := geom.NewSphere(math.P3(30, 0, 0), 50) // 30 ⊥ the ŷ axis; |rhs|=amp=1000 (would otherwise pass)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	tol := springTol(tor.Center, sp.Center)

	// Document that the tangency test alone would ADMIT this off-axis sphere — the A⊥ guard is the sole reason.
	d := sp.Center.VectorTo(tor.Center)
	amp := stdmath.Hypot(2*tor.MajorRadius*tor.MinorRadius, 2*float64(d.Dot(tor.AxisDir.AsVector()))*tor.MinorRadius)
	rhs := sp.Radius*sp.Radius - float64(d.LengthSquared()) - tor.MajorRadius*tor.MajorRadius - tor.MinorRadius*tor.MinorRadius
	if stdmath.Abs(stdmath.Abs(rhs)-amp) > tol {
		t.Fatalf("precondition: the coaxial tangency test must pass (|rhs|=%.6f amp=%.6f) so the A⊥ guard is what declines", rhs, amp)
	}
	if c, ok := torusSphereSpring(tor, sp, tol); ok {
		t.Fatalf("an off-axis sphere (A⊥=30 > tol) must decline (never fabricate); got %v", c)
	}
}

// TestTorusSphereSpring_CoaxialTangent pins the true path AND the ±amp π-branch: a coaxial sphere R_c=60
// is tangent to the OUTER equator (v=0, rhs=+amp → circle radius 60); R_c=40 is tangent to the INNER
// equator (v=π, rhs=−amp → circle radius 40). Without the `rhs<0 ⇒ v+=π` fix the inner case would return
// radius 60 (π-wrong), so the radius assertion pins that fix too.
func TestTorusSphereSpring_CoaxialTangent(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 1, 0), math.V3(1, 0, 0), 50, 10)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	for _, tc := range []struct {
		name       string
		radius     float64
		wantRadius float64
	}{
		{"outer equator (rhs=+amp, v=0)", 60, 60},
		{"inner equator (rhs=−amp, v=π)", 40, 40},
	} {
		sp, err := geom.NewSphere(math.P3(0, 0, 0), tc.radius)
		if err != nil {
			t.Fatalf("%s sphere: %v", tc.name, err)
		}
		circle, ok := torusSphereSpring(tor, sp, springTol(tor.Center, sp.Center))
		if !ok {
			t.Fatalf("%s: coaxial tangent sphere must yield a spring", tc.name)
		}
		if stdmath.Abs(circle.Radius-tc.wantRadius) > 1e-9 {
			t.Fatalf("%s: spring radius %.10f, want %.1f (π-branch fix)", tc.name, circle.Radius, tc.wantRadius)
		}
	}
}

// TestLinePlaneFoot_NearParallelDeclines pins the near-parallel ruling guard (fix: exact `denom==0` → a
// model-relative `|n̂·d̂| ≤ sinFloor`). A ruling almost in the plane (|n̂·d̂|=1e-7 < sinFloor) declines; a
// transverse ruling crosses.
func TestLinePlaneFoot_NearParallelDeclines(t *testing.T) {
	t.Parallel()
	pl := planeOn(t, math.P3(0, 0, 5), math.V3(0, 0, 1))
	grazing, err := geom.NewLine(math.P3(0, 0, 0), math.V3(1, 0, 1e-7)) // |n̂·d̂|≈1e-7 < sinFloor
	if err != nil {
		t.Fatalf("grazing line: %v", err)
	}
	if p, ok := linePlaneFoot(grazing, pl); ok {
		t.Fatalf("a near-parallel ruling must decline (|n̂·d̂| ≤ sinFloor); got %v", p)
	}
	crossing, err := geom.NewLine(math.P3(0, 0, 0), math.V3(0, 0, 1)) // straight through the plane
	if err != nil {
		t.Fatalf("crossing line: %v", err)
	}
	foot, ok := linePlaneFoot(crossing, pl)
	if !ok {
		t.Fatalf("a transverse ruling must cross the plane; declined")
	}
	if stdmath.Abs(float64(foot.Z)-5) > 1e-9 {
		t.Fatalf("crossing foot z=%v, want 5", foot.Z)
	}
}
