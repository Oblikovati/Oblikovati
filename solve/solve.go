// SPDX-License-Identifier: GPL-2.0-only

// Package solve is the one geometric constraint solver shared by the 2D sketch
// engine and the 3D assembly positioner (ADR-0011). It is the whole-system numerical
// core extracted from the sketch solver (ADR-0009): Newton/Gauss–Newton with
// Levenberg–Marquardt damping over a residual vector, warm-started from the current
// variable values, with DOF analysis falling out of the constraint Jacobian's rank.
//
// It is variable-kind-agnostic. A caller supplies residual sources and the scalar DOF
// variables they read, as []*math.Scalar — 2D points, 3D points, radii, and an
// assembly's rigid-body placement components (position + quaternion) all look the same
// to it. The graph-based decomposition into clusters is a future performance layer
// behind this same API; most systems are small enough that the whole-system solve is
// fine, which de-risks the schedule (ADR-0009/0011).
package solve

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Residual is one source of residual equations the solver drives to zero (a sketch or
// assembly constraint). Each value is zero when the relation is satisfied. The scalar
// variables it reads are supplied separately to [Solve], so the solver stays agnostic
// to how a residual source stores its variables.
type Residual interface {
	// Residuals returns the source's residual values; all zero ⇒ satisfied.
	Residuals() []float64
}

// Status summarizes a constraint system's state.
type Status uint8

const (
	// WellConstrained: zero DOF, no redundancy — a unique solution.
	WellConstrained Status = iota
	// UnderConstrained: free degrees of freedom remain.
	UnderConstrained
	// OverConstrained: redundant or conflicting constraints are present.
	OverConstrained
)

// String returns a stable name for diagnostics.
func (s Status) String() string {
	switch s {
	case WellConstrained:
		return "well-constrained"
	case UnderConstrained:
		return "under-constrained"
	default:
		return "over-constrained"
	}
}

// Options tune the numerical solve.
type Options struct {
	MaxIterations int
	Tolerance     float64
}

func (o Options) withDefaults() Options {
	if o.MaxIterations <= 0 {
		o.MaxIterations = 100
	}
	if o.Tolerance <= 0 {
		o.Tolerance = 1e-10
	}
	return o
}

// DOFAnalysis reports the constraint system's structure, derived from the Jacobian
// rank: free degrees of freedom and redundant (linearly dependent) constraints.
type DOFAnalysis struct {
	Variables int
	Equations int
	Rank      int
	DOF       int // Variables − Rank: remaining free degrees of freedom
	Redundant int // Equations − Rank: linearly dependent constraint rows
	Status    Status
}

// Result is the outcome of a solve.
type Result struct {
	Converged  bool
	Iterations int
	Residual   float64 // final max |residual|
	DOFAnalysis
}

// Solve drives the residuals to zero over the given variables, mutating them in place
// (warm-started from their current values), and reports the result with DOF analysis.
// It never hangs or returns NaN: it caps iterations and reports non-convergence as a
// result the caller turns into health (parametric-cad §2).
//
//	r := solve.Solve(residuals, vars, solve.Options{})
//	if !r.Converged { /* mark the system sick */ }
func Solve(res []Residual, vars []*math.Scalar, opts Options) Result {
	opts = opts.withDefaults()
	r := evalResiduals(res)
	lambda := 1e-3
	iter := 0
	for ; iter < opts.MaxIterations; iter++ {
		if infNorm(r) < opts.Tolerance || len(vars) == 0 {
			break
		}
		next, ok := newtonStep(res, vars, r, &lambda)
		if !ok {
			break // stuck at a singular/non-improving point
		}
		r = next
	}
	return Result{
		Converged:   infNorm(r) < opts.Tolerance,
		Iterations:  iter,
		Residual:    infNorm(r),
		DOFAnalysis: AnalyzeDOF(res, vars),
	}
}

// newtonStep performs one damped Gauss–Newton update, adjusting lambda. It accepts the
// step only if it reduces the residual norm; otherwise it increases damping and retries
// a few times. It returns the new residual vector and whether it improved.
func newtonStep(res []Residual, vars []*math.Scalar, r []float64, lambda *float64) ([]float64, bool) {
	j := Jacobian(res, vars)
	jtj, jtr := normalEquations(j, r)
	current := infNorm(r)
	for try := 0; try < 8; try++ {
		delta, ok := solveLinear(addDiagonal(jtj, *lambda), negated(jtr))
		if ok {
			snapshot := readVars(vars)
			applyDelta(vars, delta)
			trial := evalResiduals(res)
			if infNorm(trial) < current {
				*lambda = stdmath.Max(*lambda/10, 1e-12)
				return trial, true
			}
			writeVars(vars, snapshot)
		}
		*lambda *= 10
	}
	return r, false
}

// AnalyzeDOF computes the DOF/redundancy structure from the Jacobian rank.
func AnalyzeDOF(res []Residual, vars []*math.Scalar) DOFAnalysis {
	n := len(vars)
	j := Jacobian(res, vars)
	m := len(j)
	rank := matrixRank(j, 1e-7)
	a := DOFAnalysis{Variables: n, Equations: m, Rank: rank, DOF: n - rank, Redundant: m - rank}
	switch {
	case a.Redundant > 0:
		a.Status = OverConstrained
	case a.DOF > 0:
		a.Status = UnderConstrained
	default:
		a.Status = WellConstrained
	}
	return a
}

// evalResiduals concatenates every source's residuals.
func evalResiduals(res []Residual) []float64 {
	var out []float64
	for _, c := range res {
		out = append(out, c.Residuals()...)
	}
	return out
}

// Jacobian builds the m×n residual Jacobian by central finite differences (h=1e-7). It
// is exported so a caller that does its own DOF/mobility analysis can reuse it.
func Jacobian(res []Residual, vars []*math.Scalar) [][]float64 {
	const h = 1e-7
	m := len(evalResiduals(res))
	j := make([][]float64, m)
	for i := range j {
		j[i] = make([]float64, len(vars))
	}
	for col, v := range vars {
		orig := *v
		*v = orig + h
		rp := evalResiduals(res)
		*v = orig - h
		rm := evalResiduals(res)
		*v = orig
		for i := 0; i < m; i++ {
			j[i][col] = (rp[i] - rm[i]) / (2 * h)
		}
	}
	return j
}

// normalEquations returns JᵀJ (n×n) and Jᵀr (n) for the Gauss–Newton step.
func normalEquations(j [][]float64, r []float64) ([][]float64, []float64) {
	m, n := len(j), 0
	if m > 0 {
		n = len(j[0])
	}
	jtj := make([][]float64, n)
	jtr := make([]float64, n)
	for a := 0; a < n; a++ {
		jtj[a] = make([]float64, n)
		for i := 0; i < m; i++ {
			jtr[a] += j[i][a] * r[i]
			for b := 0; b < n; b++ {
				jtj[a][b] += j[i][a] * j[i][b]
			}
		}
	}
	return jtj, jtr
}

// addDiagonal returns a copy of jtj with lambda added to its diagonal (the
// Levenberg–Marquardt damping that keeps the system solvable near singularities).
func addDiagonal(jtj [][]float64, lambda float64) [][]float64 {
	n := len(jtj)
	out := make([][]float64, n)
	for i := range out {
		out[i] = append([]float64(nil), jtj[i]...)
		out[i][i] += lambda
	}
	return out
}

func negated(v []float64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = -x
	}
	return out
}

func readVars(vars []*math.Scalar) []float64 {
	out := make([]float64, len(vars))
	for i, v := range vars {
		out[i] = *v
	}
	return out
}

func writeVars(vars []*math.Scalar, values []float64) {
	for i, v := range vars {
		*v = values[i]
	}
}

func applyDelta(vars []*math.Scalar, delta []float64) {
	for i, v := range vars {
		*v += delta[i]
	}
}
