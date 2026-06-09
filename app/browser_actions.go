// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Session actions invoked from the model browser's right-click menu. Each mutates the
// active part and (when the change affects geometry) recomputes it, then clears the
// selection — the deleted/edited node no longer exists at its old identity. They are
// the undo-friendly seam between the menu (app/browser_menu.go) and the model.

// EditSketch opens an existing sketch for editing (the menu's "Edit Sketch"). It errors
// when a sketch is already being edited, since Inventor forbids nesting sketch edits.
func (s *Session) EditSketch(sk *sketch.Sketch) error {
	if sk == nil {
		return errors.New("app: EditSketch with nil sketch")
	}
	if s.activeSketch != nil {
		return errors.New("app: already editing a sketch (finish it first)")
	}
	s.EnterSketch(sk)
	return nil
}

// DeleteSketch removes a sketch from the active part. If it is the one being edited the
// environment is exited first so the editor never points at a freed sketch.
func (s *Session) DeleteSketch(sk *sketch.Sketch) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if sk == nil {
		return errors.New("app: DeleteSketch with nil sketch")
	}
	if s.activeSketch == sk {
		s.ExitSketch()
	}
	if !part.Sketches().Remove(sk.ID()) {
		return errors.New("app: DeleteSketch: sketch not in active part")
	}
	s.selection.Clear()
	part.Recompute()
	s.recordEdit(part, "Delete Sketch")
	return nil
}

// DeleteFeature removes a feature from the active part's history and recomputes so
// downstream geometry reflects its absence.
func (s *Session) DeleteFeature(f *feature.PartFeature) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if f == nil {
		return errors.New("app: DeleteFeature with nil feature")
	}
	if !part.Features().Remove(f.ID()) {
		return errors.New("app: DeleteFeature: feature not in active part")
	}
	s.selection.Clear()
	part.Recompute()
	s.recordEdit(part, "Delete Feature")
	return nil
}

// ToggleFeatureSuppressed flips a feature's explicit suppression and recomputes, so the
// part rebuilds with or without that feature's contribution.
func (s *Session) ToggleFeatureSuppressed(f *feature.PartFeature) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if f == nil {
		return errors.New("app: ToggleFeatureSuppressed with nil feature")
	}
	f.SetSuppressed(!f.Suppressed())
	part.Recompute()
	s.recordEdit(part, "Suppress Feature")
	return nil
}
