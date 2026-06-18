// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/event"

// Representation / model-state change events on the session bus (#901): emitted when a
// representation is activated or captured, or a model state is activated, so an add-in subscribing
// to the representations surface is notified of the change (the request/response methods were
// already handled; these are the missing change notifications). 0x0bxx = M11 assembly representations.
const (
	tidRepresentationActivated event.TypeID = 0x0b01
	tidRepresentationCaptured  event.TypeID = 0x0b02
	tidModelStateActivated     event.TypeID = 0x0b03
)

// RepresentationActivated announces that a representation became active. Kind is the representation
// family ("design", "positional", or "lod"); Name is the representation's name.
type RepresentationActivated struct {
	Kind string
	Name string
}

// EventID implements event.Event.
func (RepresentationActivated) EventID() event.TypeID { return tidRepresentationActivated }

// RepresentationCaptured announces that a new representation was captured (same Kind/Name shape).
type RepresentationCaptured struct {
	Kind string
	Name string
}

// EventID implements event.Event.
func (RepresentationCaptured) EventID() event.TypeID { return tidRepresentationCaptured }

// ModelStateActivated announces that a model state became active.
type ModelStateActivated struct {
	Name string
}

// EventID implements event.Event.
func (ModelStateActivated) EventID() event.TypeID { return tidModelStateActivated }
