// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/solve"
)

// variables returns the full DOF universe of the planar sketch: every point's X,Y, circle and
// ellipse radii, and each ellipse's orientation angle. The solver needs the whole universe (not
// just constraint-referenced variables) so that free, unconstrained geometry is counted in the
// DOF total — an unconstrained ellipse genuinely has a rotational DOF (#1879 AC2). Each ellipse's
// orientation is re-seeded from its authoritative MajorAxis here, so every solve/analyze path
// starts consistent regardless of who last moved the axis.
func (s *Sketch) variables() []*math.Scalar {
	vars := make([]*math.Scalar, 0, len(s.pts)*2)
	for _, p := range s.pts {
		vars = append(vars, &p.X, &p.Y)
	}
	// A reference (projected) entity is grounded: its points live in refPts (already excluded above)
	// and its scalar DOF (radius, ellipse radii/orientation) are held fixed too, so it is driven by
	// its source, not solved (ADR-0055 phase 3).
	for _, c := range s.circles.items {
		if c.reference {
			continue
		}
		vars = append(vars, &c.Radius)
	}
	for _, e := range s.ellipses.items {
		if e.reference {
			continue
		}
		e.seedOrientation()
		vars = append(vars, &e.MajorRadius, &e.MinorRadius, e.axisAngle())
	}
	// Elliptical arcs contribute only their orientation DOF here; their radii were never in the
	// DOF universe (an arc's extent is authored, not solved), and this change keeps that scope.
	for _, e := range s.ellArcs.items {
		if e.reference {
			continue
		}
		e.seedOrientation()
		vars = append(vars, e.axisAngle())
	}
	// An offset spline's signed offset distance is a solver DOF, so an offset-spline dimension can
	// drive it and an undriven offset is genuinely free (#1874).
	for _, o := range s.offSpl.items {
		vars = append(vars, &o.Dist)
	}
	return vars
}

// syncEllipseAxes rewrites every ellipse's MajorAxis from its solver-moved orientation, restoring
// the authoritative direction after a geometry-mutating solve (#1879 AC2). Analyze-only paths that
// build variables() but never move geometry skip this — orientation was only seeded, not changed.
func (s *Sketch) syncEllipseAxes() {
	// A grounded reference ellipse (a projected edge) is excluded from variables(), so its
	// orientation was never seeded/solved this pass; rewriting MajorAxis from that stale angle
	// would corrupt the driven axis, so leave reference conics untouched (ADR-0055 phase 3).
	for _, e := range s.ellipses.items {
		if e.reference {
			continue
		}
		e.syncAxisFromOrientation()
	}
	for _, e := range s.ellArcs.items {
		if e.reference {
			continue
		}
		e.syncAxisFromOrientation()
	}
}

// Solve resolves the sketch from its constraints, updating geometry in place and
// recording health: a sketch that cannot be solved (non-convergent / conflicting)
// becomes sick rather than crashing; an over-constrained-but-solvable sketch is a
// warning; otherwise it is healthy. Returns the full [SolveResult].
func (s *Sketch) Solve() SolveResult {
	result := Solve(s.Constraints(), s.variables(), Options{})
	s.syncEllipseAxes()
	s.health = healthFromSolve(result)
	return result
}

// AnalyzeConstraints reports the DOF/redundancy structure without moving geometry.
func (s *Sketch) AnalyzeConstraints() DOFAnalysis {
	return analyzeDOF(s.Constraints(), s.variables())
}

// ConflictingConstraints solves the sketch and returns the constraints it could not
// satisfy — the offending subset for UI conflict diagnosis (#1420), most severe first,
// or nil when the sketch solves cleanly. Like [Sketch.Solve] it moves geometry (it is a
// solve); call it when a sketch reports sick to name the dimension(s) to relax.
func (s *Sketch) ConflictingConstraints() []Constraint {
	cons := s.Constraints()
	r := Solve(cons, s.variables(), Options{})
	s.syncEllipseAxes()
	if len(r.Conflicts) == 0 {
		return nil
	}
	out := make([]Constraint, 0, len(r.Conflicts))
	for _, i := range r.Conflicts {
		out = append(out, cons[i])
	}
	return out
}

// RedundantConstraints returns the constraints whose removal restores full rank on an
// over-constrained sketch — solvespace-style leave-one-out identification (audit A13, #1609):
// where ConflictingConstraints names UNSATISFIABLE dimensions on a failed solve, this names
// the linearly DEPENDENT ones behind an "over-constrained" verdict, so the UI can point at a
// specific constraint to remove. nil when the sketch is not redundant.
func (s *Sketch) RedundantConstraints() []Constraint {
	cons := s.Constraints()
	idx := solve.RedundantSources(residualSources(cons), s.variables())
	out := make([]Constraint, 0, len(idx))
	for _, i := range idx {
		out = append(out, cons[i])
	}
	return out
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
