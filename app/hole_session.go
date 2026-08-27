// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Hole tool's property window (mirrors the other feature bridges).

// ActiveHole returns the running Hole tool, or nil when the active tool is not a hole
// (or there is none).
func (s *Session) ActiveHole() *HoleTool {
	return s.activeTool[*HoleTool]()
}
