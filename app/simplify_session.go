// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Simplify tool's property window.

// ActiveSimplify returns the running Simplify tool, or nil when the active tool is not a
// simplify (or there is none).
func (s *Session) ActiveSimplify() *SimplifyTool {
	return s.activeTool[*SimplifyTool]()
}
