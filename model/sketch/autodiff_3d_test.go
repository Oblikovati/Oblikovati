// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestConstraint3DPartialsMatchFiniteDifference exercises every analytic 3D constraint at
// a generic off-axis configuration and checks its Jacobian block against central
// differences (#1417). Bend3D and the join constraints are covered separately because
// their residual shape depends on shared-endpoint identity.
func TestConstraint3DPartialsMatchFiniteDifference(t *testing.T) {
	s := NewSketches3D().Add()

	a := s.AddPoint3D(math.P3(1.3, 2.1, -0.6))
	b := s.AddPoint3D(math.P3(-0.7, 3.4, 1.2))
	c := s.AddPoint3D(math.P3(2.2, -1.1, 0.8))
	l1 := s.AddLine3D(math.P3(0.2, 0.5, 0.1), math.P3(2.6, 1.9, -0.4))
	l2 := s.AddLine3D(math.P3(-1.1, 0.3, 0.7), math.P3(1.7, 2.8, 1.3))
	zUp, _ := math.NewUnitVector3(0, 0, 1)
	c1 := s.AddCircle3D(math.P3(0, 0, 0), zUp, 2.5)
	c2 := s.AddCircle3D(math.P3(5, 0, 0), zUp, 3.1)

	cases := []struct {
		name string
		c    differentiableConstraint
	}{
		{"coincident3d", NewCoincident3D(a, b)},
		{"collinear3d", NewCollinear3D(a, b, c)},
		{"concentric3d", NewConcentric3D(a, b)},
		{"equal3d", mustEqual3D(t, c1, c2)},
		{"parallel3d", NewParallel3D(l1, l2)},
		{"perpendicular3d", NewPerpendicular3D(l1, l2)},
		{"midpoint3d", NewMidpoint3D(a, l1)},
		{"ground3d", NewGround3D(a)},
		{"parallel-to-axis3d", NewParallelToYAxis3D(l1)},
		{"parallel-to-plane3d", NewParallelToXZPlane3D(l1)},
	}
	for _, tc := range cases {
		assertAnalyticMatchesFD(t, tc.name, tc.c)
	}
}

// TestSmoothConstraint3DPartialsMatchFiniteDifference covers the 3D tangent/smooth joins
// over line/arc/spline dual frames (including the spline circumcircle curvature in 3D).
func TestSmoothConstraint3DPartialsMatchFiniteDifference(t *testing.T) {
	s := NewSketches3D().Add()
	line := s.AddLine3D(math.P3(0, 0, 0), math.P3(1.4, 0.3, 0.2))
	arc := s.AddArc3D(math.P3(2, 1, 0.5), math.P3(3.1, 1.2, 0.4), math.P3(2.2, 2.3, 1.1), true)
	sp := s.AddSpline3D([]math.Point3{
		{X: 0.2, Y: 0.1, Z: 0}, {X: 1.1, Y: 0.9, Z: 0.3}, {X: 2.3, Y: 0.7, Z: -0.2}, {X: 3.0, Y: 1.8, Z: 0.5},
	}, false, false)

	cases := []struct {
		name string
		c    differentiableConstraint
	}{
		{"tangent3d-line-arc", NewTangent3D(line, arc, line.B, arc.Start)},
		{"smooth3d-line-arc", NewSmooth3D(line, arc, line.B, arc.Start)},
		{"smooth3d-arc-spline", NewSmooth3D(arc, sp, arc.End, sp.Points[0])},
		{"smooth3d-spline-line", NewSmooth3D(sp, line, sp.Points[len(sp.Points)-1], line.A)},
	}
	for _, tc := range cases {
		assertAnalyticMatchesFD(t, tc.name, tc.c)
	}
}

// TestBend3DGlobalJacobian checks the bend constraint's assembled (global) Jacobian. The
// bend shares its arc endpoints with the trimmed lines, so the shared points appear twice
// in its Variables() — only the assembled column (scatterRow accumulation) is well
// defined, so it is compared against a global finite-difference Jacobian (#1417).
func TestBend3DGlobalJacobian(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(math.P3(0, 0, 0), math.P3(4, 0, 0))
	l2 := s.AddLine3D(math.P3(4, 0, 0), math.P3(4, 4, 0))
	arc, err := s.AddBend3D(l1, l2, 1.0)
	if err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}
	// Nudge the arc center off its solved spot so the tangency/radius residuals are
	// nonzero and every variable genuinely participates.
	arc.Center.SetPosition(math.P3(3.2, 0.9, 0.1))

	cons, universe := s.Constraints(), s.variables()
	got := jacobian(cons, universe)
	want := globalFDJacobian(cons, universe)
	for i := range want {
		for j := range want[i] {
			if stdmath.Abs(got[i][j]-want[i][j]) > 1e-5 {
				t.Errorf("bend3d global J[%d][%d] = %.9f, FD says %.9f", i, j, got[i][j], want[i][j])
			}
		}
	}
}

// TestDimension3DPartialsMatchFiniteDifference checks the analytic 3D dimensions. The
// spline-length dimension is excluded: its measure is a black-box NURBS sampler with no
// closed-form derivative, so it finite-differences by design (#1417).
func TestDimension3DPartialsMatchFiniteDifference(t *testing.T) {
	s := NewSketches3D().Add()
	d := s.DimensionConstraints3D()

	a := s.AddPoint3D(math.P3(1.3, 2.1, -0.6))
	b := s.AddPoint3D(math.P3(-0.7, 3.4, 1.2))
	l1 := s.AddLine3D(math.P3(0.2, 0.5, 0.1), math.P3(2.6, 1.9, -0.4))
	l2 := s.AddLine3D(math.P3(-1.1, 0.3, 0.7), math.P3(1.7, 2.8, 1.3))
	zAxis, _ := math.NewUnitVector3(0, 0, 1)
	circ := s.AddCircle3D(math.P3(0, 0, 0), zAxis, 2.0)

	mk := func(c *DimensionConstraint3D, err error) differentiableConstraint {
		if err != nil {
			t.Fatalf("3D dimension factory: %v", err)
		}
		return c
	}
	cases := []struct {
		name string
		c    differentiableConstraint
	}{
		{"distance3d", mk(d.AddDistance(a, b, "5 cm"))},
		{"line-length3d", mk(d.AddLineLength(l1, "3 cm"))},
		{"radius3d", mk(d.AddRadius(circ, "2 cm"))},
		{"point-plane3d", mk(d.AddPointPlaneDistance(a, math.V3(0, 0, 1), "1 cm"))},
		{"two-line-angle3d", mk(d.AddTwoLineAngle(l1, l2, "30 deg"))},
	}
	for _, tc := range cases {
		assertAnalyticMatchesFD(t, tc.name, tc.c)
	}
}

// mustEqual3D builds an Equal3D over two radius-bearing curves, failing the test
// on a refused operand (#1625).
func mustEqual3D(t *testing.T, e1, e2 Entity) *Equal3D {
	t.Helper()
	eq, err := NewEqual3D(e1, e2)
	if err != nil {
		t.Fatalf("NewEqual3D(%T, %T): %v", e1, e2, err)
	}
	return eq
}
