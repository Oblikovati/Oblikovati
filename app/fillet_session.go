// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Fillet tool's property window.

// ActiveFillet returns the running Fillet tool, or nil when the active tool is not a fillet
// (or there is none).
func (s *Session) ActiveFillet() *FilletTool {
	return s.activeTool[*FilletTool]()
}
