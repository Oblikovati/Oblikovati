// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// differentiableConstraint is a constraint that supplies its own analytic partials —
// every geometric and dimensional constraint, post-#1417.
type differentiableConstraint interface {
	Constraint
	Partials() [][]float64
}

// fdPartials approximates a constraint's Jacobian block by central differences over its
// own variables — the trusted oracle the analytic Partials are checked against. Finite
// differencing is fine HERE (a test); the point of #1417 is that it is gone from the
// solve path.
func fdPartials(c differentiableConstraint) [][]float64 {
	const h = 1e-6
	vars := c.Variables()
	m := len(c.Residuals())
	jac := make([][]float64, m)
	for i := range jac {
		jac[i] = make([]float64, len(vars))
	}
	for j, v := range vars {
		orig := *v
		*v = orig + h
		rp := append([]float64(nil), c.Residuals()...)
		*v = orig - h
		rm := append([]float64(nil), c.Residuals()...)
		*v = orig
		for i := 0; i < m; i++ {
			jac[i][j] = (rp[i] - rm[i]) / (2 * h)
		}
	}
	return jac
}

// assertAnalyticMatchesFD checks every entry of the analytic block against the
// finite-difference oracle, and that the block is shaped to the residuals × variables.
func assertAnalyticMatchesFD(t *testing.T, name string, c differentiableConstraint) {
	t.Helper()
	got := c.Partials()
	want := fdPartials(c)
	if len(got) != len(c.Residuals()) {
		t.Fatalf("%s: %d partial rows, want %d residuals", name, len(got), len(c.Residuals()))
	}
	for i := range want {
		if len(got[i]) != len(c.Variables()) {
			t.Fatalf("%s: row %d width %d, want %d variables", name, i, len(got[i]), len(c.Variables()))
		}
		for j := range want[i] {
			if stdmath.Abs(got[i][j]-want[i][j]) > 1e-5 {
				t.Errorf("%s: ∂r%d/∂v%d = %.9f, FD says %.9f", name, i, j, got[i][j], want[i][j])
			}
		}
	}
}

// TestGeometricConstraintPartialsMatchFiniteDifference exercises every 2D geometric
// constraint at a generic (non-degenerate, off-axis) configuration, where the residual
// genuinely depends on every variable, and checks its analytic Jacobian against central
// differences (#1417: "agreement within tol on each constraint type").
func TestGeometricConstraintPartialsMatchFiniteDifference(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	p := s.Points().Add(math.P2(1.3, 2.1))
	q := s.Points().Add(math.P2(-0.7, 3.4))
	l1 := s.Lines().AddByTwoPoints(math.P2(0.2, 0.5), math.P2(2.6, 1.9))
	l2 := s.Lines().AddByTwoPoints(math.P2(-1.1, 0.3), math.P2(1.7, 2.8))
	c1 := s.Circles().AddByCenterRadius(math.P2(0.9, 1.4), 2.3)
	c2 := s.Circles().AddByCenterRadius(math.P2(3.2, -0.6), 1.1)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0.4, 0.7), math.P2(2.1, 0.7), math.P2(0.4, 2.4), true)

	cases := []struct {
		name string
		c    differentiableConstraint
	}{
		{"coincident", g.AddCoincident(p, q)},
		{"point-on-line", g.AddPointOnLine(p, l1)},
		{"midpoint", g.AddMidpoint(p, l1)},
		{"point-on-circle", g.AddPointOnCircle(p, c1)},
		{"point-on-arc", g.AddPointOnCircle(q, arc)},
		{"horizontal", g.AddHorizontal(p, q)},
		{"vertical", g.AddVertical(p, q)},
		{"parallel", g.AddParallel(l1, l2)},
		{"perpendicular", g.AddPerpendicular(l1, l2)},
		{"collinear", g.AddCollinear(l1, l2)},
		{"concentric", g.AddConcentric(c1, c2)},
		{"equal-length", g.AddEqualLength(l1, l2)},
		{"equal-radius-circle", g.AddEqualRadius(c1, c2)},
		{"equal-radius-arc", g.AddEqualRadius(c1, arc)},
		{"tangent-line-circle", g.AddTangent(l1, c1)},
		{"tangent-line-arc", g.AddTangent(l2, arc)},
		{"circular-tangent", g.AddCircularTangent(c1, c2)},
		{"symmetry", g.AddSymmetry(p, q, l1)},
		{"fix", g.AddFix(p)},
	}
	for _, tc := range cases {
		assertAnalyticMatchesFD(t, tc.name, tc.c)
	}
}

// globalFDJacobian builds the sketch's Jacobian by central differences over the whole
// variable universe — the oracle for the assembled (global) analytic Jacobian, the only
// level at which a shared variable's column is well defined.
func globalFDJacobian(cons []Constraint, universe []*math.Scalar) [][]float64 {
	const h = 1e-6
	resAt := func() []float64 {
		var out []float64
		for _, c := range cons {
			out = append(out, c.Residuals()...)
		}
		return out
	}
	m := len(resAt())
	jac := make([][]float64, m)
	for i := range jac {
		jac[i] = make([]float64, len(universe))
	}
	for j, v := range universe {
		orig := *v
		*v = orig + h
		rp := resAt()
		*v = orig - h
		rm := resAt()
		*v = orig
		for i := 0; i < m; i++ {
			jac[i][j] = (rp[i] - rm[i]) / (2 * h)
		}
	}
	return jac
}

