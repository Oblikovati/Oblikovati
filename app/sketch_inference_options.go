// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/sketch"

// Session-level sketch inference preferences (M06-F10, #625): whether point
// snapping and constraint auto-application run while sketching, and which
// constraint family wins. Stored per session (the settings surface of #147
// will lift them into persisted preferences when it lands).

// SketchInferenceOptions returns the session's inference configuration,
// defaulting on first read.
func (s *Session) SketchInferenceOptions() sketch.InferenceOptions {
	if s.sketchInference == nil {
		opts := sketch.DefaultInferenceOptions()
		s.sketchInference = &opts
	}
	return *s.sketchInference
}

// SetSketchInferenceOptions replaces the session's inference configuration.
func (s *Session) SetSketchInferenceOptions(opts sketch.InferenceOptions) {
	s.sketchInference = &opts
}
