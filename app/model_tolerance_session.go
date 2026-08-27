// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the model-tolerance tool's property window.

// ActiveModelTolerance returns the running Feature Control Frame / Datum Feature tool, or nil
// when the active tool is neither (or there is none).
func (s *Session) ActiveModelTolerance() *ModelToleranceTool {
	return s.activeTool[*ModelToleranceTool]()
}
