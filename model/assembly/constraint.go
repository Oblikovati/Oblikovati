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
	// SetValue overrides the constraint's driven value (offset/angle/ratio) — the seam a
	// positional representation uses (M12-F04). A constraint with no value ignores it.
	SetValue(v float64)
}

// relationshipBase carries the identity, two geometry anchors, health, and suppression
// shared by every assembly relationship — a constraint or a joint (M12-F02). Both
// constraintBase and jointBase embed it; the solve treats both as a [relationship].
type relationshipBase struct {
	id         uint64
	name       string
	a, b       anchor
	status     health.Status
	suppressed bool
}

// ID returns the relationship's session id.
func (r *relationshipBase) ID() uint64 { return r.id }

// Name returns the relationship's display name.
func (r *relationshipBase) Name() string { return r.name }

// Health reports the relationship's evaluated health.
func (r *relationshipBase) Health() types.HealthStatus { return publicHealth(r.status) }

// Suppressed reports whether the relationship is excluded from the solve.
func (r *relationshipBase) Suppressed() bool { return r.suppressed }

// SetSuppressed includes or excludes the relationship from the solve.
func (r *relationshipBase) SetSuppressed(s bool) { r.suppressed = s }

// anchors returns the two geometry inputs.
func (r *relationshipBase) anchors() []anchor { return []anchor{r.a, r.b} }

// AnchorRefs returns the two primary inputs' reportable identities.
func (r *relationshipBase) AnchorRefs() (AnchorRef, AnchorRef) {
	return AnchorRef{r.a.occ.ID(), r.a.entity}, AnchorRef{r.b.occ.ID(), r.b.entity}
}

// setHealth records the evaluated health.
func (r *relationshipBase) setHealth(s health.Status) { r.status = s }

// boundPlacements resolves both anchors' placements — the common preamble of every
// relationship's residual closure.
func (r *relationshipBase) boundPlacements(b binder) (*placement, *placement) {
	return b(r.a.occ), b(r.b.occ)
}

// constraintBase adds the constraint kind and optional limits to the shared base.
type constraintBase struct {
	relationshipBase
	kind types.AssemblyConstraintType
	lim  *limits
}

// Type returns the constraint kind.
func (c *constraintBase) Type() types.AssemblyConstraintType { return c.kind }

// Value returns the driven value; the base has none (overridden by kinds that do).
func (c *constraintBase) Value() float64 { return 0 }

// SetValue is a no-op by default — a constraint with no driven value (e.g. a custom
// residual) ignores a positional override. Value-bearing kinds override it.
func (c *constraintBase) SetValue(float64) { /* no driven value to set; value-bearing kinds override */
}

// Limits returns the driven-value bounds, or a nil interface when unbounded.
func (c *constraintBase) Limits() contract.ConstraintLimits {
	if c.lim == nil {
		return nil
	}
	return c.lim
}

// setLimits sets or clears the driven-value bounds.
func (c *constraintBase) setLimits(lim *limits) { c.lim = lim }
