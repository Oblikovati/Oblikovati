// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
	"oblikovati.org/model/sketch"
)

// errNoActiveDocument is returned when a document-addressed request targets the active document but
// none is open.
var errNoActiveDocument = errors.New("app: no active document")

// Per-document sketch settings (#147): the constraint-inference defaults persist with the document
// (model/doc stores them, persistence round-trips them in the .obk). These session methods are the
// in-proc seam the head and the addin/router both drive, addressing a document by its session id
// (0 ⇒ the active document). The sketch tools read the ACTIVE document's settings through
// SketchInferenceOptions, so each open part keeps its own inference behaviour.

// DocumentSketchSettings returns the sketch settings of the document with the given session id
// (0 ⇒ the active document), or an error when no such document is open.
func (s *Session) DocumentSketchSettings(id uint64) (types.SketchSettings, error) {
	d, err := s.documentForSettings(id)
	if err != nil {
		return types.SketchSettings{}, err
	}
	return d.SketchSettings(), nil
}

// SetDocumentSketchSettings stores the sketch settings on the document with the given session id
// (0 ⇒ the active document) and returns the stored value, or an error when no such document is open.
// A change to the active document is recorded as an undo step so it rides the metadata snapshot like
// body names (S6 #1641).
func (s *Session) SetDocumentSketchSettings(id uint64, settings types.SketchSettings) (types.SketchSettings, error) {
	d, err := s.documentForSettings(id)
	if err != nil {
		return types.SketchSettings{}, err
	}
	active := s.ActiveDocument()
	if id == 0 && active != nil {
		s.beginMetadataEdit(active)
	}
	d.SetSketchSettings(settings)
	if id == 0 && active != nil {
		s.recordMetadataEdit(active, "Sketch Settings")
	}
	return d.SketchSettings(), nil
}

// documentForSettings resolves a settings request's target document: the addressed one by session id,
// or the active document when id is 0.
func (s *Session) documentForSettings(id uint64) (sketchSettingsHolder, error) {
	if id != 0 {
		return s.DocumentByID(id)
	}
	if d := s.ActiveDocument(); d != nil {
		return d, nil
	}
	return nil, errNoActiveDocument
}

// sketchSettingsHolder is the document capability the per-document sketch settings need — the
// model.doc.Document accessors. Stated as an interface so the session methods stay testable.
type sketchSettingsHolder interface {
	SketchSettings() types.SketchSettings
	SetSketchSettings(types.SketchSettings)
}

// sketchInferenceFrom maps the persisted document settings onto the kernel's inference options that
// the sketch tools consume (the fields are 1:1; the priority enum is shared).
func sketchInferenceFrom(s types.SketchSettings) sketch.InferenceOptions {
	return sketch.InferenceOptions{
		InferEnabled:     s.InferConstraints,
		ConstrainEnabled: s.AutoApplyConstraints,
		Priority:         s.ConstraintPriority,
	}
}

// sketchSettingsFrom maps the kernel inference options back onto the persisted document settings.
func sketchSettingsFrom(o sketch.InferenceOptions) types.SketchSettings {
	return types.SketchSettings{
		InferConstraints:     o.InferEnabled,
		AutoApplyConstraints: o.ConstrainEnabled,
		ConstraintPriority:   o.Priority,
	}
}
