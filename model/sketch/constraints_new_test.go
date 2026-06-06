// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestGroundConstraintFixesEntity(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0)) // 2 points → 4 DOF
	if s.DegreesOfFreedom() != 4 {
		t.Fatalf("bare line DOF = %d, want 4", s.DegreesOfFreedom())
	}
	s.GeometricConstraints().AddGround(l)
	if dof := s.DegreesOfFreedom(); dof != 0 {
		t.Fatalf("grounded line DOF = %d, want 0", dof)
	}
}

func TestOffsetConstraintHoldsDistance(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	l2 := s.Lines().AddByTwoPoints(gmath.P2(0, 2), gmath.P2(4, 2)) // parallel, 2 above
	c := s.GeometricConstraints().AddOffset(l1, l2, 2)
	// Already satisfied ⇒ residuals ≈ 0.
	for _, r := range c.Residuals() {
		if math.Abs(r) > 1e-9 {
			t.Fatalf("offset residual = %v on a satisfied pair, want ~0", r)
		}
	}
}

func TestPatternLinkHoldsOffset(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	seed := s.Points().Add(gmath.P2(0, 0))
	member := s.Points().Add(gmath.P2(5, 0))
	c := s.GeometricConstraints().AddPatternLink(seed, member)
	for _, r := range c.Residuals() {
		if math.Abs(r) > 1e-9 {
			t.Fatalf("pattern-link residual = %v, want ~0", r)
		}
	}
	// Move the seed; the residual now reflects the member lagging behind.
	seed.X = 1
	if math.Abs(c.Residuals()[0]+1) > 1e-9 { // member.X-seed.X-dx = 5-1-5 = -1
		t.Fatalf("after moving seed, residual = %v, want -1", c.Residuals()[0])
	}
}

func TestNewConstraintsRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	l2 := s.Lines().AddByTwoPoints(gmath.P2(0, 2), gmath.P2(4, 2))
	p1 := s.Points().Add(gmath.P2(7, 7))
	p2 := s.Points().Add(gmath.P2(9, 7))
	s.GeometricConstraints().AddGround(l1)
	s.GeometricConstraints().AddOffset(l1, l2, 2)
	s.GeometricConstraints().AddPatternLink(p1, p2)

	out := roundTrip(t, sc)
	if got := out.GeometricConstraints().Count(); got != 3 {
		t.Fatalf("constraints after round trip = %d, want 3", got)
	}
}
