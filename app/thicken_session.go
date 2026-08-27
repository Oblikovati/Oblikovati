// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Thicken tool's property window.

// ActiveThicken returns the running Thicken tool, or nil when the active tool is not a
// thicken (or there is none).
func (s *Session) ActiveThicken() *ThickenTool {
	return s.activeTool[*ThickenTool]()
}
