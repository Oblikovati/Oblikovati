// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/solve"
)

// Ref is a constraint geometry input as the host supplies it: the occurrence the geometry
// belongs to, the definition-space [Primitive] the solver consumes, and the opaque Entity
// reference key the geometry was resolved from (carried for round-trip reporting and
// future persistence — the engine never interprets it).
type Ref struct {
	Occurrence *occurrence.Occurrence
	Primitive  Primitive
	Entity     string
}

// AnchorRef is the reportable identity of one of a constraint's inputs: which occurrence,
// and the reference key on it.
type AnchorRef struct {
	Occurrence uint64
	Entity     string
}

// anchor pairs one of a constraint's geometry inputs with the occurrence it belongs to.
// The occurrence is resolved when the constraint is created; the [Primitive] is in that
// component's definition space and is transformed by the occurrence's placement at solve
// time. entity is the opaque reference key, kept for reporting.
type anchor struct {
	occ    *occurrence.Occurrence
	prim   Primitive
	entity string
}

// toAnchor builds an internal anchor from a host-supplied [Ref].
func toAnchor(r Ref) anchor { return anchor{occ: r.Occurrence, prim: r.Primitive, entity: r.Entity} }

// binder resolves an occurrence to its live placement during a solve, so a constraint's
// residual reads the same mutable variable block the solver is driving.
type binder func(*occurrence.Occurrence) *placement

// residualFn adapts a closure to solve.Residual, letting each constraint express its
// residuals as a function over the bound placements' live matrices.
type residualFn func() []float64

// Residuals implements solve.Residual.
func (f residualFn) Residuals() []float64 { return f() }

// single wraps one residual closure as the one-element residual-source slice most
// constraints return from bind.
func single(fn func() []float64) []solve.Residual { return []solve.Residual{residualFn(fn)} }

// Constraint is the internal interface every assembly relationship implements: the public
// read surface (contract.AssemblyConstraint) plus the solver-binding and mutation hooks
// the ConstraintSet drives. Concrete kinds embed [constraintBase] and add their Value and
// bind behavior.
type Constraint interface {
	contract.AssemblyConstraint
	// SetSuppressed includes or excludes the constraint from the solve.
	SetSuppressed(bool)
	// anchors returns the constraint's geometry inputs (for the per-occurrence view).
	anchors() []anchor
	// AnchorRefs returns the two primary inputs' reportable identities (A, B).
	AnchorRefs() (AnchorRef, AnchorRef)
	// bind returns the residual sources for this constraint over the bound placements.
	bind(b binder) []solve.Residual
	// setHealth records the constraint's evaluated health.
	setHealth(health.Status)
	// setLimits sets (lim non-nil) or clears (lim nil) the driven-value bounds.
	setLimits(lim *limits)
}

// constraintBase carries the identity, kind, anchors, health, suppression, and optional
// limits shared by every constraint. Concrete kinds embed it by pointer.
type constraintBase struct {
	id         uint64
	name       string
	kind       types.AssemblyConstraintType
	a, b       anchor
	status     health.Status
	suppressed bool
	lim        *limits
}

// ID returns the constraint's session id.
func (c *constraintBase) ID() uint64 { return c.id }

// Type returns the relationship kind.
func (c *constraintBase) Type() types.AssemblyConstraintType { return c.kind }

// Name returns the constraint's display name.
func (c *constraintBase) Name() string { return c.name }

// Value returns the driven value; the base has none (overridden by kinds that do).
func (c *constraintBase) Value() float64 { return 0 }

// Health reports the constraint's evaluated health.
func (c *constraintBase) Health() types.HealthStatus { return publicHealth(c.status) }

// Suppressed reports whether the constraint is excluded from the solve.
func (c *constraintBase) Suppressed() bool { return c.suppressed }

// SetSuppressed includes or excludes the constraint from the solve.
func (c *constraintBase) SetSuppressed(s bool) { c.suppressed = s }

// Limits returns the driven-value bounds, or a nil interface when unbounded.
func (c *constraintBase) Limits() contract.ConstraintLimits {
	if c.lim == nil {
		return nil
	}
	return c.lim
}

// anchors returns the two geometry inputs.
func (c *constraintBase) anchors() []anchor { return []anchor{c.a, c.b} }

// AnchorRefs returns the two primary inputs' reportable identities.
func (c *constraintBase) AnchorRefs() (AnchorRef, AnchorRef) {
	return AnchorRef{c.a.occ.ID(), c.a.entity}, AnchorRef{c.b.occ.ID(), c.b.entity}
}

// setHealth records the evaluated health.
func (c *constraintBase) setHealth(s health.Status) { c.status = s }

// setLimits sets or clears the driven-value bounds.
func (c *constraintBase) setLimits(lim *limits) { c.lim = lim }

// boundMatrices resolves both anchors' placements and returns their live transforms — the
// common preamble of every constraint's residual closure.
func (c *constraintBase) boundPlacements(b binder) (*placement, *placement) {
	return b(c.a.occ), b(c.b.occ)
}
