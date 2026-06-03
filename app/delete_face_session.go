// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Delete Face tool's property window.

// ActiveDeleteFace returns the running Delete Face tool, or nil when the active tool is not
// a delete-face (or there is none).
func (s *Session) ActiveDeleteFace() *DeleteFaceTool {
	if s.tool == nil {
		return nil
	}
	d, _ := s.tool.tool.(*DeleteFaceTool)
	return d
}
