// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// variables returns the full DOF universe of the planar sketch: every point's X,Y
// plus circle and ellipse radii. The solver needs the whole universe (not just
// constraint-referenced variables) so that free, unconstrained geometry is counted
// in the DOF total.
func (s *Sketch) variables() []*math.Scalar {
	vars := make([]*math.Scalar, 0, len(s.pts)*2)
	for _, p := range s.pts {
		vars = append(vars, &p.X, &p.Y)
	}
	for _, c := range s.circles.items {
		vars = append(vars, &c.Radius)
	}
	for _, e := range s.ellipses.items {
		vars = append(vars, &e.MajorRadius, &e.MinorRadius)
	}
	return vars
}

// Solve resolves the sketch from its constraints, updating geometry in place and
// recording health: a sketch that cannot be solved (non-convergent / conflicting)
// becomes sick rather than crashing; an over-constrained-but-solvable sketch is a
// warning; otherwise it is healthy. Returns the full [SolveResult].
func (s *Sketch) Solve() SolveResult {
	result := Solve(s.Constraints(), s.variables(), Options{})
	s.health = healthFromSolve(result)
	return result
}

// AnalyzeConstraints reports the DOF/redundancy structure without moving geometry.
func (s *Sketch) AnalyzeConstraints() DOFAnalysis {
	return analyzeDOF(s.Constraints(), s.variables())
}

// DegreesOfFreedom returns the sketch's remaining free degrees of freedom (0 when
// fully constrained).
func (s *Sketch) DegreesOfFreedom() int {
	return s.AnalyzeConstraints().DOF
}

// healthFromSolve maps a solve outcome onto modeling health.
func healthFromSolve(r SolveResult) health.Health {
	switch {
	case !r.Converged:
		return health.Sicken("sketch could not be solved")
	case r.Status == OverConstrained:
		return health.Health{Status: health.Warning, Reason: "redundant or conflicting constraints"}
	default:
		return health.Healthy
	}
}
