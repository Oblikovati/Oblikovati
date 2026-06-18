// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Face Fillet tool's property window.

// ActiveFaceFillet returns the running Face Fillet tool, or nil when the active tool is not a
// face fillet (or there is none).
func (s *Session) ActiveFaceFillet() *FaceFilletTool {
	if s.tool == nil {
		return nil
	}
	f, _ := s.tool.tool.(*FaceFilletTool)
	return f
}
