// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
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

// EmitFeatureLifecycle publishes a granular feature-lifecycle notification (#1085) on the
// session bus, keyed to the active document, so the add-in event relay forwards
// feature.added/edited/deleted for UI- and add-in-driven feature ops alike (lifting #148's
// router-only v1 scope). It is the single emit seam the session-level feature ops and the host
// method router both call. A nil feature is a no-op — a producer tool that built nothing.
func (s *Session) EmitFeatureLifecycle(op FeatureOp, f *feature.PartFeature) {
	if f == nil {
		return
	}
	var d doc.ID
	if active := s.ActiveDocument(); active != nil {
		d = active.ID()
	}
	event.Emit(s.bus, event.After, FeatureLifecycleChanged{
		Document: d, Op: op, Feature: uint64(f.ID()), Name: f.Name(), Kind: f.Kind(),
	})
}

// featureProducer is implemented by the tools that create a part feature; AddedFeature returns
// the feature they just appended (nil before commit). The tool-commit seam (Session.OK) reads it
// to fire featureAdded for UI-driven creation (#1085), so every add-tool emits without each one
// wiring the event itself.
type featureProducer interface {
	AddedFeature() *feature.PartFeature
}
