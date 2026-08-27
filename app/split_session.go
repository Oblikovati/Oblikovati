// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Split tool's property window (mirrors the other feature bridges).

// ActiveSplit returns the running Split tool, or nil when the active tool is not a split.
func (s *Session) ActiveSplit() *SplitTool {
	return s.activeTool[*SplitTool]()
}
