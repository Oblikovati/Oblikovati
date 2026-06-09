// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The constraint solver (ADR-0009). This is the whole-system numerical core:
// Newton/Gauss–Newton with Levenberg–Marquardt damping over the residual vector,
// warm-started from the current variable values (so an edit re-solves in a few
// iterations). DOF analysis falls out of the constraint Jacobian's rank. The
// graph-based decomposition into clusters is a future performance/diagnostics layer
// behind this same API; most sketches are small enough that the whole-system solve
// is fine, which de-risks the schedule (ADR-0009). It is dimension-agnostic: 2D and
// 3D constraints both expose variables as []*math.Scalar.

// SolveStatus summarizes a sketch's constraint state.
type SolveStatus uint8

const (
	// WellConstrained: zero DOF, no redundancy — a unique solution.
	WellConstrained SolveStatus = iota
	// UnderConstrained: free degrees of freedom remain.
	UnderConstrained
	// OverConstrained: redundant or conflicting constraints are present.
	OverConstrained
)

// String returns a stable name for diagnostics.
func (s SolveStatus) String() string {
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
	Status    SolveStatus
}

// SolveResult is the outcome of a solve.
type SolveResult struct {
	Converged  bool
	Iterations int
	Residual   float64 // final max |residual|
	DOFAnalysis
}

// Solve drives the constraints to zero over the given variables, mutating them in
// place (warm-started from their current values), and reports the result with DOF
// analysis. It never hangs or returns NaN: it caps iterations and reports
// non-convergence as a result the caller turns into health (parametric-cad §2).
func Solve(cons []Constraint, vars []*math.Scalar, opts Options) SolveResult {
	opts = opts.withDefaults()
	res := evalResiduals(cons)
	lambda := 1e-3
	iter := 0
	for ; iter < opts.MaxIterations; iter++ {
		if infNorm(res) < opts.Tolerance || len(vars) == 0 {
			break
		}
		next, ok := newtonStep(cons, vars, res, &lambda)
		if !ok {
			break // stuck at a singular/non-improving point
		}
		res = next
	}
	converged := infNorm(res) < opts.Tolerance
	return SolveResult{
		Converged:   converged,
		Iterations:  iter,
		Residual:    infNorm(res),
		DOFAnalysis: analyzeDOF(cons, vars),
	}
}

// newtonStep performs one damped Gauss–Newton update, adjusting lambda. It accepts
// the step only if it reduces the residual norm; otherwise it increases damping and
// retries a few times. It returns the new residual vector and whether it improved.
func newtonStep(cons []Constraint, vars []*math.Scalar, res []float64, lambda *float64) ([]float64, bool) {
	j := jacobian(cons, vars)
	jtj, jtr := normalEquations(j, res)
	current := infNorm(res)
	for try := 0; try < 8; try++ {
		delta, ok := solveLinear(addDiagonal(jtj, *lambda), negated(jtr))
		if ok {
			snapshot := readVars(vars)
			applyDelta(vars, delta)
			trial := evalResiduals(cons)
			if infNorm(trial) < current {
				*lambda = stdmath.Max(*lambda/10, 1e-12)
				return trial, true
			}
			writeVars(vars, snapshot)
		}
		*lambda *= 10
	}
	return res, false
}

// analyzeDOF computes the DOF/redundancy structure from the Jacobian rank.
func analyzeDOF(cons []Constraint, vars []*math.Scalar) DOFAnalysis {
	n := len(vars)
	j := jacobian(cons, vars)
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

// evalResiduals concatenates every constraint's residuals.
func evalResiduals(cons []Constraint) []float64 {
	var out []float64
	for _, c := range cons {
		out = append(out, c.Residuals()...)
	}
	return out
}

// jacobian builds the m×n residual Jacobian by central finite differences.
func jacobian(cons []Constraint, vars []*math.Scalar) [][]float64 {
	const h = 1e-7
	m := len(evalResiduals(cons))
	j := make([][]float64, m)
	for i := range j {
		j[i] = make([]float64, len(vars))
	}
	for col, v := range vars {
		orig := *v
		*v = orig + h
		rp := evalResiduals(cons)
		*v = orig - h
		rm := evalResiduals(cons)
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
