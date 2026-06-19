// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// emitFeatureLifecycle publishes a granular feature-lifecycle notification (#148) on the session
// bus, so the add-in event relay forwards feature.added/edited/deleted. v1 scope is the host method
// router (features.add/edit/delete), matching edit.committed (ADR-0004) — the batched model.changed
// remains the coarse signal that also covers UI-driven paths.
func emitFeatureLifecycle(s *app.Session, op app.FeatureOp, f *feature.PartFeature) {
	if f == nil {
		return
	}
	var docID doc.ID
	if active := s.ActiveDocument(); active != nil {
		docID = active.ID()
	}
	event.Emit(s.Events(), event.After, app.FeatureLifecycleChanged{
		Document: docID, Op: op, Feature: uint64(f.ID()), Name: f.Name(), Kind: f.Kind(),
	})
}

// lastPartFeature returns the active part's most recently added feature, or nil — the feature a
// successful features.add just appended (every part-feature op appends one feature at the end).
func lastPartFeature(s *app.Session) *feature.PartFeature {
	part, err := modelaccess.ActivePart(s)
	if err != nil || part == nil {
		return nil
	}
	feats := part.Features()
	if feats.Count() == 0 {
		return nil
	}
	return feats.Item(feats.Count() - 1)
}
