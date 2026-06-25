// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// residualOnly wraps a constraint as a plain (non-Differentiable) residual source, so
// solve.Solve falls back to finite differencing — the "before #1417" behaviour, used to
// contrast against the analytic path on the same fixture.
type residualOnly struct{ c Constraint }

func (r residualOnly) Residuals() []float64 { return r.c.Residuals() }

func snapshotVars(vars []*math.Scalar) []float64 {
	out := make([]float64, len(vars))
	for i, v := range vars {
		out[i] = float64(*v)
	}
	return out
}

// maxResidual returns the largest |residual| across all constraints — the inf-norm the
// solver drives to zero, used to measure starting-configuration error.
func maxResidual(cons []Constraint) float64 {
	m := 0.0
	for _, c := range cons {
		for _, r := range c.Residuals() {
			if a := stdmath.Abs(r); a > m {
				m = a
			}
		}
	}
	return m
}

// constraintResiduals views constraints as the solver's residual sources (each is
// Differentiable, so the solver assembles the Jacobian analytically).
func constraintResiduals(cons []Constraint) []solve.Residual {
	out := make([]solve.Residual, len(cons))
	for i, c := range cons {
		out[i] = c
	}
	return out
}

// fdResiduals wraps every constraint as a non-Differentiable source, forcing the solver's
// finite-difference Jacobian fallback.
func fdResiduals(cons []Constraint) []solve.Residual {
	out := make([]solve.Residual, len(cons))
	for i, c := range cons {
		out[i] = residualOnly{c}
	}
	return out
}

// TestNearTangentSketchConvergesAnalytically is acceptance criterion 3 of #1417: a point
// pulled onto BOTH a circle and a line that are nearly tangent near it — the near-singular
// Jacobian where Gauss–Newton is most sensitive to derivative error — converges cleanly
// with the exact analytic Jacobian.
func TestNearTangentSketchConvergesAnalytically(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()

	center := s.Points().Add(math.P2(0, 0))
	g.AddFix(center)
	circle := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	circle.Radius = 1
	// A near-horizontal line just below the circle's apex, almost tangent at (0,1).
	la := s.Points().Add(math.P2(-3, 0.999))
	lb := s.Points().Add(math.P2(3, 1.0006))
	line := s.Lines().AddByTwoPoints(la.Position(), lb.Position())
	line.A, line.B = la, lb
	g.AddFix(la)
	g.AddFix(lb)
	p := s.Points().Add(math.P2(0.05, 0.97)) // start near, but off, the contact
	g.AddPointOnCircle(p, circle)
	g.AddPointOnLine(p, line)

	r := solve.Solve(constraintResiduals(s.Constraints()), s.variables(), solve.Options{MaxIterations: 200})
	if !r.Converged || r.Residual > 1e-9 {
		t.Fatalf("near-tangent analytic solve: converged=%v residual=%g, want converged with residual < 1e-9", r.Converged, r.Residual)
	}
}

// TestFarFromOriginAnalyticBeatsFiniteDifference is the deterministic "where FD stalled"
// evidence: far from the origin the solver's fixed absolute finite-difference step
// (h=1e-7) underflows the coordinate ULP, so the FD Jacobian collapses to noise and FD
// makes NO progress — while the exact analytic Jacobian still drives the residual down by
// orders of magnitude (the absolute floor it reaches is the coordinate-precision limit of
// #1399, orthogonal to the Jacobian quality demonstrated here).
func TestFarFromOriginAnalyticBeatsFiniteDifference(t *testing.T) {
	const off = 1e10
	build := func() ([]Constraint, []*math.Scalar) {
		s := NewSketches().Add(XYPlane())
		g := s.GeometricConstraints()
		a := s.Points().Add(math.P2(off, off))
		b := s.Points().Add(math.P2(off, off))
		b.SetPosition(math.P2(off+0.3, off+0.1)) // start ~0.32 apart; target 5
		g.AddFix(a)
		if _, err := s.DimensionConstraints().AddDistance(a, b, "5 cm"); err != nil {
			t.Fatalf("AddDistance: %v", err)
		}
		return s.Constraints(), s.variables()
	}

	cons, _ := build()
	initial := maxResidual(cons) // residual at the starting configuration, before any solve

	cons, vars := build()
	analytic := solve.Solve(constraintResiduals(cons), vars, solve.Options{MaxIterations: 200})

	cons, vars = build()
	fd := solve.Solve(fdResiduals(cons), vars, solve.Options{MaxIterations: 200})

	// FD's step underflows: it barely moves off the initial residual, so it never reaches
	// the (now relative, #1420) tolerance — it reports non-convergence.
	if fd.Converged || fd.Residual < initial*0.5 {
		t.Errorf("finite-difference solve made unexpected progress (converged=%v residual=%g vs initial %g) — it was expected to stall at this scale", fd.Converged, fd.Residual, initial)
	}
	// The exact analytic Jacobian drives the system to the relative tolerance and converges
	// — the relative tolerance (relTol·scale) is the achievable precision at scale 1e10
	// (below it the coordinate ULP dominates, #1399).
	if !analytic.Converged {
		t.Errorf("analytic solve did not converge at scale: residual %g", analytic.Residual)
	}
	if analytic.Residual >= fd.Residual {
		t.Errorf("analytic residual %g not better than stalled FD %g", analytic.Residual, fd.Residual)
	}
	t.Logf("initial %.3e → analytic %.3e conv=%v (it=%d) vs FD %.3e conv=%v (it=%d)",
		initial, analytic.Residual, analytic.Converged, analytic.Iterations, fd.Residual, fd.Converged, fd.Iterations)
}

// TestWellConditionedAnalyticIsCorrect is the control: on a well-conditioned fixture the
// analytic solve converges to the right geometry, confirming the near-tangent and
// far-origin gaps are conditioning/scale effects, not a wrong analytic derivative.
func TestWellConditionedAnalyticIsCorrect(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(3, 0.2))
	g.AddFix(a)
	if _, err := s.DimensionConstraints().AddDistance(a, b, "5 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	r := solve.Solve(constraintResiduals(s.Constraints()), s.variables(), solve.Options{})
	if !r.Converged || r.Residual > 1e-9 {
		t.Fatalf("well-conditioned analytic solve: converged=%v residual=%g", r.Converged, r.Residual)
	}
	if d := float64(a.Position().DistanceTo(b.Position())); stdmath.Abs(d-5) > 1e-6 {
		t.Errorf("distance %v after solve, want 5", d)
	}
}
