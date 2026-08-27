// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Coil tool's property window (mirrors the Revolve bridge).

// ActiveCoil returns the running Coil tool, or nil when the active tool is not a coil
// (or there is none).
func (s *Session) ActiveCoil() *CoilTool {
	return s.activeTool[*CoilTool]()
}
