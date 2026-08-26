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
// adjusting lambda by the Moré/Nielsen gain-ratio schedule (audit A13, #1609). The step solves
// the Marquardt-scaled LM system [Jₛ; √λ·D]·δ = [−rₛ; 0] by Householder QR — D = diag of the
// column norms, so damping is invariant to per-variable scaling (an ill-scaled sketch mixing
// 1e-2 and 1e3 dimensions no longer stalls) — and accepts on any residual decrease: λ shrinks
// by the gain-ratio factor max(1/3, 1−(2ρ−1)³) on accept and grows by a doubling ν on reject
// (Madsen/Nielsen/Tingleff 2004), terminating when λ overflows the trust budget instead of the
// old fixed ×10/8-try cap that declared solvable systems falsely stuck.
func dampedStep(res []Residual, vars []*math.Scalar, j [][]float64, r []float64, lambda *float64) ([]float64, bool) {
	current := sumSquares(r)
	d := columnNorms(j, len(vars))
	nu := 2.0
	for *lambda < lambdaMax {
		aug, rhs := augmentedLM(j, r, *lambda, d)
		if delta, ok := leastSquares(aug, rhs); ok {
			snapshot := readVars(vars)
			applyDelta(vars, delta)
			trial := evalResiduals(res)
			if next := sumSquares(trial); next < current {
				*lambda *= gainRatioShrink(current, next, delta, j, r)
				*lambda = stdmath.Max(*lambda, 1e-12) // tol:numeric — LM damping floor
				return trial, true
			}
			writeVars(vars, snapshot)
		}
		*lambda *= nu
		nu *= 2
	}
	return r, false
}

// lambdaMax is the damping magnitude at which LM gives up: the step is then ~1e15 times
// shorter than Gauss–Newton, indistinguishable from no progress in float64.
const lambdaMax = 1e15 // tol:numeric — LM trust-budget ceiling

// gainRatioShrink is the Moré/Nielsen accept-side damping factor max(1/3, 1−(2ρ−1)³), where
// the gain ratio ρ compares the achieved decrease against the linear model's prediction
// L(0)−L(δ) = δᵀ(λ·D²·δ − Jᵀr) (Madsen/Nielsen/Tingleff eq. 2.21, with Marquardt scaling).
func gainRatioShrink(current, next float64, delta []float64, j [][]float64, r []float64) float64 {
	predicted := predictedDecrease(delta, j, r)
	if predicted <= 0 {
		return 1.0 / 3
	}
	rho := (current - next) / predicted
	f := 1 - (2*rho-1)*(2*rho-1)*(2*rho-1)
	return stdmath.Max(1.0/3, f)
}

// predictedDecrease is the linear-model decrease ‖r‖² − ‖r + J·δ‖² for step δ.
func predictedDecrease(delta []float64, j [][]float64, r []float64) float64 {
	after := 0.0
	for i := range j {
		ri := r[i]
		for c, dc := range delta {
			ri += j[i][c] * dc
		}
		after += ri * ri
	}
	return sumSquares(r) - after
}

// columnNorms returns the Marquardt scaling diagonal: each variable's Jacobian column norm,
// floored at 1 so a variable no constraint currently touches still receives bounded damping.
func columnNorms(j [][]float64, n int) []float64 {
	d := make([]float64, n)
	for c := range n {
		s := 0.0
		for i := range j {
			s += j[i][c] * j[i][c]
		}
		d[c] = stdmath.Max(stdmath.Sqrt(s), 1)
	}
	return d
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
	rank := pivotedQRRank(rowNormalized(j)) // relative rank-revealing CPQR (see rankRelTol)
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

// finiteDiffJacobian builds the Jacobian by central differences with a per-variable RELATIVE
// step h = ∛ε·max(1, |x|) (Nocedal & Wright §8.1; cube root for central differences) — the
// fallback for residual sources that cannot supply analytic partials. The retired absolute
// h=1e-7 made partials vanish into round-off at large coordinates (a sketch translated to 1e5
// lost rank for no geometric reason, audit A13 #1609). It perturbs the live variables and
// restores them, so it must not be used on the interactive analysis path.
func finiteDiffJacobian(res []Residual, vars []*math.Scalar) [][]float64 {
	m := len(evalResiduals(res))
	j := make([][]float64, m)
	for i := range j {
		j[i] = make([]float64, len(vars))
	}
	for col, v := range vars {
		orig := *v
		h := math.Scalar(fdStep * stdmath.Max(1, stdmath.Abs(float64(orig))))
		*v = orig + h
		rp := evalResiduals(res)
		*v = orig - h
		rm := evalResiduals(res)
		*v = orig
		for i := range m {
			j[i][col] = (rp[i] - rm[i]) / float64(2*h)
		}
	}
	return j
}

// fdStep is the relative central-difference step coefficient ∛ε ≈ 6e-6: optimal for the
// O(h²) truncation vs ε/h round-off tradeoff of central differences (Nocedal & Wright).
const fdStep = 6.06e-6 // tol:numeric — cube root of float64 machine epsilon

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

// RedundantSources identifies WHICH residual sources are removable to fix an over-constrained
// system by solvespace's leave-one-out search (System::FindWhichToRemoveToFixJacobian; audit
// A13 #1609): a source is removable when dropping its rows leaves the rank unchanged — its
// equations were linearly dependent on the rest — so removing it restores DOF bookkeeping
// without freeing the sketch. Returns source indices in input order; empty when the system is
// not redundant. O(sources × rank cost), invoked on demand (a diagnosis, not the solve path).
//
//	for _, i := range solve.RedundantSources(residuals, vars) { flag(residuals[i]) }
func RedundantSources(res []Residual, vars []*math.Scalar) []int {
	j := Jacobian(res, vars)
	full := analyzeJacobian(j, len(vars))
	if full.Redundant == 0 {
		return nil
	}
	var out []int
	row := 0
	for i, src := range res {
		k := len(src.Residuals())
		rest := append(append([][]float64(nil), j[:row]...), j[row+k:]...)
		if pivotedQRRank(rowNormalized(rest)) == full.Rank {
			out = append(out, i)
		}
		row += k
	}
	return out
}
