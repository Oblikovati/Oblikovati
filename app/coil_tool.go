// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// CoilTool is the interactive Coil command: activate it, click a sketch region, choose
// the helix axis, pitch, number of revolutions and operation in the property window,
// and OK to add a coil feature to the active part. It mirrors [RevolveTool].
type CoilTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed coil (see editCoilTool)
	profile         *ProfileHandle
	axis            feature.WorkRef
	pitch           float64
	revolutions     float64
	pitchRows       []feature.CoilPitchRow
	startEnd        feature.CoilEndCondition
	endEnd          feature.CoilEndCondition
	operation       ops.PartFeatureOperation
	added           *feature.PartFeature
}

// NewCoilTool returns a coil tool defaulting to a single-pitch, 3-revolution helix about
// the Y origin axis that creates a new body.
func NewCoilTool() *CoilTool {
	return &CoilTool{axis: feature.OriginYAxis, pitch: 1, revolutions: 3, operation: ops.NewBody}
}

// Name implements [Tool].
func (t *CoilTool) Name() string { return "Coil" }

// Start sets the selection filter to profiles so clicks pick a region.
func (t *CoilTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures the region the user clicked.
func (t *CoilTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		pc := p
		t.profile = &pc
	}
}

// The options the property window drives: the helix axis, pitch, revolutions, operation.
func (t *CoilTool) SetAxis(ref feature.WorkRef)              { t.axis = ref }
func (t *CoilTool) Axis() feature.WorkRef                    { return t.axis }
func (t *CoilTool) SetPitch(p float64)                       { t.pitch = p }
func (t *CoilTool) Pitch() float64                           { return t.pitch }
func (t *CoilTool) SetRevolutions(r float64)                 { t.revolutions = r }
func (t *CoilTool) Revolutions() float64                     { return t.revolutions }
func (t *CoilTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *CoilTool) Operation() ops.PartFeatureOperation      { return t.operation }

// PickedProfile returns the picked region (and true), or false when none picked yet.
func (t *CoilTool) PickedProfile() (ProfileHandle, bool) {
	if t.profile == nil {
		return ProfileHandle{}, false
	}
	return *t.profile, true
}

// CanCommit reports whether a region is picked and the revolutions are positive.
func (t *CoilTool) CanCommit() bool { return t.profile != nil && t.revolutions > 0 }

// SetPitchRows installs the variable-pitch row table (nil restores the
// constant pitch); PitchRows returns it (M06-F09, #624).
func (t *CoilTool) SetPitchRows(rows []feature.CoilPitchRow) {
	t.pitchRows = append([]feature.CoilPitchRow(nil), rows...)
}
func (t *CoilTool) PitchRows() []feature.CoilPitchRow {
	return append([]feature.CoilPitchRow(nil), t.pitchRows...)
}

// SetEndConditions / StartEnd / EndEnd manage the flat-end treatments.
func (t *CoilTool) SetEndConditions(start, end feature.CoilEndCondition) {
	t.startEnd, t.endEnd = start, end
}
func (t *CoilTool) StartEnd() feature.CoilEndCondition { return t.startEnd }
func (t *CoilTool) EndEnd() feature.CoilEndCondition   { return t.endEnd }

// applyVariableRail copies the panel's rail extras into a definition.
func (t *CoilTool) applyVariableRail(def *feature.CoilDefinition) {
	def.PitchRows = append([]feature.CoilPitchRow(nil), t.pitchRows...)
	def.StartEnd, def.EndEnd = t.startEnd, t.endEnd
}

// Commit adds the coil feature to the active part and recomputes; a sick feature keeps
// the tool open by returning an error.
func (t *CoilTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.added, err = t.addCoil(part, part.Features()); err != nil {
		return err
	}
	part.Recompute()
	s.recordEdit(part, "Coil")
	if !t.added.Health().OK() {
		return errors.New("coil: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// commitEdit writes the panel state back into the committed coil's definition — the
// same properties the create path passes to Add.
func (t *CoilTool) commitEdit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	axis, ok := part.WorkGeometry().AxisByRef(t.axis)
	if !ok {
		return errors.New("coil edit: axis " + string(t.axis) + " not found")
	}
	def := t.target.Definition().(*feature.CoilFeature).Definition()
	def.Sketch, def.ProfileIndex, def.Axis = t.profile.Sketch, t.profile.ProfileIndex, axis
	def.Pitch, def.Revolutions = konst(t.pitch), konst(t.revolutions)
	def.Operation = t.operation
	t.applyVariableRail(def)
	return commitFeatureEdit(s, t.target)
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *CoilTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the coil steps.
func (t *CoilTool) Prompt(*Session) string {
	if t.profile == nil {
		return "Select a region to coil"
	}
	return "Set the axis, pitch and revolutions, then click OK"
}

// addCoil resolves the helix axis and builds the coil feature (including the variable-rail
// extras) into engine fs — the shared constructor used by both Commit (the part's engine) and
// DraftFeature (a scratch engine), so the preview matches the committed result.
func (t *CoilTool) addCoil(part *compdef.PartComponentDefinition, fs *feature.PartFeatures) (*feature.PartFeature, error) {
	axis, ok := part.WorkGeometry().AxisByRef(t.axis)
	if !ok {
		return nil, errors.New("coil: axis " + string(t.axis) + " not found")
	}
	pitch, revs := t.pitch, t.revolutions
	pf := feature.NewCoilFeatures(fs).Add(t.profile.Sketch, t.profile.ProfileIndex, axis,
		func() float64 { return pitch }, func() float64 { return revs }, 0, t.operation)
	t.applyVariableRail(pf.Definition().(*feature.CoilFeature).Definition())
	return pf, nil
}

// DraftFeature returns the unattached coil feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addCoil the commit uses. Empty until a
// region is picked and revolutions are set.
func (t *CoilTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	part, err := activePart(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addCoil(part, fs)
	})
}

// Cancel restores the default selection filter.
func (t *CoilTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
	s.Selection().SetFilter(NewSelectionFilter())
}

// ClearProfile empties the picked profile — the property panel's selector clear (⊗) —
// returning the tool to its select-a-region step.
func (t *CoilTool) ClearProfile() { t.profile = nil }

// SourceSketchName returns the sketch the picked profile comes from, for the property
// panel's breadcrumb; "" until a profile is picked.
func (t *CoilTool) SourceSketchName() string {
	if t.profile == nil {
		return ""
	}
	return t.profile.Sketch.Name()
}
