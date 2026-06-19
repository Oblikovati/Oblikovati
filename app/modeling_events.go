// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// Modeling/sketch event type ids (#148): the granular feature-lifecycle and sketch-edit
// notifications beyond the batched model.changed (0x08xx = modeling).
const (
	tidFeatureLifecycle event.TypeID = 0x0801
	tidSketchEdit       event.TypeID = 0x0802
)

// FeatureOp is which feature-lifecycle transition occurred — added, edited, or deleted.
type FeatureOp uint8

const (
	FeatureAdded FeatureOp = iota
	FeatureEdited
	FeatureDeleted
)

// FeatureLifecycleChanged announces that a feature was created, edited, or deleted on a document
// (#148). Op says which transition; Feature/Name/Kind carry the affected feature's identity so a
// relay forwards it without diffing the model tree.
type FeatureLifecycleChanged struct {
	Document doc.ID
	Op       FeatureOp
	Feature  uint64
	Name     string
	Kind     string
}

// EventID implements event.Event.
func (FeatureLifecycleChanged) EventID() event.TypeID { return tidFeatureLifecycle }

// SketchEditChanged announces that the host entered or left a sketch's edit mode (#148). Entered
// distinguishes the two; Sketch/Name identify the sketch. Fires for UI- and add-in-driven edits.
type SketchEditChanged struct {
	Document doc.ID
	Entered  bool
	Sketch   uint64
	Name     string
}

// EventID implements event.Event.
func (SketchEditChanged) EventID() event.TypeID { return tidSketchEdit }

// emitSketchEdit publishes a sketch-edit transition (#148) keyed to the active document, so the
// 2D and 3D enter/exit paths share one emit. A missing active document yields a zero id.
func (s *Session) emitSketchEdit(seq uint64, name string, entered bool) {
	var d doc.ID
	if active := s.ActiveDocument(); active != nil {
		d = active.ID()
	}
	event.Emit(s.bus, event.After, SketchEditChanged{Document: d, Entered: entered, Sketch: seq, Name: name})
}
