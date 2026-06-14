// SPDX-License-Identifier: GPL-2.0-only

// Package assembly is the assembly constraint engine (M12-F01,
// Oblikovati/Oblikovati#358/#363): the relationships that position one occurrence
// relative to another, and the solver that applies them. It implements ADR-0011 — the
// constraints build a system of rigid-body variables (6 DOF per occurrence: position
// plus a unit quaternion) and residual equations consumed by the shared solve package,
// the same engine the 2D sketch uses. Static positioning only; contact and dynamics are
// separate (M12-F05, M18).
//
// The package is deliberately decoupled from topology and the document model: a
// constraint reads its geometry as a [Primitive] in a component's definition space, so
// the engine is unit-testable with synthetic primitives. The host (model/compdef)
// extracts primitives from picked faces/edges and owns the [ConstraintSet]; topology
// resolution lives at the boundary (topo_primitive.go), not in the residual math.
package assembly

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/math"
)

// ConstraintListener is notified when the constraint set changes, so the host can raise
// the assembly's constraint events (model/compdef wires this to its event bus). It is
// injected through [NewConstraintSet] rather than imported, keeping this package free of
// the event and document layers (CLAUDE.md: inject dependencies).
type ConstraintListener interface {
	// ConstraintAdded reports a constraint just added to the set.
	ConstraintAdded(c contract.AssemblyConstraint)
	// ConstraintDeleted reports a constraint just removed from the set.
	ConstraintDeleted(c contract.AssemblyConstraint)
	// AssemblyResolved reports that the set was re-solved (positions may have changed).
	AssemblyResolved()
}

// CustomConstraintSolver solves the relationships a [CustomConstraint] represents — an
// add-in-supplied behavior the built-in solver does not know. The host installs one
// through [ConstraintSet.UseCustomSolver]; without it, custom constraints contribute no
// residual (they are held inert, not an error). Injected, never global.
type CustomConstraintSolver interface {
	// Residuals returns the residual values for a custom relationship of the given kind
	// with params, between world-space geometry a and b. All zero ⇒ satisfied.
	Residuals(kind string, params []float64, a, b math.Point3) []float64
}
