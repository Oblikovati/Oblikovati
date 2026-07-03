// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// Metadata-mutation events (S10, #1644): renames and property changes (suppression, sketch
// settings) previously fired nothing, so add-ins mirroring the document went stale and had to
// poll. Like the feature-lifecycle events (#1085) these are emitted at the SESSION seam — the
// shared verb both the UI and the wire router call — so an interactive rename and a body.rename
// call alike reach the relay. 0x08xx = the modeling event block (see modeling_events.go).
const (
	tidObjectRenamed   event.TypeID = 0x0803
	tidPropertyChanged event.TypeID = 0x0804
)

// ObjectRenamed announces that a document object (body/sketch/feature/occurrence/document) was
// renamed. Kind is the object kind, Key its stable reference key, and Old/New the prior and new
// names — enough for a relay to forward object.renamed without re-querying the model.
type ObjectRenamed struct {
	Document doc.ID
	Kind     types.ObjectKind
	Key      string
	OldName  string
	NewName  string
}

// EventID implements event.Event.
func (ObjectRenamed) EventID() event.TypeID { return tidObjectRenamed }

// PropertyChanged announces that a document object's property changed (e.g. suppression toggled, a
// sketch setting edited). Property names the property; Old/New are its values as strings so one
// event carries any property.
type PropertyChanged struct {
	Document doc.ID
	Kind     types.ObjectKind
	Key      string
	Property string
	OldValue string
	NewValue string
}

// EventID implements event.Event.
func (PropertyChanged) EventID() event.TypeID { return tidPropertyChanged }

// activeDocID returns the active document's id, or the zero id when none is active — the shared
// keying every metadata emit uses.
func (s *Session) activeDocID() doc.ID {
	if active := s.ActiveDocument(); active != nil {
		return active.ID()
	}
	return 0
}

// emitObjectRenamed publishes an object.renamed notification on the session bus, keyed to the
// active document, so the add-in relay forwards it for UI- and wire-driven renames alike (#1644).
func (s *Session) emitObjectRenamed(kind types.ObjectKind, key, oldName, newName string) {
	event.Emit(s.bus, event.After, ObjectRenamed{
		Document: s.activeDocID(), Kind: kind, Key: key, OldName: oldName, NewName: newName,
	})
}

// emitPropertyChanged publishes a property.changed notification on the session bus, keyed to the
// active document (#1644).
func (s *Session) emitPropertyChanged(kind types.ObjectKind, key, property, oldVal, newVal string) {
	event.Emit(s.bus, event.After, PropertyChanged{
		Document: s.activeDocID(), Kind: kind, Key: key, Property: property, OldValue: oldVal, NewValue: newVal,
	})
}

// featureMetaKey is a feature's stable reference key for a metadata event — its session id, the
// same identity the feature-lifecycle events carry, rendered as a string for the generic Key field.
func featureMetaKey(f *feature.PartFeature) string {
	return strconv.FormatUint(uint64(f.ID()), 10)
}

// boolProperty renders a bool property value for a [PropertyChanged] event ("true"/"false").
func boolProperty(v bool) string { return strconv.FormatBool(v) }
