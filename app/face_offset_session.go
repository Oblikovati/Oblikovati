// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Offset Face tool's property window.

// ActiveFaceOffset returns the running Offset Face tool, or nil when the active tool is not
// a face offset (or there is none).
func (s *Session) ActiveFaceOffset() *FaceOffsetTool {
	if s.tool == nil {
		return nil
	}
	o, _ := s.tool.tool.(*FaceOffsetTool)
	return o
}
