// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestRectangularPatternLiveTracksSpacing verifies a pattern clone follows a live
// (parameter-driven) spacing: changing the step closure and re-solving repositions the
// copy, which is what makes a sketch pattern's spacing parametric.
func TestRectangularPatternLiveTracksSpacing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	seed := s.Circles().Add(s.Points().Add(math.P2(0, 0)), 0.5)
	s.GeometricConstraints().AddGround(seed.Center)

	spacing := math.Scalar(2)
	step1 := func() math.Vector2 { return math.V2(spacing, 0) }
	step2 := func() math.Vector2 { return math.Vector2{} }
	clones, err := s.RectangularPatternLive([]Entity{seed}, step1, 2, step2, 1)
	if err != nil {
		t.Fatalf("RectangularPatternLive: %v", err)
	}
	if len(clones) != 1 {
		t.Fatalf("got %d clones, want 1 (2×1 grid minus seed)", len(clones))
	}
	clone := clones[0].(*Circle)

	s.Solve()
	if got := float64(clone.Center.X); stdmath.Abs(got-2) > 1e-9 {
		t.Fatalf("clone X = %v at spacing 2, want 2", got)
	}

	spacing = 5 // a parameter drove the spacing wider
	s.Solve()
	if got := float64(clone.Center.X); stdmath.Abs(got-5) > 1e-9 {
		t.Errorf("clone X = %v after spacing→5, want 5 (clone did not track spacing)", got)
	}
}

// TestCircularPatternIsRigidAndTracks verifies a circular pattern fully constrains its clones
// (no free DOF beyond the seed's) and rotates each clone to its place. Regression for the
// circular pattern omitting clone constraints — the clones floated free, leaving a 3-hole bolt
// circle 6 DOF short (the dual of the rectangular-pattern fix).
func TestCircularPatternIsRigidAndTracks(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	seed := s.Circles().Add(s.Points().Add(math.P2(1, 0)), 0.1)
	s.GeometricConstraints().AddGroundPoints(seed.Center) // seed centre fixed at (1,0); radius free

	clones, err := s.CircularPattern([]Entity{seed}, math.P2(0, 0), 3, 2*stdmath.Pi)
	if err != nil {
		t.Fatalf("CircularPattern: %v", err)
	}
	if len(clones) != 2 {
		t.Fatalf("got %d clones, want 2 (3 instances minus seed)", len(clones))
	}
	s.Solve()
	// Only the seed's radius is free; if the clones were unconstrained it would be 1 + 2×3 = 7.
	if dof := s.DegreesOfFreedom(); dof != 1 {
		t.Fatalf("circular pattern left %d DOF, want 1 (clones unconstrained?)", dof)
	}
	// The first clone sits 120° round: (1,0) → (cos120°, sin120°) = (−0.5, 0.866).
	c1 := clones[0].(*Circle)
	if x, y := float64(c1.Center.X), float64(c1.Center.Y); stdmath.Abs(x+0.5) > 1e-6 || stdmath.Abs(y-stdmath.Sqrt(3)/2) > 1e-6 {
		t.Errorf("clone0 centre = (%.4f,%.4f), want (−0.5, 0.866)", x, y)
	}
	// The equal-radius link ties the clone's radius to the seed's.
	if r := float64(c1.Radius); stdmath.Abs(r-0.1) > 1e-9 {
		t.Errorf("clone0 radius = %.4f, want 0.1 (equal-radius link missing?)", r)
	}
}
