// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// Assembly occurrence-lifecycle event type ids. The high byte 0x0B mirrors the
// milestone (M11); ids are stable across versions because they cross the gRPC seam to
// add-ins (never renumber). Constraint/joint (relationship) events join this family
// once M12-F01 (#358) lands; they are intentionally absent here.
const (
	tidOccurrenceAdd       event.TypeID = 0x0B01
	tidOccurrenceDelete    event.TypeID = 0x0B02
	tidOccurrenceReplace   event.TypeID = 0x0B03
	tidOccurrenceTransform event.TypeID = 0x0B04
	tidOccurrenceSuppress  event.TypeID = 0x0B05
)

// OccurrenceAdd is raised (After) when an occurrence is placed in the assembly — the
// reference API's AssemblyEvents.OnOccurrenceAdd.
type OccurrenceAdd struct{ Occurrence *occurrence.Occurrence }

// EventID implements event.Event.
func (OccurrenceAdd) EventID() event.TypeID { return tidOccurrenceAdd }

// OccurrenceDelete is raised around deleting an occurrence: vetoable in the Before
// phase (driven by [AssemblyComponentDefinition.DeleteOccurrence]), then After once the
// removal commits.
type OccurrenceDelete struct{ Occurrence *occurrence.Occurrence }

// EventID implements event.Event.
func (OccurrenceDelete) EventID() event.TypeID { return tidOccurrenceDelete }

// OccurrenceReplace is raised (After) when an occurrence's definition is swapped,
// carrying the definition it held before.
type OccurrenceReplace struct {
	Occurrence *occurrence.Occurrence
	Previous   occurrence.Definition
}

// EventID implements event.Event.
func (OccurrenceReplace) EventID() event.TypeID { return tidOccurrenceReplace }

// OccurrenceTransform is raised (After) when an occurrence is repositioned, carrying
// its prior placement. During a drag batch it fires once per occurrence at resume (see
// occurrence.Occurrences.SuspendNotifications), carrying the pre-batch placement.
type OccurrenceTransform struct {
	Occurrence *occurrence.Occurrence
	Previous   math.Matrix4
}

// EventID implements event.Event.
func (OccurrenceTransform) EventID() event.TypeID { return tidOccurrenceTransform }

// OccurrenceSuppress is raised around toggling an occurrence's suppression: vetoable in
// the Before phase (driven by [AssemblyComponentDefinition.SetOccurrenceSuppressed]),
// then After once the change commits. Suppressed reports the requested/new state.
type OccurrenceSuppress struct {
	Occurrence *occurrence.Occurrence
	Suppressed bool
}

// EventID implements event.Event.
func (OccurrenceSuppress) EventID() event.TypeID { return tidOccurrenceSuppress }

// AssemblyEvents is an assembly definition's occurrence-lifecycle event source — the
// reference API's AssemblyEvents (M11-F07, #632). It bridges raw occurrence mutations
// to a typed [event.Bus] that add-ins subscribe to: it implements
// [occurrence.OccurrenceListener] so the assembly's occurrences notify it in the After
// phase, and it raises the vetoable Before phase for operations a caller initiates
// through the definition (delete, suppress).
//
// Example: subscribe to placements —
//
//	event.Subscribe(asm.Events().Bus(), event.After,
//	    func(_ event.Context, e compdef.OccurrenceAdd) event.Outcome {
//	        log.Printf("placed %s", e.Occurrence.Name())
//	        return event.Continue()
//	    })
type AssemblyEvents struct{ bus *event.Bus }

// AssemblyEvents is the sink the occurrence collection notifies.
var _ occurrence.OccurrenceListener = (*AssemblyEvents)(nil)

// newAssemblyEvents returns an event source on a fresh bus, installed by the assembly
// definition constructor.
func newAssemblyEvents() *AssemblyEvents { return &AssemblyEvents{bus: event.NewBus()} }

// Bus returns the event bus to subscribe handlers on with [event.Subscribe].
func (e *AssemblyEvents) Bus() *event.Bus { return e.bus }

// OccurrenceAdded raises OccurrenceAdd (After). Implements occurrence.OccurrenceListener.
func (e *AssemblyEvents) OccurrenceAdded(o *occurrence.Occurrence) {
	event.Emit(e.bus, event.After, OccurrenceAdd{Occurrence: o})
}

// OccurrenceRemoved raises OccurrenceDelete (After).
func (e *AssemblyEvents) OccurrenceRemoved(o *occurrence.Occurrence) {
	event.Emit(e.bus, event.After, OccurrenceDelete{Occurrence: o})
}

// OccurrenceReplaced raises OccurrenceReplace (After) with the prior definition.
func (e *AssemblyEvents) OccurrenceReplaced(o *occurrence.Occurrence, previous occurrence.Definition) {
	event.Emit(e.bus, event.After, OccurrenceReplace{Occurrence: o, Previous: previous})
}

// OccurrenceTransformed raises OccurrenceTransform (After) with the prior placement.
func (e *AssemblyEvents) OccurrenceTransformed(o *occurrence.Occurrence, previous math.Matrix4) {
	event.Emit(e.bus, event.After, OccurrenceTransform{Occurrence: o, Previous: previous})
}

// OccurrenceSuppressionChanged raises OccurrenceSuppress (After) with the new state.
func (e *AssemblyEvents) OccurrenceSuppressionChanged(o *occurrence.Occurrence) {
	event.Emit(e.bus, event.After, OccurrenceSuppress{Occurrence: o, Suppressed: o.Suppressed()})
}

// AssemblyVetoError reports that a Before-phase handler cancelled an assembly
// operation. Callers distinguish it from real failures via errors.As.
type AssemblyVetoError struct {
	Operation string
	Reason    string
}

// Error implements error.
func (e *AssemblyVetoError) Error() string {
	return fmt.Sprintf("compdef: %s vetoed: %s", e.Operation, e.Reason)
}

// raiseBefore emits ev in the Before phase and returns an [AssemblyVetoError] if any
// handler vetoed the operation, or nil to proceed. It mirrors doc.vetoed for the
// assembly domain.
func raiseBefore[E event.Event](bus *event.Bus, operation string, ev E) error {
	if out := event.Emit(bus, event.Before, ev); out.Vetoed() {
		return &AssemblyVetoError{Operation: operation, Reason: out.Reason}
	}
	return nil
}
