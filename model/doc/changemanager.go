// SPDX-License-Identifier: GPL-2.0-only

package doc

import "oblikovati.org/event"

// ChangeKind classifies a model change so a processor can react selectively.
type ChangeKind uint8

const (
	ObjectAdded ChangeKind = iota
	ObjectModified
	ObjectDeleted
	ObjectRenamed
	ObjectSuppressed
)

// ChangeDefinition describes one change to the model: what kind, and which object
// (by label, later by reference key). It is the unit a ChangeProcessor inspects.
type ChangeDefinition struct {
	Kind        ChangeKind
	ObjectLabel string
}

// ModelChanged is the modeling event carrying a batch of changes committed to a
// document — the ModelingEvents surface. The ChangeManager dispatches it to
// registered processors; a Before handler may veto the change.
type ModelChanged struct {
	Document *Document
	Changes  []ChangeDefinition
}

// EventID implements event.Event.
func (ModelChanged) EventID() event.TypeID { return tidModelChanged }

// ChangeProcessor is an add-in or subsystem that reacts to model changes — the
// modernized ChangeProcessor. It is invoked (After commit) for every change batch
// while it is enabled; the command + event primitives compose into this, so no
// separate framework is needed (architecture core/06).
type ChangeProcessor interface {
	Name() string
	ProcessChange(ModelChanged) error
}

// ChangeManager registers change processors and dispatches committed model changes
// to them. It listens on the workspace bus for [ModelChanged] (After), so issuing
// a model change and reacting to it both go through the same event primitives.
type ChangeManager struct {
	bus   *event.Bus
	sub   event.Subscription
	procs []*Registration
}

// NewChangeManager creates a manager bound to bus and subscribes it to model
// changes.
func NewChangeManager(bus *event.Bus) *ChangeManager {
	m := &ChangeManager{bus: bus}
	m.sub = event.Subscribe(bus, event.After, func(_ event.Context, e ModelChanged) event.Outcome {
		m.dispatch(e)
		return event.Continue()
	})
	return m
}

// Registration is the process-control handle for one registered processor: it can
// be enabled, disabled, or removed without re-registering (the
// ChangeManagerProcessControl).
type Registration struct {
	processor ChangeProcessor
	enabled   bool
	manager   *ChangeManager
}

// Register adds a processor (enabled) and returns its control handle.
func (m *ChangeManager) Register(p ChangeProcessor) *Registration {
	reg := &Registration{processor: p, enabled: true, manager: m}
	m.procs = append(m.procs, reg)
	return reg
}

// Processor returns the registered processor.
func (r *Registration) Processor() ChangeProcessor { return r.processor }

// Enabled reports whether the processor currently receives changes.
func (r *Registration) Enabled() bool { return r.enabled }

// SetEnabled turns dispatch to this processor on or off.
func (r *Registration) SetEnabled(on bool) { r.enabled = on }

// Unregister removes the processor from its manager, reporting whether it was found.
func (r *Registration) Unregister() bool {
	for i, reg := range r.manager.procs {
		if reg == r {
			r.manager.procs = append(r.manager.procs[:i], r.manager.procs[i+1:]...)
			return true
		}
	}
	return false
}

// dispatch invokes every enabled processor with the change batch, collecting the
// first error so one failing processor does not stop the others.
func (m *ChangeManager) dispatch(e ModelChanged) {
	for _, reg := range m.procs {
		if reg.enabled {
			_ = reg.processor.ProcessChange(e)
		}
	}
}

// Close detaches the manager from the bus.
func (m *ChangeManager) Close() { m.sub.Cancel() }
