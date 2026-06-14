// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/solve"
)

// OccurrenceDOF reports one occurrence's remaining free degrees of freedom after a solve.
type OccurrenceDOF struct {
	Occurrence       uint64
	DegreesOfFreedom int
}

// SolveReport is the outcome of positioning an assembly: overall health, whether the
// numeric solve converged, the active-constraint and redundant-row counts, the total free
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
// health/DOF report. It warm-starts from the current placements, runs the shared solver,
// writes the solved transforms back (coalescing the moves into one notification batch),
// and tells the listener the assembly resolved.
func (s *ConstraintSet) Solve() SolveReport {
	order, byOcc := s.placements()
	residuals, freeVars := s.assemble(order, byOcc)
	result := solve.Solve(residuals, freeVars, solve.Options{})
	s.commit(order)
	if s.listener != nil {
		s.listener.AssemblyResolved()
	}
	return s.report(result, residuals, order)
}

// placements builds a placement for every non-suppressed occurrence, returned both in
// occurrence order (deterministic variable ordering) and keyed for the binder.
func (s *ConstraintSet) placements() ([]*placement, map[*occurrence.Occurrence]*placement) {
	var order []*placement
	byOcc := map[*occurrence.Occurrence]*placement{}
	for _, o := range s.occs.All() {
		if o.Suppressed() {
			continue
		}
		p := newPlacement(o)
		order = append(order, p)
		byOcc[o] = p
	}
	return order, byOcc
}

// assemble collects the residual sources (active constraints plus a normalization per free
// placement) and the free variables the solver drives.
func (s *ConstraintSet) assemble(order []*placement, byOcc map[*occurrence.Occurrence]*placement) ([]solve.Residual, []*math.Scalar) {
	bind := func(o *occurrence.Occurrence) *placement { return byOcc[o] }
	var residuals []solve.Residual
	for _, c := range s.items {
		if c.Suppressed() || !resolvable(c, byOcc) {
			continue
		}
		c.setHealth(health.OK)
		residuals = append(residuals, c.bind(bind)...)
	}
	var freeVars []*math.Scalar
	for _, p := range order {
		if p.grounded {
			continue
		}
		residuals = append(residuals, normalization{p})
		freeVars = append(freeVars, p.vars()...)
	}
	return residuals, freeVars
}

// resolvable reports whether every anchor of c lands on a placed (non-suppressed)
// occurrence, so its residual can be evaluated.
func resolvable(c Constraint, byOcc map[*occurrence.Occurrence]*placement) bool {
	for _, an := range c.anchors() {
		if byOcc[an.occ] == nil {
			return false
		}
	}
	return true
}

// commit writes each solved placement back to its occurrence in one notification batch, so
// a multi-occurrence solve raises a single coalesced round of transform events.
func (s *ConstraintSet) commit(order []*placement) {
	s.occs.SuspendNotifications()
	defer s.occs.ResumeNotifications()
	for _, p := range order {
		p.commit()
	}
}

// report turns the solve outcome into the health/DOF report, including the per-occurrence
// DOF (a grounded occurrence is fixed, hence 0).
func (s *ConstraintSet) report(result solve.Result, residuals []solve.Residual, order []*placement) SolveReport {
	rep := SolveReport{
		Status:           healthFromStatus(result.Status),
		Converged:        result.Converged,
		Constraints:      s.activeCount(),
		Redundant:        result.Redundant,
		DegreesOfFreedom: result.DOF,
	}
	for _, p := range order {
		rep.Occurrences = append(rep.Occurrences, OccurrenceDOF{
			Occurrence:       p.occ.ID(),
			DegreesOfFreedom: occurrenceDOF(p, residuals),
		})
	}
	return rep
}

// activeCount returns the number of non-suppressed constraints.
func (s *ConstraintSet) activeCount() int {
	n := 0
	for _, c := range s.items {
		if !c.Suppressed() {
			n++
		}
	}
	return n
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
