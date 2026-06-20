// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerDocumentSettingsHandlers wires the per-document settings methods (#147). Starts with the
// Sketch tab — the constraint-inference defaults the sketch tools read.
func (r *Router) registerDocumentSettingsHandlers() {
	r.handlers[wire.MethodDocumentGetSketchSettings] = getDocumentSketchSettings
	r.handlers[wire.MethodDocumentSetSketchSettings] = setDocumentSketchSettings
}

// getDocumentSketchSettings returns a document's persisted sketch settings
// (wire.MethodDocumentGetSketchSettings).
func getDocumentSketchSettings(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.GetSketchSettingsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	set, err := s.DocumentSketchSettings(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.SketchSettingsResult{Settings: set})
}

// setDocumentSketchSettings stores a document's sketch settings and returns the stored value
// (wire.MethodDocumentSetSketchSettings).
func setDocumentSketchSettings(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetSketchSettingsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	set, err := s.SetDocumentSketchSettings(a.Document, a.Settings)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.SketchSettingsResult{Settings: set})
}
