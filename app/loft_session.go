// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Loft tool's property window (mirrors the Revolve/Coil bridges).

// ActiveLoft returns the running Loft tool, or nil when the active tool is not a loft
// (or there is none).
func (s *Session) ActiveLoft() *LoftTool {
	if s.tool == nil {
		return nil
	}
	l, _ := s.tool.tool.(*LoftTool)
	return l
}
