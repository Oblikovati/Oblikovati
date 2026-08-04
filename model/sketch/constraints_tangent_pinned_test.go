// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// Regression tests for the pinned-touch-point tangency formulations (#2014).
//
// The defect: when a line endpoint is also one of an arc's defining points, the arc's radius is
// structurally |centre − P|, so "perpendicular distance from centre to line equals radius"
// reduces to R(|sin φ| − 1). That peaks at the φ = 90° solution, giving a DOUBLE root — residual
// zero AND derivative zero. The constraint contributed no rank, so the solver neither enforced
// nor reported it, and every slot and fillet built that way came out floppy.
//
// These tests assert RANK, not residual. The broken formulation has a residual of zero at the
// solution too — only the Jacobian tells the two apart.

// tangentRankGain reports how much rank a constraint adds, by measuring before and after.
func tangentRankGain(t *testing.T, build func(*Sketch) func()) int {
	t.Helper()
	s := NewSketches().Add(XYPlane())
	apply := build(s)
	before := s.AnalyzeConstraints().Rank
	apply()
	return s.AnalyzeConstraints().Rank - before
}

func TestTangentAtSharedPointContributesRank(t *testing.T) {
	gain := tangentRankGain(t, func(s *Sketch) func() {
		center := s.Points().Add(math.P2(0, 0))
		touch := s.Points().Add(math.P2(0, 1))
		other := s.Points().Add(math.P2(0, -1))
		arc := s.Arcs().Add(center, touch, other, true)
		far := s.Points().Add(math.P2(5, 1)) // exactly tangent: the degenerate configuration
		line := s.Lines().Add(touch, far)
		return func() { s.GeometricConstraints().AddTangent(line, arc) }
	})
	if gain != 1 {
		t.Errorf("rank gain = %d, want 1 — a tangency at a shared point must constrain something", gain)
	}
}

func TestTangentWithFreeTouchPointStillContributesRank(t *testing.T) {
	gain := tangentRankGain(t, func(s *Sketch) func() {
		line := s.Lines().AddByTwoPoints(math.P2(-5, 1), math.P2(5, 1))
		circle := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
		return func() { s.GeometricConstraints().AddTangent(line, circle) }
	})
	if gain != 1 {
		t.Errorf("rank gain = %d, want 1 — the free-touch-point formulation must be unchanged", gain)
	}
}

func TestCircularTangentAtSharedPointContributesRank(t *testing.T) {
	gain := tangentRankGain(t, func(s *Sketch) func() {
		// Two arcs meeting at a shared point, already exactly tangent: centres and the touch
		// point are collinear, which is where the centre-distance residual is stationary.
		c1 := s.Points().Add(math.P2(0, 0))
		touch := s.Points().Add(math.P2(2, 0))
		e1 := s.Points().Add(math.P2(0, 2))
		a1 := s.Arcs().Add(c1, touch, e1, true)
		c2 := s.Points().Add(math.P2(3, 0))
		e2 := s.Points().Add(math.P2(3, 1))
		a2 := s.Arcs().Add(c2, touch, e2, true)
		return func() { s.GeometricConstraints().AddCircularTangent(a1, a2) }
	})
	if gain != 1 {
		t.Errorf("rank gain = %d, want 1 — a circle-circle tangency at a shared point must constrain something", gain)
	}
}

// The pinned formulation must actually drive the geometry to tangency, not merely report rank.
func TestPinnedTangencyConvergesFromOffTangent(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	center := s.Points().Add(math.P2(0, 0))
	touch := s.Points().Add(math.P2(0, 1))
	other := s.Points().Add(math.P2(0, -1))
	arc := s.Arcs().Add(center, touch, other, true)
	far := s.Points().Add(math.P2(5, 3)) // clearly not tangent
	line := s.Lines().Add(touch, far)
	s.GeometricConstraints().AddTangent(line, arc)
	s.GeometricConstraints().AddGroundPoints(center, touch)
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve did not converge: %+v", res)
	}
	u := line.A.Position().VectorTo(line.B.Position())
	radial := touch.Position().VectorTo(center.Position())
	cos := u.Dot(radial) / (u.Length() * radial.Length())
	if cos > 1e-9 || cos < -1e-9 {
		t.Errorf("line is not perpendicular to the radius at the touch point (cos = %v)", cos)
	}
}