// TestEqualLengthSharedVertexGlobalJacobian covers what the per-constraint table misses:
// two lines sharing a vertex, so the shared point appears TWICE in one constraint's
// Variables(). Its two occurrences' partials must SUM into the shared global column
// (#1417 scatterRow accumulation). This is only well defined after assembly, so it is
// checked against a global finite-difference Jacobian.
func TestEqualLengthSharedVertexGlobalJacobian(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	shared := s.Points().Add(math.P2(1.0, 1.0))
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 0))
	l1.B, l2.A = shared, shared // l1 ends and l2 starts at the same point
	l1.A.SetPosition(math.P2(-1.3, 0.4))
	l2.B.SetPosition(math.P2(2.1, -0.7))
	s.GeometricConstraints().AddEqualLength(l1, l2)

	cons, universe := s.Constraints(), s.variables()
	got := jacobian(cons, universe)
	want := globalFDJacobian(cons, universe)
	for i := range want {
		for j := range want[i] {
			if stdmath.Abs(got[i][j]-want[i][j]) > 1e-5 {
				t.Errorf("global J[%d][%d] = %.9f, FD says %.9f", i, j, got[i][j], want[i][j])
			}
		}
	}
}

// TestSmoothConstraintPartialsMatchFiniteDifference exercises the G2 smooth constraint's
// per-curve dual frames (line/arc/spline) at a generic configuration, where the spline
// curvature (circumcircle of three control points) genuinely depends on every variable.
func TestSmoothConstraintPartialsMatchFiniteDifference(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	line := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1.4, 0.3))
	arc := s.Arcs().AddByCenterStartEnd(math.P2(2, 1), math.P2(3.1, 1.2), math.P2(2.2, 2.3), true)
	sp := s.Splines().AddByControlPoints([]math.Point2{{X: 0.2, Y: 0.1}, {X: 1.1, Y: 0.9}, {X: 2.3, Y: 0.7}, {X: 3.0, Y: 1.8}}, false)

	cases := []struct {
		name string
		c    differentiableConstraint
	}{
		{"smooth-line-arc", g.AddSmooth(line, arc, line.B, arc.Start)},
		{"smooth-arc-spline", g.AddSmooth(arc, sp, arc.End, sp.Points[0])},
		{"smooth-spline-line", g.AddSmooth(sp, line, sp.Points[len(sp.Points)-1], line.A)},
	}
	for _, tc := range cases {
		assertAnalyticMatchesFD(t, tc.name, tc.c)
	}
}

// TestDimensionPartialsMatchFiniteDifference checks each 2D dimension kind's analytic
// Jacobian row against central differences at a generic configuration.
func TestDimensionPartialsMatchFiniteDifference(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	d := s.DimensionConstraints()

	p := s.Points().Add(math.P2(1.3, 2.1))
	q := s.Points().Add(math.P2(-0.7, 3.4))
	vtx := s.Points().Add(math.P2(0.5, 0.5))
	l1 := s.Lines().AddByTwoPoints(math.P2(0.2, 0.5), math.P2(2.6, 1.9))
	l2 := s.Lines().AddByTwoPoints(math.P2(-1.1, 0.3), math.P2(1.7, 2.8))
	c1 := s.Circles().AddByCenterRadius(math.P2(0.9, 1.4), 2.3)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0.4, 0.7), math.P2(2.1, 0.7), math.P2(0.4, 2.4), true)
	ell := s.Ellipses().Add(math.P2(1, 1), math.V2(1, 0), 3, 1.5)

	mk := func(c *DimensionConstraint, err error) *DimensionConstraint {
		if err != nil {
			t.Fatalf("dimension factory: %v", err)
		}
		return c
	}
	cases := []struct {
		name string
		c    differentiableConstraint
	}{
		{"distance", mk(d.AddDistance(p, q, "5 cm"))},
		{"radius-circle", mk(d.AddRadius(c1, "2 cm"))},
		{"radius-arc", mk(d.AddRadius(arc, "2 cm"))},
		{"diameter", mk(d.AddDiameter(c1, "4 cm"))},
		{"angle", mk(d.AddAngle(l1, l2, "30 deg"))},
		{"tangent-distance-near", mk(d.AddTangentDistance(l1, c1, false, "1 cm"))},
		{"tangent-distance-far", mk(d.AddTangentDistance(l2, c1, true, "1 cm"))},
		{"arc-length", mk(d.AddArcLength(arc, "3 cm"))},
		{"offset", mk(d.AddOffsetDim(p, l1, "1 cm"))},
		{"three-point-angle", mk(d.AddThreePointAngle(vtx, p, q, "45 deg"))},
		{"ellipse-radius", mk(d.AddEllipseRadius(ell, "3 cm"))},
	}
	for _, tc := range cases {
		assertAnalyticMatchesFD(t, tc.name, tc.c)
	}
}
