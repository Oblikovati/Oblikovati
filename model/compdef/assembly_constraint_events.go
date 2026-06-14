// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/event"
	"oblikovati.org/model/assembly"
)

// Assembly relationship (constraint) event type ids. The high byte 0x0C mirrors the
// milestone (M12); ids are stable across versions because they cross the seam to add-ins
// (never renumber). They join the assembly's occurrence-lifecycle family (0x0B,
// assembly_events.go) on the same bus.
const (
	tidConstraintAdd    event.TypeID = 0x0C01
	tidConstraintDelete event.TypeID = 0x0C02
	tidAssemblyResolved event.TypeID = 0x0C03
)

// ConstraintAdd is raised (After) when a constraint is added to the assembly — the
// reference API's relationship-added notification.
type ConstraintAdd struct{ Constraint contract.AssemblyConstraint }

// EventID implements event.Event.
func (ConstraintAdd) EventID() event.TypeID { return tidConstraintAdd }

// ConstraintDelete is raised (After) when a constraint is removed from the assembly.
type ConstraintDelete struct{ Constraint contract.AssemblyConstraint }

// EventID implements event.Event.
func (ConstraintDelete) EventID() event.TypeID { return tidConstraintDelete }

// AssemblyResolved is raised (After) when the constraint set is re-solved — occurrence
// placements may have changed, so a consumer re-reads the occurrence tree.
type AssemblyResolved struct{}

// EventID implements event.Event.
func (AssemblyResolved) EventID() event.TypeID { return tidAssemblyResolved }

// AssemblyEvents is the sink the constraint set notifies (M12-F01).
var _ assembly.ConstraintListener = (*AssemblyEvents)(nil)

// ConstraintAdded raises ConstraintAdd (After). Implements assembly.ConstraintListener.
func (e *AssemblyEvents) ConstraintAdded(c contract.AssemblyConstraint) {
	event.Emit(e.bus, event.After, ConstraintAdd{Constraint: c})
}

// ConstraintDeleted raises ConstraintDelete (After).
func (e *AssemblyEvents) ConstraintDeleted(c contract.AssemblyConstraint) {
	event.Emit(e.bus, event.After, ConstraintDelete{Constraint: c})
}

// AssemblyResolved raises AssemblyResolved (After).
func (e *AssemblyEvents) AssemblyResolved() {
	event.Emit(e.bus, event.After, AssemblyResolved{})
}
