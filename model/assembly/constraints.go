// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/internal/collview"
	"oblikovati.org/model/occurrence"
)

// ConstraintSet is an assembly's constraint collection and the entry point to the solver
// (the reference API's AssemblyConstraints, plus AddMate/AddFlush/... factories). It owns
// no geometry: it positions the occurrences of the [occurrence.Occurrences] it is given,
// reading each constraint's [Primitive] inputs through their occurrences' placements.
type ConstraintSet struct {
	occs     *occurrence.Occurrences
	items    []Constraint
	listener ConstraintListener
	custom   CustomConstraintSolver
	nextID   uint64
	counts   map[types.AssemblyConstraintType]int
}

// NewConstraintSet builds an empty set positioning the given occurrences. The listener
// (may be nil) receives add/delete/resolve notifications for the host's event bus.
func NewConstraintSet(occs *occurrence.Occurrences, l ConstraintListener) *ConstraintSet {
	return &ConstraintSet{occs: occs, listener: l, counts: map[types.AssemblyConstraintType]int{}}
}

// UseCustomSolver installs the solver for custom constraints (replacing any previous one).
func (s *ConstraintSet) UseCustomSolver(cs CustomConstraintSolver) { s.custom = cs }

// Count returns the number of constraints in the set.
func (s *ConstraintSet) Count() int { return len(s.items) }

// Item returns the constraint at index i (0-based), or nil when out of range.
func (s *ConstraintSet) Item(i int) contract.AssemblyConstraint {
	return collview.ItemAs(s.items, i, asContractConstraint)
}

// All returns the constraints in creation order.
func (s *ConstraintSet) All() []Constraint {
	out := make([]Constraint, len(s.items))
	copy(out, s.items)
	return out
}

// ByID returns the constraint with the given session id, or nil.
func (s *ConstraintSet) ByID(id uint64) Constraint {
	for _, c := range s.items {
		if c.ID() == id {
			return c
		}
	}
	return nil
}

// Delete removes the constraint with the given id, returning whether it was found.
func (s *ConstraintSet) Delete(id uint64) bool {
	for i, c := range s.items {
		if c.ID() != id {
			continue
		}
		s.items = append(s.items[:i], s.items[i+1:]...)
		if s.listener != nil {
			s.listener.ConstraintDeleted(c)
		}
		return true
	}
	return false
}

// SetLimits sets (lim non-nil) or clears (lim nil) the constraint's driven-value bounds,
// returning an error naming the id when it is unknown.
func (s *ConstraintSet) SetLimits(id uint64, lim *limits) error {
	c := s.ByID(id)
	if c == nil {
		return fmt.Errorf("assembly: no constraint with id %d", id)
	}
	c.setLimits(lim)
	return nil
}

// ForOccurrence returns the per-occurrence view of the constraints that reference o (the
// reference API's per-component Constraints enumerator).
func (s *ConstraintSet) ForOccurrence(o *occurrence.Occurrence) *OccurrenceConstraints {
	var hit []Constraint
	for _, c := range s.items {
		if constraintReferences(c, o) {
			hit = append(hit, c)
		}
	}
	return &OccurrenceConstraints{items: hit}
}

// add appends a new constraint and notifies the listener.
func (s *ConstraintSet) add(c Constraint) {
	s.items = append(s.items, c)
	if s.listener != nil {
		s.listener.ConstraintAdded(c)
	}
}

// newBase mints the shared identity/name for a constraint of the given kind over the two
// geometry inputs.
func (s *ConstraintSet) newBase(kind types.AssemblyConstraintType, a, b Ref) *constraintBase {
	s.nextID++
	s.counts[kind]++
	return &constraintBase{
		relationshipBase: relationshipBase{id: s.nextID, name: constraintName(kind, s.counts[kind]), a: toAnchor(a), b: toAnchor(b)},
		kind:             kind,
	}
}

// constraintReferences reports whether c has an anchor on occurrence o.
func constraintReferences(c Constraint, o *occurrence.Occurrence) bool {
	for _, an := range c.anchors() {
		if an.occ == o {
			return true
		}
	}
	return false
}

// OccurrenceConstraints is the per-occurrence constraints view
// (contract.AssemblyConstraintsEnumerator).
type OccurrenceConstraints struct{ items []Constraint }

// Count returns the number of constraints referencing the occurrence.
func (e *OccurrenceConstraints) Count() int { return len(e.items) }

// Item returns the i-th constraint referencing the occurrence, or nil when out of range.
func (e *OccurrenceConstraints) Item(i int) contract.AssemblyConstraint {
	return collview.ItemAs(e.items, i, asContractConstraint)
}

// asContractConstraint widens the engine's Constraint to the public contract for the
// shared collview guard (#1655).
func asContractConstraint(c Constraint) contract.AssemblyConstraint { return c }
