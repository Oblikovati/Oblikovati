// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Sweep tool's property window (mirrors the other feature bridges).

// ActiveSweep returns the running Sweep tool, or nil when the active tool is not a sweep
// (or there is none).
func (s *Session) ActiveSweep() *SweepTool {
	return s.activeTool[*SweepTool]()
}
