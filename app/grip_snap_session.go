// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Grip Snap tool's Move-Options panel.

// ActiveGripSnap returns the running Grip Snap tool, or nil when the active tool is not a grip snap
// (or there is none).
func (s *Session) ActiveGripSnap() *GripSnapTool {
	return s.activeTool[*GripSnapTool]()
}
