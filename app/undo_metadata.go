// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
	"oblikovati.org/model/doc"
)

// Document metadata (body names, body color styles, sketch settings, display settings) persists to
// the .obk but does not live in the parametric recipe, so before M40 audit S6/S5 (#1641/#1640) it
// bypassed the undo stream entirely: renaming a body or recoloring it marked the document dirty but
// Ctrl+Z could not touch it, and undoing a geometry step restored an old recipe while the newer
// metadata silently survived — a recipe/metadata hybrid the user never created.
//
// documentMetaStore closes that gap by folding the metadata INTO the same snapshot as the recipe:
// one atomic snapshot per cursor position, so recipe and metadata can never diverge. It wraps a
// document's recipe store and marshals both halves together; the [recipeStore] machinery
// (SnapshotLog, RecipeEvent, the no-op-delta check) stores the composite bytes unchanged.
type documentMetaStore struct {
	inner recipeStore
	doc   *doc.Document
}

var _ recipeStore = documentMetaStore{}

// docMetaSnapshot is the JSON envelope: the inner recipe bytes plus the document metadata. Empty
// fields omit, so a document that never touched metadata produces the same information as before,
// just nested under "recipe" — the format change is internal to the undo stream, never the file.
type docMetaSnapshot struct {
	Recipe          json.RawMessage       `json:"recipe"`
	BodyNames       map[string]string     `json:"bodyNames,omitempty"`
	BodyColorStyles map[string]string     `json:"bodyColorStyles,omitempty"`
	SketchSettings  *types.SketchSettings `json:"sketchSettings,omitempty"`
	DisplaySettings *display.Settings     `json:"displaySettings,omitempty"`
}

// MarshalSnapshot captures the recipe and the document's current metadata as one blob.
func (m documentMetaStore) MarshalSnapshot() ([]byte, error) {
	recipe, err := m.inner.MarshalSnapshot()
	if err != nil {
		return nil, err
	}
	snap := docMetaSnapshot{
		Recipe:          recipe,
		BodyNames:       m.doc.BodyNames(),
		BodyColorStyles: m.doc.BodyColorStyles(),
	}
	if m.doc.SketchSettingsSet() {
		ss := m.doc.SketchSettings()
		snap.SketchSettings = &ss
	}
	if ds, ok := m.doc.DisplaySettings(); ok {
		snap.DisplaySettings = &ds
	}
	return json.Marshal(snap)
}

// RestoreSnapshot restores the recipe first, then reinstalls the metadata through the document's
// Restore* setters (which do not re-dirty), so undo/redo moves both halves in lockstep.
func (m documentMetaStore) RestoreSnapshot(b []byte) error {
	var snap docMetaSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	if err := m.inner.RestoreSnapshot(snap.Recipe); err != nil {
		return err
	}
	m.doc.RestoreBodyNames(snap.BodyNames)
	m.doc.RestoreBodyColorStyles(snap.BodyColorStyles)
	restoreDocSettings(m.doc, snap)
	return nil
}

// restoreDocSettings reinstalls the pointer-optional sketch/display settings, clearing them back to
// "unset" when the snapshot carries none (so undo past the first SetSketchSettings truly reverts).
func restoreDocSettings(d *doc.Document, snap docMetaSnapshot) {
	if snap.SketchSettings != nil {
		d.RestoreSketchSettings(*snap.SketchSettings)
	} else {
		d.ClearSketchSettings()
	}
	if snap.DisplaySettings != nil {
		d.RestoreDisplaySettings(*snap.DisplaySettings)
	} else {
		d.ClearDisplaySettings()
	}
}

// metaStoreFor wraps a document's recipe content so undo snapshots carry its metadata too; false when
// the content is not a recipe store (a non-recipe document has no undo stream).
func metaStoreFor(d *doc.Document) (recipeStore, bool) {
	inner, ok := d.Content().(recipeStore)
	if !ok {
		return nil, false
	}
	return documentMetaStore{inner: inner, doc: d}, true
}

// beginMetadataEdit captures the pre-edit baseline before a UI-driven metadata mutation on the active
// document, so the mutation records a real delta even when it is the document's first recorded edit
// (the router does the same via EnsureActiveEditBaseline before a wire method). A no-op on a
// non-active document, whose per-document stream this seam does not own.
func (s *Session) beginMetadataEdit(d *doc.Document) {
	if d != nil && d == s.ActiveDocument() {
		s.EnsureActiveEditBaseline()
	}
}

// recordMetadataEdit records a metadata mutation of d as one undo step, but only when d is the active
// document — the per-document undo stream records against the active document, exactly as the recipe
// stream does. A metadata change to a background document is not undoable, the same scope limit the
// recipe stream already carries.
func (s *Session) recordMetadataEdit(d *doc.Document, label string) {
	if d == nil || d != s.ActiveDocument() {
		return
	}
	s.RecordActiveEdit(label)
}
