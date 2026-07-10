// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// offsetSplineFixture builds a parent spline and an offset of it at the given signed distance.
func offsetSplineFixture(s *Sketch, dist float64) *OffsetSpline {
	parent := s.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 1), math.P2(2, 0)}, false)
	return s.OffsetSplines().Add(parent, dist)
}

// TestOffsetSplineDistIsSolverDOF: adding an offset spline grows the DOF universe by exactly one —
// its offset distance is a solver variable, so an offset-spline dimension can drive it (#1874).
func TestOffsetSplineDistIsSolverDOF(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	parent := s.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 1), math.P2(2, 0)}, false)
	before := s.AnalyzeConstraints().Variables
	s.OffsetSplines().Add(parent, 2)
	if after := s.AnalyzeConstraints().Variables; after != before+1 {
		t.Errorf("offset spline changed the variable count by %d, want +1", after-before)
	}
}

// TestAddOffsetSplineDimDrivesDistance drives the offset to a 5 mm target and checks the solved
// distance is 0.5 cm on the same (positive) side it was created (#1874).
func TestAddOffsetSplineDimDrivesDistance(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	off := offsetSplineFixture(s, 2) // starts at +2 cm

	if _, err := s.DimensionConstraints().AddOffsetSplineDim(off, "5 mm"); err != nil {
		t.Fatalf("AddOffsetSplineDim: %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual=%v", r.Residual)
	}
	if stdmath.Abs(float64(off.Dist)-0.5) > 1e-6 {
		t.Errorf("solved offset = %v cm, want +0.5 (magnitude 5 mm, side preserved)", off.Dist)
	}
}

// TestOffsetSplineDimPreservesNegativeSide: an offset created on the negative side keeps that side
// when driven — the measure is the magnitude, so the sign is untouched (#1874).
func TestOffsetSplineDimPreservesNegativeSide(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	off := offsetSplineFixture(s, -2)

	if _, err := s.DimensionConstraints().AddOffsetSplineDim(off, "5 mm"); err != nil {
		t.Fatalf("AddOffsetSplineDim: %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual=%v", r.Residual)
	}
	if stdmath.Abs(float64(off.Dist)+0.5) > 1e-6 {
		t.Errorf("solved offset = %v cm, want -0.5 (negative side preserved)", off.Dist)
	}
}

// TestOffsetSplineDimSurvivesRoundTrip serializes an offset-spline dimension and restores it (#1874).
func TestOffsetSplineDimSurvivesRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	off := offsetSplineFixture(s, 1.5)
	if _, err := s.DimensionConstraints().AddOffsetSplineDim(off, "1 cm"); err != nil {
		t.Fatalf("AddOffsetSplineDim: %v", err)
	}

	out := roundTrip(t, sc)
	dims := out.DimensionConstraints().All()
	if len(dims) != 1 || dims[0].Kind() != OffsetSplineDim {
		t.Fatalf("restored dims = %+v, want one offsetSplineDim", dims)
	}
	if len(dims[0].Refs()) != 1 {
		t.Fatalf("restored offsetSplineDim refs = %d, want 1", len(dims[0].Refs()))
	}
	if _, ok := dims[0].Refs()[0].(*OffsetSpline); !ok {
		t.Errorf("restored offsetSplineDim target = %T, want *OffsetSpline", dims[0].Refs()[0])
	}
}
