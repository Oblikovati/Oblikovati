// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Editing a committed assembly machining feature (#766): the browser's Edit re-opens the feature
// as a tool whose Params() are its EditableParams (distance / radius / angle / depth …), shown in
// the generic tool-param dialog. Each setter writes through to the feature's closure-backed scalar,
// so OK recomputes the assembly with the new values and records an undo step; Cancel restores the
// values captured when editing opened. Suppress and Delete round out the feature-node menu.

// AssemblyFeatureEditTool edits one committed assembly feature's scalar parameters in place.
type AssemblyFeatureEditTool struct {
	dialogTool
	feature *compdef.AssemblyFeature
	params  []feature.EditableParam
	orig    []float64 // values captured at open, restored on Cancel
}

// NewAssemblyFeatureEditTool opens af for editing, snapshotting its current parameter values.
func NewAssemblyFeatureEditTool(af *compdef.AssemblyFeature) *AssemblyFeatureEditTool {
	t := &AssemblyFeatureEditTool{feature: af}
	if ed, ok := af.Definition().(feature.Editable); ok {
		t.params = ed.EditableParams()
	}
	t.orig = make([]float64, len(t.params))
	for i, p := range t.params {
		t.orig[i] = p.Get()
	}
	return t
}

func (t *AssemblyFeatureEditTool) Name() string { return "Edit " + t.feature.Name() }

// Start is a no-op: the editable parameters were snapshotted in the constructor, so the dialog
// can open straight away.

// Pick is unused — an assembly feature edit changes scalar parameters, not geometric references
// (re-picking participant edges/faces is a later refinement).

// CanCommit reports the feature has editable parameters to apply.
func (t *AssemblyFeatureEditTool) CanCommit() bool { return len(t.params) > 0 }

// Commit recomputes the assembly with the edited values (already written through by the dialog's
// setters) and records the undo step.
func (t *AssemblyFeatureEditTool) Commit(s *Session) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Edit "+t.feature.Kind())
	return nil
}

// Cancel restores the parameter values captured at open and recomputes, so an abandoned edit
// leaves no change.
func (t *AssemblyFeatureEditTool) Cancel(s *Session) {
	for i, p := range t.params {
		p.Set(t.orig[i])
	}
	if asm, err := activeAssembly(s); err == nil {
		asm.RecomputeFeatures()
	}
}

// Params exposes the feature's editable scalars to the generic dialog.
func (t *AssemblyFeatureEditTool) Params() ToolParams {
	floats := make([]FloatParam, len(t.params))
	for i, p := range t.params {
		floats[i] = FloatParam{Label: p.Label, Get: p.Get, Set: p.Set}
	}
	return ToolParams{Floats: floats}
}

// BeginEditAssemblyFeature opens the feature edit tool for the browser-selected assembly feature
// (the menu's Edit / a double-click). A feature with no editable parameters surfaces a notice
// instead of opening an empty dialog.
func (s *Session) BeginEditAssemblyFeature(h AssemblyFeatureHandle) {
	if h.Feature == nil {
		return
	}
	if !assemblyFeatureEditable(h.Feature) {
		s.notice = h.Feature.Name() + " has no editable parameters"
		return
	}
	s.StartTool(NewAssemblyFeatureEditTool(h.Feature))
}

// SuppressAssemblyFeature toggles a feature's suppression and recomputes (the program rebuilds with
// or without its contribution), recording an undo step.
func (s *Session) SuppressAssemblyFeature(af *compdef.AssemblyFeature, suppressed bool) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if af == nil {
		return errors.New("app: SuppressAssemblyFeature with nil feature")
	}
	if af.Suppressed() == suppressed {
		return nil
	}
	af.SetSuppressed(suppressed)
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Suppress Feature")
	return nil
}

// ToggleAssemblyFeatureSuppressed flips a feature's suppression (the browser's Suppress/Unsuppress).
func (s *Session) ToggleAssemblyFeatureSuppressed(af *compdef.AssemblyFeature) error {
	if af == nil {
		return errors.New("app: ToggleAssemblyFeatureSuppressed with nil feature")
	}
	return s.SuppressAssemblyFeature(af, !af.Suppressed())
}

// DeleteAssemblyFeature removes a feature from the program, clears the selection, recomputes, and
// records an undo step.
func (s *Session) DeleteAssemblyFeature(af *compdef.AssemblyFeature) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if af == nil {
		return errors.New("app: DeleteAssemblyFeature with nil feature")
	}
	if !asm.Features().Remove(af.ID()) {
		return errors.New("app: DeleteAssemblyFeature: feature not in this assembly")
	}
	s.selection.Clear()
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Delete Feature")
	return nil
}

// assemblyFeatureEditable reports whether af exposes editable scalar parameters.
func assemblyFeatureEditable(af *compdef.AssemblyFeature) bool {
	ed, ok := af.Definition().(feature.Editable)
	return ok && len(ed.EditableParams()) > 0
}
