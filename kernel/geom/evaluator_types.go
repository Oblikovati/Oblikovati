// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/api/types"

// The evaluator result vocabulary (M01-F06, #603) is defined once in api/types
// (ADR-0018) and aliased here so kernel call sites read naturally.

// SolutionNature classifies the solution set of a closest-point query.
type SolutionNature = types.SolutionNature

const (
	// UnknownSolutionNature means no solution strategy could be determined.
	UnknownSolutionNature = types.UnknownSolutionNature
	// UniqueSolution means there is exactly one solution.
	UniqueSolution = types.UniqueSolution
	// DistinctlyManySolutions means finitely many, equally good solutions exist.
	DistinctlyManySolutions = types.DistinctlyManySolutions
	// InfinitelyManySolutions means a continuum of equally good solutions exists.
	InfinitelyManySolutions = types.InfinitelyManySolutions
	// NoSolution means no solution exists.
	NoSolution = types.NoSolution
)

// ParamAnomaly reports one parameter direction's irregularities.
type ParamAnomaly = types.ParamAnomaly

// ContinuityInfinite is the continuity order of analytic (C∞) geometry.
const ContinuityInfinite = types.ContinuityInfinite
