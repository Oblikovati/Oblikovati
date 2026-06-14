// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// The sketch constraint solver is now a thin adapter over the shared engine in
// oblikovati.org/solve (ADR-0011): the numeric core (damped Gauss–Newton + DOF
// analysis) was extracted so the 2D sketch and the 3D assembly positioner share one
// solver. This file keeps the sketch-facing names (Solve over []Constraint, the
// SolveStatus/SolveResult/DOFAnalysis types) so existing call sites are unchanged; the
// math lives in the solve package.

// SolveStatus summarizes a sketch's constraint state (alias of [solve.Status]).
type SolveStatus = solve.Status

const (
	// WellConstrained: zero DOF, no redundancy — a unique solution.
	WellConstrained = solve.WellConstrained
	// UnderConstrained: free degrees of freedom remain.
	UnderConstrained = solve.UnderConstrained
	// OverConstrained: redundant or conflicting constraints are present.
	OverConstrained = solve.OverConstrained
)

// Options tune the numerical solve (alias of [solve.Options]).
type Options = solve.Options

// DOFAnalysis reports the constraint system's structure (alias of [solve.DOFAnalysis]).
type DOFAnalysis = solve.DOFAnalysis

// SolveResult is the outcome of a solve (alias of [solve.Result]).
type SolveResult = solve.Result

// Solve drives the constraints to zero over the given variables, mutating them in place
// (warm-started), and reports the result with DOF analysis. It adapts the sketch's
// []Constraint onto the shared solver's residual interface.
func Solve(cons []Constraint, vars []*math.Scalar, opts Options) SolveResult {
	return solve.Solve(residualSources(cons), vars, opts)
}

// analyzeDOF computes the DOF/redundancy structure from the constraint Jacobian's rank.
func analyzeDOF(cons []Constraint, vars []*math.Scalar) DOFAnalysis {
	return solve.AnalyzeDOF(residualSources(cons), vars)
}

// jacobian builds the residual Jacobian for the constraints over vars (consumed by the
// mobility analysis in moveable_status.go).
func jacobian(cons []Constraint, vars []*math.Scalar) [][]float64 {
	return solve.Jacobian(residualSources(cons), vars)
}

// residualSources views the sketch constraints as the shared solver's residual sources;
// [Constraint] already exposes Residuals, so each one satisfies [solve.Residual].
func residualSources(cons []Constraint) []solve.Residual {
	out := make([]solve.Residual, len(cons))
	for i, c := range cons {
		out[i] = c
	}
	return out
}
