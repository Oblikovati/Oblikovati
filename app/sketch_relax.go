// SPDX-License-Identifier: GPL-2.0-only

package app

// Relax Mode (#791): a session-wide toggle, exposed on the status bar and persisted across
// sessions, that lets a left-drag move fully- or over-constrained sketch geometry. When on,
// BeginEntityDrag no longer refuses dimensioned geometry (see sketch_drag.go) and
// UpdateEntityDrag re-solves through Sketch.RelaxSolve, which relaxes the driving dimensions
// to follow the cursor instead of holding the geometry rigid. It is purely an interaction
// mode — it changes nothing about how a sketch solves outside of a drag.

// RelaxMode reports whether Relax Mode is active — the status-bar toggle's state and the
// predicate UpdateEntityDrag uses to pick the relaxing solve.
func (s *Session) RelaxMode() bool { return s.relaxMode }

// SetRelaxMode turns Relax Mode on or off and persists the choice so it survives across
// sessions (Inventor keeps the status-bar toggle sticky). Persisting a no-op change is
// harmless; the store write is skipped when the state is unchanged.
func (s *Session) SetRelaxMode(on bool) {
	if s.relaxMode == on {
		return
	}
	s.relaxMode = on
	s.appOptions.Sketch.RelaxMode = on
	_ = s.saveOptions()
}

// ToggleRelaxMode flips Relax Mode and returns the new state — the status-bar button's action.
func (s *Session) ToggleRelaxMode() bool {
	s.SetRelaxMode(!s.relaxMode)
	return s.relaxMode
}
