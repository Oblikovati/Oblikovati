// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/solve"

// CustomConstraint is a relationship solved by an add-in (the reference API's
// CustomConstraint), not the built-in solver. It carries a kind name and driving params;
// at solve time it dispatches to the host's installed [CustomConstraintSolver]
// ([ConstraintSet.UseCustomSolver]). With no solver installed it contributes no residual —
// it is held inert (registered, listed, deletable), never an error.
type CustomConstraint struct {
	*constraintBase
	kind   string
	params []float64
	set    *ConstraintSet
}

// Value returns the first driving parameter, or 0 when there are none.
func (c *CustomConstraint) Value() float64 {
	if len(c.params) == 0 {
		return 0
	}
	return c.params[0]
}

// Kind returns the add-in relationship name the host dispatches to.
func (c *CustomConstraint) Kind() string { return c.kind }

// Params returns a copy of the driving parameters.
func (c *CustomConstraint) Params() []float64 {
	return append([]float64(nil), c.params...)
}

// bind dispatches to the installed custom solver over the two anchors' world points, or
// contributes no residual when no solver is installed.
func (c *CustomConstraint) bind(b binder) []solve.Residual {
	if c.set == nil || c.set.custom == nil {
		return nil
	}
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		return c.set.custom.Residuals(c.kind, c.params,
			worldPoint(pa.matrix(), c.a.prim), worldPoint(pb.matrix(), c.b.prim))
	})
}
