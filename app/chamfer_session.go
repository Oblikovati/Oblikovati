// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Chamfer tool's property window.

// ActiveChamfer returns the running Chamfer tool, or nil when the active tool is not a
// chamfer (or there is none).
func (s *Session) ActiveChamfer() *ChamferTool {
	if s.tool == nil {
		return nil
	}
	c, _ := s.tool.tool.(*ChamferTool)
	return c
}
