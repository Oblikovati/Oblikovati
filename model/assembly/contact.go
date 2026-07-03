// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"
	"slices"

	"oblikovati.org/api/contract"
	"oblikovati.org/model/internal/collview"
)

// Contact (M12-F05, Oblikovati/Oblikovati#362/#368): a contact set is a named group of
// occurrences that resist interpenetration when one is dragged, and the contact solver toggles
// whether that resistance is enforced. Membership is pure bookkeeping (ids); the geometric
// contact-stop and interference analysis read the occurrence bodies (compdef). These satisfy
// contract.ContactSet(s)/ContactSolver.

// ContactSetView is a contact set's host-read surface (so the router encodes it without naming
// the unexported concrete type).
type ContactSetView interface {
	contract.ContactSet
	Members() []uint64
}

// contactSet is a named group of occurrence ids that resist interpenetration.
type contactSet struct {
	id      uint64
	name    string
	members []uint64
}

// ID returns the contact set's session id.
func (c *contactSet) ID() uint64 { return c.id }

// Name returns the contact set's display name.
func (c *contactSet) Name() string { return c.name }

// MemberCount returns the number of occurrences in the set.
func (c *contactSet) MemberCount() int { return len(c.members) }

// Members returns the member occurrence ids (a copy).
func (c *contactSet) Members() []uint64 {
	out := make([]uint64, len(c.members))
	copy(out, c.members)
	return out
}

// ContactSolver owns an assembly's contact sets and the enforce-on-drag toggle.
type ContactSolver struct {
	enabled bool
	sets    []*contactSet
	nextID  uint64
}

// NewContactSolver builds an empty, disabled contact solver.
func NewContactSolver() *ContactSolver { return &ContactSolver{} }

// Enabled reports whether contact is enforced during a drag.
func (s *ContactSolver) Enabled() bool { return s.enabled }

// SetEnabled turns contact enforcement on or off.
func (s *ContactSolver) SetEnabled(on bool) { s.enabled = on }

// SetCount returns the number of contact sets.
func (s *ContactSolver) SetCount() int { return len(s.sets) }

// Create adds a new contact set named name and returns its host-read view.
func (s *ContactSolver) Create(name string) ContactSetView {
	s.nextID++
	if name == "" {
		name = fmt.Sprintf("Contact:%d", len(s.sets)+1)
	}
	cs := &contactSet{id: s.nextID, name: name}
	s.sets = append(s.sets, cs)
	return cs
}

// ByID returns the contact set with the given id (host-read view), or nil.
func (s *ContactSolver) ByID(id uint64) ContactSetView {
	if cs := s.setByID(id); cs != nil {
		return cs
	}
	return nil
}

// setByID returns the concrete contact set for internal mutation, or nil.
func (s *ContactSolver) setByID(id uint64) *contactSet {
	for _, cs := range s.sets {
		if cs.id == id {
			return cs
		}
	}
	return nil
}

// All returns the contact sets in creation order (host-read views).
func (s *ContactSolver) All() []ContactSetView {
	out := make([]ContactSetView, len(s.sets))
	for i, cs := range s.sets {
		out[i] = cs
	}
	return out
}

// AddMember adds an occurrence to a contact set (idempotent), erroring on an unknown set.
func (s *ContactSolver) AddMember(setID, occurrence uint64) error {
	cs := s.setByID(setID)
	if cs == nil {
		return fmt.Errorf("assembly: no contact set with id %d", setID)
	}
	for _, m := range cs.members {
		if m == occurrence {
			return nil
		}
	}
	cs.members = append(cs.members, occurrence)
	return nil
}

// RemoveMember removes an occurrence from a contact set, erroring on an unknown set.
func (s *ContactSolver) RemoveMember(setID, occurrence uint64) error {
	cs := s.setByID(setID)
	if cs == nil {
		return fmt.Errorf("assembly: no contact set with id %d", setID)
	}
	for i, m := range cs.members {
		if m == occurrence {
			cs.members = append(cs.members[:i], cs.members[i+1:]...)
			return nil
		}
	}
	return nil
}

// Delete removes a contact set by id, reporting whether it was found.
func (s *ContactSolver) Delete(id uint64) bool {
	for i, cs := range s.sets {
		if cs.id == id {
			s.sets = append(s.sets[:i], s.sets[i+1:]...)
			return true
		}
	}
	return false
}

// Count / Item give the contract.ContactSets read surface.
func (s *ContactSolver) Count() int { return len(s.sets) }
func (s *ContactSolver) Item(i int) contract.ContactSet {
	return collview.ItemAs(s.sets, i, func(cs *contactSet) contract.ContactSet { return cs })
}

// PartnersOf returns the occurrences that share a contact set with occID — the components its
// motion must not interpenetrate.
func (s *ContactSolver) PartnersOf(occID uint64) []uint64 {
	seen := map[uint64]bool{}
	var out []uint64
	for _, cs := range s.sets {
		if !slices.Contains(cs.members, occID) {
			continue
		}
		for _, m := range cs.members {
			if m != occID && !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	return out
}

// Contacts reports whether the two occurrences share a contact set — the pairs the contact
// solver keeps from interpenetrating.
func (s *ContactSolver) Contacts(a, b uint64) bool {
	for _, cs := range s.sets {
		if slices.Contains(cs.members, a) && slices.Contains(cs.members, b) {
			return true
		}
	}
	return false
}
