// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/solve"
)

// relationship is anything that contributes residuals to the assembly solve — a constraint
// (F01) or a joint (F02). Both Constraint and Joint satisfy it, so constraints and joints
// solve together over one placement set (ADR-0011: one solver, joints are reduced-DOF
// residual bundles, not a second mechanism).
type relationship interface {
	Suppressed() bool
	anchors() []anchor
	bind(b binder) []solve.Residual
	setHealth(health.Status)
}

// solveOver positions the occurrences to satisfy the relationships and returns the
// health/DOF report. move=true runs the solver and commits the result (coalescing the moves
// into one notification batch); move=false analyzes the current configuration without
// moving anything. It is the shared engine behind ConstraintSet.Solve/Health and the
// assembly's combined constraint+joint solve.
func solveOver(occs *occurrence.Occurrences, rels []relationship, move bool) SolveReport {
	order, byOcc := buildPlacements(occs)
	residuals, freeVars := assembleResiduals(order, byOcc, rels)
	var analysis solve.DOFAnalysis
	var converged bool
	if move {
		result := solve.Solve(residuals, freeVars, solve.Options{})
		analysis, converged = result.DOFAnalysis, result.Converged
		commitPlacements(occs, order)
	} else {
		analysis = solve.AnalyzeDOF(residuals, freeVars)
		converged = residualsSatisfied(residuals)
	}
	return buildReport(analysis, converged, residuals, order, activeCount(rels))
}

// buildPlacements builds a placement for every non-suppressed occurrence, returned in
// occurrence order (deterministic variable ordering) and keyed for the binder.
func buildPlacements(occs *occurrence.Occurrences) ([]*placement, map[*occurrence.Occurrence]*placement) {
	var order []*placement
	byOcc := map[*occurrence.Occurrence]*placement{}
	for _, o := range occs.All() {
		if o.Suppressed() {
			continue
		}
		p := newPlacement(o)
		order = append(order, p)
		byOcc[o] = p
	}
	return order, byOcc
}

// assembleResiduals collects the residual sources (active relationships plus a
// normalization per free placement) and the free variables the solver drives.
func assembleResiduals(order []*placement, byOcc map[*occurrence.Occurrence]*placement, rels []relationship) ([]solve.Residual, []*math.Scalar) {
	bind := func(o *occurrence.Occurrence) *placement { return byOcc[o] }
	var residuals []solve.Residual
	for _, r := range rels {
		if r.Suppressed() || !resolvable(r, byOcc) {
			continue
		}
		r.setHealth(health.OK)
		residuals = append(residuals, r.bind(bind)...)
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

// resolvable reports whether every anchor of r lands on a placed (non-suppressed)
// occurrence, so its residual can be evaluated.
func resolvable(r relationship, byOcc map[*occurrence.Occurrence]*placement) bool {
	for _, an := range r.anchors() {
		if byOcc[an.occ] == nil {
			return false
		}
	}
	return true
}

// commitPlacements writes each solved placement back to its occurrence in one notification
// batch, so a multi-occurrence solve raises a single coalesced round of transform events.
func commitPlacements(occs *occurrence.Occurrences, order []*placement) {
	occs.SuspendNotifications()
	defer occs.ResumeNotifications()
	for _, p := range order {
		p.commit()
	}
}

// buildReport turns a DOF analysis into the health/DOF report, including the per-occurrence
// DOF (a grounded occurrence is fixed, hence 0). active is the number of non-suppressed
// relationships that participated.
func buildReport(analysis solve.DOFAnalysis, converged bool, residuals []solve.Residual, order []*placement, active int) SolveReport {
	rep := SolveReport{
		Status:           healthFromStatus(analysis.Status),
		Converged:        converged,
		Constraints:      active,
		Redundant:        analysis.Redundant,
		DegreesOfFreedom: analysis.DOF,
	}
	for _, p := range order {
		total := occurrenceDOF(p, residuals)
		rep.Occurrences = append(rep.Occurrences, OccurrenceDOF{
			Occurrence:       p.occ.ID(),
			DegreesOfFreedom: total,
			Split:            occurrenceDOFSplit(p, residuals, total),
		})
	}
	return rep
}

// activeCount returns the number of non-suppressed relationships.
func activeCount(rels []relationship) int {
	n := 0
	for _, r := range rels {
		if !r.Suppressed() {
			n++
		}
	}
	return n
}
