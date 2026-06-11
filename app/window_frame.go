// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/api/types"

// WindowFrameStatus is the host window's live state (M05-F10): the head reports it
// each frame so windows.listFrames serves real geometry without the app touching
// GLFW (the head owns the window; the session only mirrors it).
type WindowFrameStatus struct {
	Caption string
	State   types.WindowState
	Width   int
	Height  int
}

// SetWindowFrameStatus mirrors the head window's current state into the session.
func (s *Session) SetWindowFrameStatus(status WindowFrameStatus) { s.windowFrame = status }

// WindowFrameStatus returns the mirrored host-window state. A headless session
// (tests, CLI) reports a zero frame — callers present what exists.
func (s *Session) WindowFrameStatus() WindowFrameStatus { return s.windowFrame }
