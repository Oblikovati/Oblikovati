// SPDX-License-Identifier: GPL-2.0-only

package solve

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Residual nondimensionalisation and scaling for a robust, scale-invariant solve
// (Oblikovati/Oblikovati#1420). The solver sees a mix of residual units — distances
// (length), and the normalised orientation residuals of #1418 (sine/cosine, dimensionless)
// — over geometry that may live at any model scale (µm to m). Left raw, whichever
// constraint carries the largest units dominates convergence and the LM damping, and
// absolute tolerances are wrong across scales.
//
// The fix is to weight each residual by 1/‖Jᵢ‖: row i then measures the displacement (in
// variable units) needed to satisfy constraint i — a COMMON length scale for every
// constraint regardless of its native units. A relative tolerance tied to the model's
// coordinate magnitude (ADR-0042 working scale) then makes convergence and rank/DOF
// classification identical across a scale sweep.

// modelScale returns the characteristic coordinate magnitude of the variable set — the
// largest absolute value — used as the length scale for the relative tolerance. It floors
// at 1e-12 so an all-at-origin (degenerate) system does not yield a zero threshold.
func modelScale(vars []*math.Scalar) float64 {
	m := 0.0
	for _, v := range vars {
		if a := stdmath.Abs(float64(*v)); a > m {
			m = a
		}
	}
	if m < 1e-12 {
		return 1
	}
	return m
}

// rowWeights returns 1/‖Jᵢ‖₂ for each Jacobian row — the factor that nondimensionalises
// residual i to a variable-space displacement. A zero row (a constraint touching no active
// variable) gets weight 1, leaving its residual unchanged.
func rowWeights(j [][]float64) []float64 {
	w := make([]float64, len(j))
	for i, row := range j {
		n := 0.0
		for _, x := range row {
			n += x * x
		}
		if n = stdmath.Sqrt(n); n < 1e-12 {
			w[i] = 1
			continue
		}
		w[i] = 1 / n
	}
	return w
}

// rowNormalized returns the Jacobian with every row scaled to unit norm — the matrix the
// rank/DOF analysis operates on, so the rank tolerance is relative (scale-invariant) and a
// short-segment or large-scale constraint row is never spuriously dropped. Row scaling
// preserves rank, so this only improves the numerical classification.
func rowNormalized(j [][]float64) [][]float64 {
	w := rowWeights(j)
	out := make([][]float64, len(j))
	for i, row := range j {
		nr := make([]float64, len(row))
		for k, x := range row {
			nr[k] = x * w[i]
		}
		out[i] = nr
	}
	return out
}

// augmentedLM builds the Levenberg–Marquardt least-squares system [J; √λ·I]·δ = [−r; 0],
// whose QR solution is the damped Gauss–Newton step WITHOUT forming the normal equations
// JᵀJ — so the condition number of an already-delicate J is not squared (#1420). The √λ·I
// block is the trust-region damping that keeps the augmented matrix full column rank near
// singularities. The step uses the RAW Jacobian (not row-weighted): weighting by 1/‖Jᵢ‖
// would down-weight a stiff constraint — e.g. a curvature (G2) residual — exactly where it
// needs the most attention, stalling the solve. Nondimensionalisation is applied to the
// convergence test, rank and conflict report instead (the observable, scale-sensitive
// parts), not to the step direction.
func augmentedLM(j [][]float64, r []float64, lambda float64, n int) ([][]float64, []float64) {
	m := len(j)
	sqrtL := stdmath.Sqrt(lambda)
	aug := make([][]float64, m+n)
	rhs := make([]float64, m+n)
	for i := 0; i < m; i++ {
		aug[i] = j[i]
		rhs[i] = -r[i]
	}
	for d := 0; d < n; d++ {
		row := make([]float64, n)
		row[d] = sqrtL
		aug[m+d] = row
	}
	return aug, rhs
}

// weightedInfNorm returns max|rᵢ·wᵢ| — the nondimensionalised residual norm the
// convergence test drives to zero, comparable across mixed-unit constraints and scales.
func weightedInfNorm(r, w []float64) float64 {
	m := 0.0
	for i, v := range r {
		if a := stdmath.Abs(v * w[i]); a > m {
			m = a
		}
	}
	return m
}

// sumSquares returns Σrᵢ² — the objective the Levenberg–Marquardt step minimises, used for
// step acceptance (a step is taken only when it decreases this). Acceptance on the 2-norm
// (not the inf-norm) matches the quadratic model the LM step is built from and avoids
// rejecting good steps that trade one constraint's error for a net reduction.
func sumSquares(r []float64) float64 {
	s := 0.0
	for _, v := range r {
		s += v * v
	}
	return s
}
