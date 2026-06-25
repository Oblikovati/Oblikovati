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
	"sort"

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

// Differentiable is a residual source that supplies its OWN exact partial derivatives,
// letting the solver assemble the Jacobian analytically — no finite differencing and,
// crucially, no perturbation of the live variables (Oblikovati/Oblikovati#1417). When
// every source is Differentiable the solver never finite-differences; a source that is
// not (an opaque residual closure, e.g. the assembly positioner) makes the whole solve
// fall back to central differences.
type Differentiable interface {
	Residual
	// Variables returns the scalar DOFs the partials are ordered by — the columns of
	// the Partials block, by pointer, so the solver can map them to global columns.
	Variables() []*math.Scalar
	// Partials returns ∂residualᵢ/∂varⱼ at the current point: one row per value of
	// Residuals() (same order), each row holding a partial for every variable of
	// Variables() (same order). It reads, never mutates, the live variables.
	Partials() [][]float64
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
		o.Tolerance = 1e-10 // tol:numeric — solver residual-norm convergence target (residual normalisation is #1420)
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
	// Jacobian is the residual Jacobian at the converged point, computed once at the
	// end of the solve and reused for the rank/DOF analysis here (#1417). Callers can
	// reuse it for mobility analysis instead of rebuilding it.
	Jacobian [][]float64
	// Conflicts lists, on a FAILED solve, the indices of the residual sources still
	// unsatisfied (their nondimensionalised residual exceeds the tolerance), most severe
	// first — the offending-constraint subset a caller surfaces for conflict diagnosis
	// (#1420). Empty on a converged solve.
	Conflicts []int
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
	scale := modelScale(vars)
	r := evalResiduals(res)
	lambda := 1e-3 // tol:numeric — Levenberg–Marquardt initial damping
	iter := 0
	j := Jacobian(res, vars)
	for ; iter < opts.MaxIterations; iter++ {
		if len(vars) == 0 || converged(r, j, scale, opts.Tolerance) {
			break
		}
		next, ok := dampedStep(res, vars, j, r, &lambda)
		if !ok {
			break // stuck at a singular/non-improving point
		}
		r = next
		j = Jacobian(res, vars) // at the new iterate, reused by the next convergence test and step
	}
	// j is the Jacobian at the final iterate; derive the DOF analysis from it (computed
	// once, #1417) and, on failure, the offending-constraint subset (#1420).
	conv := len(vars) == 0 || converged(r, j, scale, opts.Tolerance)
	result := Result{
		Converged:   conv,
		Iterations:  iter,
		Residual:    infNorm(r),
		DOFAnalysis: analyzeJacobian(j, len(vars)),
		Jacobian:    j,
	}
	if !conv {
		result.Conflicts = conflictingSources(res, j, scale, opts.Tolerance)
	}
	return result
}

// converged reports whether every constraint is satisfied to a RELATIVE tolerance: the
// nondimensionalised residual (max |rᵢ|/‖Jᵢ‖, a variable-space displacement) is within
// relTol·scale of zero. Tying the threshold to the model's coordinate scale makes
// convergence identical across a µm→m scale sweep and independent of mixed residual units
// (#1420).
func converged(r []float64, j [][]float64, scale, relTol float64) bool {
	return weightedInfNorm(r, rowWeights(j)) < relTol*scale
}

// dampedStep performs one damped Gauss–Newton update from the precomputed Jacobian j,
// adjusting lambda. The step solves the nondimensionalised LM system [Jₛ; √λ·I]·δ = [−rₛ; 0]
// by Householder QR — no normal-equations JᵀJ, so the condition number is not squared
// (#1420). It accepts the step only if it reduces the weighted residual norm, otherwise it
// raises the damping and retries. Returns the new residual vector and whether it improved.
func dampedStep(res []Residual, vars []*math.Scalar, j [][]float64, r []float64, lambda *float64) ([]float64, bool) {
	current := sumSquares(r)
	n := len(vars)
	for try := 0; try < 8; try++ {
		aug, rhs := augmentedLM(j, r, *lambda, n)
		delta, ok := leastSquares(aug, rhs)
		if ok {
			snapshot := readVars(vars)
			applyDelta(vars, delta)
			trial := evalResiduals(res)
			if sumSquares(trial) < current {
				*lambda = stdmath.Max(*lambda/10, 1e-12) // tol:numeric — LM damping floor
				return trial, true
			}
			writeVars(vars, snapshot)
		}
		*lambda *= 10
	}
	return r, false
}

