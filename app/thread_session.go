// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Thread tool's property window.

// ActiveThread returns the running Thread tool, or nil when the active tool is not a thread.
func (s *Session) ActiveThread() *ThreadTool {
	return s.activeTool[*ThreadTool]()
}
