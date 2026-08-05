// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// UnwrapTool is the interactive Unwrap command: activate it, click a cylindrical face, and OK to
// append its flat development — the face unrolled to a rectangle of arc length × axial height —
// as a sheet body beside the solid.
//
// UnwrapDefinition and ModifyFeatures.AddUnwrap were implemented and routed over the API, but no
// ribbon command, tool, dialog or menu entry referenced them: a case-insensitive search for
// "unwrap" across app/commands_*.go and head/ui/ returned zero hits (#2047).
type UnwrapTool struct {
	face  *FaceHandle
	added *feature.PartFeature
}

// NewUnwrapTool returns an unwrap tool awaiting a face pick.
func NewUnwrapTool() *UnwrapTool { return &UnwrapTool{} }

// Name implements [Tool].
func (t *UnwrapTool) Name() string { return "Unwrap" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *UnwrapTool) Start(*Session) {}

// AcceptedKinds declares unwrap picks the face to flatten.
func (t *UnwrapTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked face for the unified highlight.
func (t *UnwrapTool) Picks() []Selectable {
	if t.face == nil {
		return nil
	}
	return []Selectable{*t.face}
}

// Pick records the face to flatten, replacing any previous pick — the feature flattens exactly
// one face, so a second click means "that one instead".
func (t *UnwrapTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		t.face = &f
	}
}

// FacePicked reports whether the face is chosen, so the dialog knows what to prompt for.
func (t *UnwrapTool) FacePicked() bool { return t.face != nil }

// CanCommit reports whether a face is picked.
func (t *UnwrapTool) CanCommit() bool { return t.face != nil }

// Commit appends the flattened patch. A sick feature — a face that is not a cylinder, or a
// degenerate one — keeps the tool open by returning an error rather than leaving a dead node.
func (t *UnwrapTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addUnwrap(part.Features())
	if t.added == nil {
		return errors.New("unwrap: no face picked")
	}
	part.Recompute()
	s.recordEdit(part, "Unwrap")
	if !t.added.Health().OK() {
		return errors.New("unwrap: " + t.added.Health().Reason)
	}
	return nil
}

// addUnwrap builds the unwrap feature into engine fs — shared by Commit and the preview.
func (t *UnwrapTool) addUnwrap(fs *feature.PartFeatures) *feature.PartFeature {
	if t.face == nil {
		return nil
	}
	return feature.NewModifyFeatures(fs).AddUnwrap(t.face.Face.ReferenceKey())
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *UnwrapTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature satisfies DraftPreviewable so the commit gate has a draft to inspect (#1626).
func (t *UnwrapTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addUnwrap(fs), nil
	})
}

// Prompt guides the user through the unwrap step.
func (t *UnwrapTool) Prompt(*Session) string {
	if t.face == nil {
		return "Click the cylindrical face to flatten"
	}
	return "Click OK to append the flat development"
}

// Cancel is a no-op; the engine restores the ambient filter.
func (t *UnwrapTool) Cancel(*Session) {}

// ClearFace drops the picked face — the property panel's selector clear (⊗).
func (t *UnwrapTool) ClearFace() { t.face = nil }
