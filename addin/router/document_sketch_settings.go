// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerDocumentSettingsHandlers wires the per-document settings methods (#147). Starts with the
// Sketch tab — the constraint-inference defaults the sketch tools read.
func (r *Router) registerDocumentSettingsHandlers() {
	r.readOnly(wire.MethodDocumentGetSketchSettings, typed(getDocumentSketchSettings))
	r.readOnly(wire.MethodDocumentSetSketchSettings, typed(setDocumentSketchSettings))
}

// getDocumentSketchSettings returns a document's persisted sketch settings
// (wire.MethodDocumentGetSketchSettings).
func getDocumentSketchSettings(s *app.Session, a wire.GetSketchSettingsArgs) (wire.SketchSettingsResult, error) {
	set, err := s.DocumentSketchSettings(a.Document)
	if err != nil {
		return wire.SketchSettingsResult{}, err
	}
	return wire.SketchSettingsResult{Settings: set}, nil
}

// setDocumentSketchSettings stores a document's sketch settings and returns the stored value
// (wire.MethodDocumentSetSketchSettings).
func setDocumentSketchSettings(s *app.Session, a wire.SetSketchSettingsArgs) (wire.SketchSettingsResult, error) {
	set, err := s.SetDocumentSketchSettings(a.Document, a.Settings)
	if err != nil {
		return wire.SketchSettingsResult{}, err
	}
	return wire.SketchSettingsResult{Settings: set}, nil
}
