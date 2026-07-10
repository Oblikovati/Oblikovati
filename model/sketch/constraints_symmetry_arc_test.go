// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// p2close reports whether p is within tol of (x,y).
func p2close(p math.Point2, x, y, tol float64) bool {
	dx, dy := float64(p.X)-x, float64(p.Y)-y
	return dx*dx+dy*dy <= tol*tol
}

const solveTol = 1e-6

// TestArcMidpointSeedsAndSolves: AddMidpointToArc seeds the point at the arc-length midpoint,
// and the point tracks the midpoint as the arc's sweep changes (#1872). A quarter arc (origin,
// start (1,0), end (0,1), CCW) has its midpoint at 45°; a half arc (end (−1,0)) at 90°.
func TestArcMidpointSeedsAndSolves(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	p := s.Points().Add(math.P2(5, 5)) // arbitrary; the factory seeds it to the arc midpoint
	c := g.AddMidpointToArc(p, arc)

	const invSqrt2 = 0.70710678118
	if !p2close(p.Position(), invSqrt2, invSqrt2, solveTol) {
		t.Fatalf("seeded midpoint = %v, want (0.707,0.707) at the quarter arc's 45°", p.Position())
	}
	if !satisfied(c) {
		t.Fatalf("arc midpoint residuals not zero at the seeded midpoint: %v", c.Residuals())
	}

	// Widen the arc to a half circle (end at (−1,0), still radius 1) and pin the arc; the point
	// must track the new midpoint at 90° = (0,1), not jump to the complementary arc's (0,−1).
	arc.End.SetPosition(math.P2(-1, 0))
	g.AddFix(arc.Center)
	g.AddFix(arc.Start)
	g.AddFix(arc.End)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if !p2close(p.Position(), 0, 1, solveTol) {
		t.Errorf("half-arc midpoint = %v, want (0,1)", p.Position())
	}
}

// TestArcMidpointClockwise: the seed follows the arc's own sweep direction — a CW arc from
// (0,1) to (1,0) also bisects at 45°, on the same side as the swept path (#1872).
func TestArcMidpointClockwise(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(0, 1), math.P2(1, 0), false)
	p := s.Points().Add(math.P2(0, 0))
	g.AddMidpointToArc(p, arc)
	const invSqrt2 = 0.70710678118
	if !p2close(p.Position(), invSqrt2, invSqrt2, solveTol) {
		t.Errorf("CW quarter-arc midpoint = %v, want (0.707,0.707)", p.Position())
	}
}

// TestLineSymmetrySolvesToMirror: two lines made symmetric about the Y axis, with the first
// line and the axis pinned, drive the second line onto the exact mirror image (#1870).
func TestLineSymmetrySolvesToMirror(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	axis := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 1)) // Y axis
	l1 := s.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(2, 1))
	l2 := s.Lines().AddByTwoPoints(math.P2(-1.2, 0.1), math.P2(-1.8, 0.9)) // near the mirror
	g.AddLineSymmetry(l1, l2, axis)
	for _, p := range []*Point{l1.A, l1.B, axis.A, axis.B} {
		g.AddFix(p)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	// Mirror of (1,0),(2,1) across the Y axis is (−1,0),(−2,1); the straight pairing (A↔A) is
	// the one the initial geometry is closer to.
	if !p2close(l2.A.Position(), -1, 0, solveTol) || !p2close(l2.B.Position(), -2, 1, solveTol) {
		t.Errorf("mirrored line = %v..%v, want (−1,0)..(−2,1)", l2.A.Position(), l2.B.Position())
	}
}

// TestLineSymmetryCrossedPairing: when the second line is drawn with its endpoints swapped
// relative to the first, the constraint picks the crossed pairing and still mirrors (#1870).
func TestLineSymmetryCrossedPairing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	axis := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 1))
	l1 := s.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(2, 1))
	// l2 endpoints are swapped: A near mirror of l1.B, B near mirror of l1.A.
	l2 := s.Lines().AddByTwoPoints(math.P2(-1.8, 0.9), math.P2(-1.2, 0.1))
	c := g.AddLineSymmetry(l1, l2, axis)
	if !c.cross {
		t.Error("expected the crossed endpoint pairing for swapped endpoints")
	}
	for _, p := range []*Point{l1.A, l1.B, axis.A, axis.B} {
		g.AddFix(p)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	// Crossed: l2.A mirrors l1.B → (−2,1); l2.B mirrors l1.A → (−1,0).
	if !p2close(l2.A.Position(), -2, 1, solveTol) || !p2close(l2.B.Position(), -1, 0, solveTol) {
		t.Errorf("crossed mirror = %v..%v, want (−2,1)..(−1,0)", l2.A.Position(), l2.B.Position())
	}
}

// TestCircularSymmetrySolves: two circles made symmetric about the Y axis mirror their centres
// and equalise their radii (#1870).
func TestCircularSymmetrySolves(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	axis := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 1))
	c1 := s.Circles().AddByCenterRadius(math.P2(3, 1), 2)
	c2 := s.Circles().AddByCenterRadius(math.P2(-2.5, 1.2), 1.0)
	g.AddCircularSymmetry(c1, c2, axis)
	g.AddFix(c1.Center)
	for _, p := range []*Point{axis.A, axis.B} {
		g.AddFix(p)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if !p2close(c2.Center.Position(), -3, 1, solveTol) {
		t.Errorf("mirrored centre = %v, want (−3,1)", c2.Center.Position())
	}
	if d := float64(c1.Radius - c2.Radius); d > solveTol || d < -solveTol {
		t.Errorf("radii not equal after symmetry: r1=%v r2=%v", c1.Radius, c2.Radius)
	}
}
