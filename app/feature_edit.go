// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/param"
)

// Editing a placed feature's parameters — Inventor's double-click-to-edit on a browser
// feature node. Double-clicking re-opens the feature as the session's "feature edit";
// the head shows a dialog bound to its parameters (today an extrude's distance), and OK
// recomputes the part so the change flows through the feature history. Cancel restores
// the values captured when the edit opened. Only extrude is editable so far — other
// feature kinds open no edit (BeginEditFeature is a no-op for them).

// featureEditState holds the feature being edited and a snapshot of its parameters, so
// Cancel can restore them. It is non-nil exactly while an edit dialog is open.
type featureEditState struct {
	feature *feature.PartFeature
	origDir float64 // original extent distance (database units), for Cancel
	origOp  ops.PartFeatureOperation
}

// BeginEditFeature re-opens an extrude feature for parameter editing (the browser
// double-click path), snapshotting its distance and operation so Cancel can restore
// them. A non-extrude feature is not editable yet, so this is a no-op for it.
func (s *Session) BeginEditFeature(h FeatureHandle) {
	ext, ok := h.Feature.Definition().(*feature.ExtrudeFeature)
	if !ok {
		return
	}
	s.featureEdit = &featureEditState{feature: h.Feature, origDir: ext.DistanceValue(), origOp: ext.Operation()}
}

// IsEditingFeature reports whether a feature edit dialog should be open.
func (s *Session) IsEditingFeature() bool { return s.featureEdit != nil }

// EditingFeatureName returns the name of the feature being edited (the dialog title),
// or "" when none is being edited.
func (s *Session) EditingFeatureName() string {
	if s.featureEdit == nil {
		return ""
	}
	return s.featureEdit.feature.Name()
}

// editingExtrude returns the extrude being edited, or false when no edit is open.
func (s *Session) editingExtrude() (*feature.ExtrudeFeature, bool) {
	if s.featureEdit == nil {
		return nil, false
	}
	ext, ok := s.featureEdit.feature.Definition().(*feature.ExtrudeFeature)
	return ext, ok
}

// EditFeatureDistanceDisplay returns the edited extrude's distance in the document's
// length unit (the value the dialog field shows), or 0 when no extrude is being edited.
func (s *Session) EditFeatureDistanceDisplay() float64 {
	ext, ok := s.editingExtrude()
	if !ok {
		return 0
	}
	return s.DocumentUnits().ToPreferred(param.Q(ext.DistanceValue(), param.Length))
}

// SetEditFeatureDistanceDisplay sets the edited extrude's distance from a value given in
// the document's length unit (e.g. 25 mm). The change is applied on CommitFeatureEdit.
func (s *Session) SetEditFeatureDistanceDisplay(value float64) {
	if ext, ok := s.editingExtrude(); ok {
		ext.SetDistance(s.DocumentUnits().FromPreferred(value, param.Length).Value)
	}
}

// CommitFeatureEdit recomputes the part with the edited parameters and closes the edit.
// A sick result (e.g. a zero distance) keeps the edit open by returning an error so the
// dialog can stay up for correction — mirroring the extrude tool's commit.
func (s *Session) CommitFeatureEdit() error {
	if s.featureEdit == nil {
		return errors.New("app: no feature is being edited")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	pf := s.featureEdit.feature
	part.Features().MarkDirty(pf)
	part.Recompute()
	s.recordEdit(part, "Edit "+pf.Name())
	if !pf.Health().OK() {
		return errors.New("feature edit: " + pf.Health().Reason)
	}
	s.featureEdit = nil
	return nil
}

// CancelFeatureEdit restores the feature's snapshotted parameters, recomputes, and
// closes the edit — so an aborted edit leaves the part exactly as it was.
func (s *Session) CancelFeatureEdit() {
	if s.featureEdit == nil {
		return
	}
	if ext, ok := s.editingExtrude(); ok {
		ext.SetDistance(s.featureEdit.origDir)
		ext.SetOperation(s.featureEdit.origOp)
	}
	if part, err := activePart(s); err == nil {
		part.Features().MarkDirty(s.featureEdit.feature)
		part.Recompute()
	}
	s.featureEdit = nil
}
