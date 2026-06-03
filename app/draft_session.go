// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Draft tool's property window.

// ActiveDraft returns the running Draft tool, or nil when the active tool is not a draft
// (or there is none).
func (s *Session) ActiveDraft() *DraftTool {
	if s.tool == nil {
		return nil
	}
	d, _ := s.tool.tool.(*DraftTool)
	return d
}
