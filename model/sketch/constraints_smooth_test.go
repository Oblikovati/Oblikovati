// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestSmoothFrameByCurveType(t *testing.T) {
	s := NewSketches().Add(XYPlane())

	// Line: tangent points toward the other endpoint, zero curvature.
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	tan, curv, ok := l.smoothFrame(l.A)
	if !ok || !tan.IsEqualTo(math.V2(1, 0), 1e-9) || !curv.IsEqualTo(math.V2(0, 0), 1e-9) {
		t.Errorf("line frame at A = (%v,%v,%v), want tangent +X, zero curvature", tan, curv, ok)
	}
	if _, _, ok := l.smoothFrame(s.Points().Add(math.P2(9, 9))); ok {
		t.Error("line frame should report ok=false for a non-endpoint")
	}

	// Arc on the unit circle centred at (0,1): curvature vector points to the centre,
	// magnitude 1/r = 1.
	a := s.Arcs().AddByCenterStartEnd(math.P2(0, 1), math.P2(0, 0), math.P2(1, 1), true)
	_, curv, ok = a.smoothFrame(a.Start)
	if !ok || !curv.IsEqualTo(math.V2(0, 1), 1e-9) {
		t.Errorf("arc curvature at start = %v (ok=%v), want (0,1)", curv, ok)
	}
}

func TestSmoothTwoLines(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	// Two abutting collinear segments meeting at the origin (separate points there).
	l1 := s.Lines().AddByTwoPoints(math.P2(-1, 0), math.P2(0, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	sm := g.AddSmooth(l1, l2, l1.B, l2.A)
	if !satisfied(sm) {
		t.Error("collinear abutting lines should be smooth (G0+G1+G2 all zero)")
	}
	// Kink the second line: tangent continuity (G1) breaks.
	l2.B.SetPosition(math.P2(1, 1))
	if satisfied(sm) {
		t.Error("smooth should fail once the lines are no longer collinear")
	}
}

func TestSmoothDrivesSplineOntoLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	// A line pinned along the X axis, ending at the origin.
	line := s.Lines().AddByTwoPoints(math.P2(-2, 0), math.P2(0, 0))
	g.AddFix(line.A)
	g.AddFix(line.B)
	// A spline that starts off the line with a bend at its second point.
	sp := s.Splines().AddByControlPoints([]math.Point2{{X: 0.5, Y: 0.5}, {X: 1, Y: 1}, {X: 2, Y: 0}}, false)
	sm := g.AddSmooth(line, sp, line.B, sp.Points[0])
	if satisfied(sm) {
		t.Fatal("smooth should start unsatisfied for the bent spline")
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("smooth solve did not converge: residual %v", r.Residual)
	}
	if !satisfied(sm) {
		t.Error("spline should be driven to a smooth (G2) join with the line")
	}
	// The join must sit on the line end, and the spline must leave it horizontally.
	if d := sp.Points[0].Position().DistanceTo(line.B.Position()); d > 1e-6 {
		t.Errorf("spline start not coincident with the line end: d = %v", d)
	}
	if len(sm.Variables()) == 0 {
		t.Error("smooth exposed no variables")
	}
}
