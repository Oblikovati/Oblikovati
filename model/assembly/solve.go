// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/types"
	"oblikovati.org/solve"
)

// OccurrenceDOF reports one occurrence's remaining free degrees of freedom after a solve: the scalar
// total plus its translation/rotation split, the DOF centre and the free axes (#1980).
type OccurrenceDOF struct {
	Occurrence       uint64
	DegreesOfFreedom int
	Split            DOFSplit
}

// SolveReport is the outcome of positioning an assembly: overall health, whether the
// numeric solve converged, the active-relationship and redundant-row counts, the total free
// DOF, and the per-occurrence DOF breakdown.
type SolveReport struct {
	Status           types.HealthStatus
	Converged        bool
	Constraints      int
	Redundant        int
	DegreesOfFreedom int
	Occurrences      []OccurrenceDOF
}

// Solve positions the assembly's occurrences to satisfy its constraints and returns the
// health/DOF report. It delegates to the shared relationship solver (so constraints and
// joints can solve together) and tells the listener the assembly resolved.
func (s *ConstraintSet) Solve() SolveReport {
	rep := solveOver(s.occs, s.relationships(), true)
	if s.listener != nil {
		s.listener.AssemblyResolved()
	}
	return rep
}

// Health reports the assembly's current constraint health and DOF without moving any
// occurrence — the analysis of the configuration as it stands.
func (s *ConstraintSet) Health() SolveReport {
	return solveOver(s.occs, s.relationships(), false)
}

// relationships views the constraint set's items as the shared solver's relationships.
func (s *ConstraintSet) relationships() []relationship {
	out := make([]relationship, len(s.items))
	for i, c := range s.items {
		out[i] = c
	}
	return out
}

// residualsSatisfied reports whether every active residual is currently within tolerance —
// i.e. the assembly already sits at a solution.
func residualsSatisfied(residuals []solve.Residual) bool {
	const tol = 1e-7
	for _, r := range residuals {
		for _, v := range r.Residuals() {
			if v > tol || v < -tol {
				return false
			}
		}
	}
	return true
}

// occurrenceDOF returns how many of an occurrence's six rigid-body DOF remain free: 0 when
// grounded, else seven variables minus the rank of the residuals touching them (the extra
// quaternion gauge variable is removed by the normalization row, so a free unconstrained
// occurrence reports 6).
func occurrenceDOF(p *placement, residuals []solve.Residual) int {
	if p.grounded {
		return 0
	}
	return solve.AnalyzeDOF(residuals, p.allVars()).DOF
}