// conflictingSources returns the residual sources still unsatisfied at a failed solve —
// those whose largest nondimensionalised residual exceeds the tolerance — most severe
// first, for UI conflict diagnosis (#1420). The residual rows are walked in source order,
// aligned with the Jacobian.
func conflictingSources(res []Residual, j [][]float64, scale, relTol float64) []int {
	w := rowWeights(j)
	full := evalResiduals(res)
	threshold := relTol * scale
	type offender struct {
		index int
		mag   float64
	}
	var bad []offender
	row := 0
	for s, src := range res {
		k := len(src.Residuals())
		worst := 0.0
		for i := row; i < row+k; i++ {
			if v := stdmath.Abs(full[i] * w[i]); v > worst {
				worst = v
			}
		}
		row += k
		if worst > threshold {
			bad = append(bad, offender{s, worst})
		}
	}
	sort.Slice(bad, func(a, b int) bool { return bad[a].mag > bad[b].mag })
	out := make([]int, len(bad))
	for i, o := range bad {
		out[i] = o.index
	}
	return out
}

// AnalyzeDOF computes the DOF/redundancy structure from the Jacobian rank. It is the
// entry point for analysis WITHOUT a solve (e.g. a hover-time DOF query); within a
// solve the Jacobian is computed once and [analyzeJacobian] is reused directly.
func AnalyzeDOF(res []Residual, vars []*math.Scalar) DOFAnalysis {
	return analyzeJacobian(Jacobian(res, vars), len(vars))
}

// analyzeJacobian derives the DOF/redundancy structure from an already-built Jacobian,
// so the matrix is computed once per solve and reused (#1417). It ranks the ROW-NORMALISED
// Jacobian so the tolerance is relative — a short-segment or large-scale constraint row is
// never spuriously dropped, giving scale-invariant DOF classification (#1420).
func analyzeJacobian(j [][]float64, n int) DOFAnalysis {
	m := len(j)
	rank := matrixRank(rowNormalized(j), 1e-7) // tol:numeric — relative rank tolerance (rows are unit-norm)
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

// Jacobian builds the m×n residual Jacobian over vars. When every residual source
// supplies its own exact partials it is assembled analytically — no finite differencing
// and no perturbation of the live variables (#1417). Otherwise it falls back to central
// differences (the legacy path for opaque residual closures, e.g. the assembly
// positioner). It is exported so a caller doing its own DOF/mobility analysis can reuse
// it.
func Jacobian(res []Residual, vars []*math.Scalar) [][]float64 {
	if j, ok := analyticJacobian(res, vars); ok {
		return j
	}
	return finiteDiffJacobian(res, vars)
}

// analyticJacobian assembles the Jacobian from each source's exact partials, mapping the
// source's own variables to global columns by pointer identity. It reports ok=false (so
// the caller finite-differences instead) the moment any source is not [Differentiable].
func analyticJacobian(res []Residual, vars []*math.Scalar) ([][]float64, bool) {
	col := make(map[*math.Scalar]int, len(vars))
	for i, v := range vars {
		col[v] = i
	}
	var rows [][]float64
	for _, src := range res {
		d, ok := src.(Differentiable)
		if !ok {
			return nil, false
		}
		dvars := d.Variables()
		for _, partials := range d.Partials() {
			rows = append(rows, scatterRow(partials, dvars, col, len(vars)))
		}
	}
	return rows, true
}

// scatterRow places a source's local partials into a full-width Jacobian row, keyed by
// each local variable's global column. Contributions ACCUMULATE: when a constraint lists
// the same variable twice (e.g. EqualLength on two lines sharing a vertex, where the
// shared point is both lines' endpoint), the column's true derivative is the sum of the
// partials w.r.t. each occurrence — matching the finite-difference path, which perturbs
// the shared column once and sees both. Variables outside the universe (not in col) are
// dropped, again matching finite differencing, which only perturbs universe columns.
func scatterRow(partials []float64, dvars []*math.Scalar, col map[*math.Scalar]int, width int) []float64 {
	row := make([]float64, width)
	for li, p := range partials {
		if c, in := col[dvars[li]]; in {
			row[c] += p
		}
	}
	return row
}

// finiteDiffJacobian builds the Jacobian by central differences (h=1e-7) — the fallback
// for residual sources that cannot supply analytic partials. It perturbs the live
// variables and restores them, so it must not be used on the interactive analysis path.
func finiteDiffJacobian(res []Residual, vars []*math.Scalar) [][]float64 {
	const h = 1e-7 // tol:numeric — finite-difference Jacobian step (analytic path is #1417)
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
