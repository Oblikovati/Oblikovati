// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// pointOnLineFixture builds a rigid line of the given length on the X axis (both endpoints
// fixed) plus a free point constrained onto it, then returns the sketch. The point starts
// off the line. With the line fixed, a correctly-counted PointOnLine leaves exactly one DOF
// (the point slides along the line); a dropped row would leave two.
func pointOnLineFixture(length, perpOffset float64) *Sketch {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(math.Scalar(length), 0))
	line := s.Lines().Add(a, b)
	g.AddFix(a)
	g.AddFix(b)
	p := s.Points().Add(math.P2(math.Scalar(length/2), math.Scalar(perpOffset)))
	g.AddPointOnLine(p, line)
	return s
}

// TestPointOnLineResidualIsADistanceNotArea is acceptance criterion 1 of #1418: the
// PointOnLine residual is the perpendicular distance (length units), independent of the
// line's length — not the area |line|·|offset| it used to be.
func TestPointOnLineResidualIsADistanceNotArea(t *testing.T) {
	const perp = 0.1
	for _, length := range []float64{1, 1e-3, 100} {
		s := pointOnLineFixture(length, perp)
		c := s.GeometricConstraints().All()[2] // the PointOnLine constraint
		r := c.Residuals()[0]
		if stdmath.Abs(stdmath.Abs(r)-perp) > 1e-9 {
			t.Errorf("length %g: |residual| = %g, want the perpendicular distance %g (area-scaling not removed)",
				length, stdmath.Abs(r), perp)
		}
	}
}

// TestShortSegmentConstraintIsCounted is acceptance criterion 2 of #1418: a PointOnLine on
// a very short segment is still counted in the rank, so the DOF is correct. An area-scaled
// residual would give a Jacobian row below the rank tolerance and be dropped, inflating the
// reported DOF.
func TestShortSegmentConstraintIsCounted(t *testing.T) {
	s := pointOnLineFixture(1e-8, 0.05)
	if dof := s.DegreesOfFreedom(); dof != 1 {
		t.Errorf("short-segment PointOnLine: DOF = %d, want 1 (the constraint must be counted, not dropped)", dof)
	}
}

// TestLongShortParityClassification is acceptance criterion 3 of #1418: a geometrically
// identical sketch classifies the same whether its segments are long or short.
func TestLongShortParityClassification(t *testing.T) {
	long := pointOnLineFixture(10, 0.2)
	short := pointOnLineFixture(1e-7, 0.2)
	if l, s := long.DegreesOfFreedom(), short.DegreesOfFreedom(); l != s {
		t.Errorf("DOF parity broken: long sketch %d vs short sketch %d", l, s)
	}
	if l, s := long.AnalyzeConstraints().Status, short.AnalyzeConstraints().Status; l != s {
		t.Errorf("status parity broken: long %v vs short %v", l, s)
	}
}

// TestParallelCollinearShortSegmentOverConstrained covers the issue's over-constrained
// fixture: parallel AND collinear on short segments is redundant (collinear implies
// parallel), and must be reported over-constrained — which only holds if the short-segment
// rows are counted at their true (normalised) magnitude.
func TestParallelCollinearShortSegmentOverConstrained(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	const e = 1e-7 // short segments
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(e, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(e, 0))
	l2.A.SetPosition(math.P2(2*e, 0))
	l2.B.SetPosition(math.P2(3*e, 0))
	g.AddParallel(l1, l2)
	g.AddCollinear(l1, l2) // collinear ⊇ parallel: the parallel row is redundant
	if st := s.AnalyzeConstraints(); st.Redundant < 1 || st.Status != OverConstrained {
		t.Errorf("short parallel+collinear: %+v, want over-constrained with ≥1 redundant", st)
	}
}
