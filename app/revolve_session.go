// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Revolve tool's property window: the head reads/sets the axis
// and angle through the tool without touching its internals, mirroring the Extrude
// bridge.

// ActiveRevolve returns the running Revolve tool, or nil when the active tool is not a
// revolve (or there is none).
func (s *Session) ActiveRevolve() *RevolveTool {
	return s.activeTool[*RevolveTool]()
}
