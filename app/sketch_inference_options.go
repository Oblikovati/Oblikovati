// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/sketch"

// Sketch inference preferences (M06-F10, #625): whether point snapping and constraint
// auto-application run while sketching, and which constraint family wins. As of #147 these are
// PER-DOCUMENT (persisted in the .obk via doc.SketchSettings); the sketch tools read the active
// document's settings here. With no active document (headless paths, tests) the session keeps a
// fallback so the behaviour is still well-defined.

// SketchInferenceOptions returns the inference configuration the sketch tools should apply: the
// active document's persisted sketch settings, or the session fallback when none is active.
func (s *Session) SketchInferenceOptions() sketch.InferenceOptions {
	if d := s.ActiveDocument(); d != nil {
		return sketchInferenceFrom(d.SketchSettings())
	}
	if s.sketchInference == nil {
		opts := sketch.DefaultInferenceOptions()
		s.sketchInference = &opts
	}
	return *s.sketchInference
}

// SetSketchInferenceOptions stores the inference configuration on the active document (persisted,
// per-document, #147), or on the session fallback when none is active.
func (s *Session) SetSketchInferenceOptions(opts sketch.InferenceOptions) {
	if d := s.ActiveDocument(); d != nil {
		d.SetSketchSettings(sketchSettingsFrom(opts))
		return
	}
	s.sketchInference = &opts
}
