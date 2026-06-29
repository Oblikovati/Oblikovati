// SPDX-License-Identifier: GPL-2.0-only

package app

// WantsContinuousRedraw reports whether the session has time-driven work in flight that the
// head must keep rendering frame-after-frame even with no user input: a running camera tween
// or joint-drive playback (both advanced by per-frame delta time), or an in-progress bug
// report (a capture→submit state machine the loop services each frame). The render-on-demand
// loop (#1493) sleeps when this is false and the UI is otherwise idle; it must consult this
// so an animation in progress never freezes mid-transition. Centralize new time-driven
// states here so the idle loop keeps covering them.
//
// Example: for !win.ShouldClose() { drawFrame(); if session.WantsContinuousRedraw() { tick() } else { block() } }
func (s *Session) WantsContinuousRedraw() bool {
	return s.CameraAnimating() || s.DriveAnimating() || s.BugReportInProgress()
}
