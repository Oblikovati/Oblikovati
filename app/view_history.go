// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/scene"

// View history backs Previous View (F5, Inventor's N14): each discrete view change — Home,
// a standard orientation, Zoom All, a named-view restore — records the camera it replaced, so
// F5 steps back through them. Continuous orbit/pan/zoom drags are NOT recorded (they would flood
// the history); the snapshot is taken once, before the discrete jump.

// maxViewHistory bounds the recorded views so a long session does not grow unboundedly.
const maxViewHistory = 64

// viewHistory is the bounded stack of cameras Previous View pops.
type viewHistory struct {
	stack []scene.Camera
}

// PushViewHistory records the current camera so PreviousView can return to it. The view-change
// commands call it before they move the camera.
func (s *Session) PushViewHistory() {
	s.viewHistory.stack = append(s.viewHistory.stack, s.camera)
	if len(s.viewHistory.stack) > maxViewHistory {
		s.viewHistory.stack = s.viewHistory.stack[len(s.viewHistory.stack)-maxViewHistory:]
	}
}

// PreviousView restores the most recently recorded view and removes it from the history (F5). It
// keeps the current viewport pixel size, and is a no-op when nothing has been recorded.
func (s *Session) PreviousView() {
	n := len(s.viewHistory.stack)
	if n == 0 {
		return
	}
	cam := s.viewHistory.stack[n-1]
	s.viewHistory.stack = s.viewHistory.stack[:n-1]
	cam.Width, cam.Height = s.camera.Width, s.camera.Height
	s.SetCamera(cam)
}

// ViewHistoryDepth returns how many views Previous View can step back through (for tests/UI).
func (s *Session) ViewHistoryDepth() int { return len(s.viewHistory.stack) }
