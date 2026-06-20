// SPDX-License-Identifier: GPL-2.0-only

package sketch

import stdmath "math"

// Relax-mode drag-resolve (#791): Inventor's Relax Mode lets a user drag fully- or
// over-constrained geometry — the conflicting driving dimensions yield and update to the
// dragged value instead of holding the geometry rigid. A normal DragSolve cannot do this:
// it keeps every driving dimension in the constraint set, so a fully-dimensioned entity has
// no remaining freedom and the pin just fights the dimensions. RelaxSolve drops the driving
// dimensions from the solve so the geometry follows the cursor under its geometric
// constraints alone, then rewrites each dimension that actually moved to its new measured
// value — so the dimension "relaxes" to follow the drag rather than blocking it.

// relaxEpsilon is the measured-vs-target drift above which a driving dimension is considered
// to have been disturbed by the relax drag and is rewritten to its new value. Below it the
// dimension is left untouched so its expression/formula survives an unrelated drag.
const relaxEpsilon = 1e-9

// RelaxSolve re-solves the sketch with the dragged points pinned to their cursor targets
// while relaxing every driving dimension. Geometric constraints (coincident, horizontal,
// parallel, …) are still honoured, so the geometry deforms sensibly; only the dimensional
// constraints yield. After the solve it rewrites each disturbed dimension to its measured
// value. The pins are temporary (never persisted); geometry is left in the solved
// configuration. Returns the solve result.
//
//	sk.RelaxSolve([]PinTarget{{P: line.A, Target: cursor}}) // drag a dimensioned line freely
func (s *Sketch) RelaxSolve(pins []PinTarget) SolveResult {
	cons := append([]Constraint(nil), s.geomCons.All()...) // geometric only — dimensions relax
	for _, pt := range pins {
		cons = append(cons, dragFix(pt.P, pt.Target))
	}
	res := Solve(cons, s.variables(), Options{})
	s.relaxDrivingDimensions()
	return res
}

// relaxDrivingDimensions rewrites each driving dimension whose geometry moved during a relax
// drag to its freshly measured value, so the dimension reads the dragged geometry (no
// residual) afterwards. Dimensions whose measurement did not drift are left alone so an
// unrelated drag does not flatten their expressions into literals. Driven dimensions already
// only report, so they are skipped.
func (s *Sketch) relaxDrivingDimensions() {
	for _, d := range s.dimCons.items {
		if d.driven {
			continue
		}
		measured := d.Measured()
		if stdmath.Abs(measured-d.param.ModelValue()) < relaxEpsilon {
			continue
		}
		_ = d.Drive(measured) // honours any enabled drive limits; literal value is the new target
	}
}
