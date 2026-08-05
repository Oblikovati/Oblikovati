// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Unwrap tool's property window.

// ActiveUnwrap returns the running Unwrap tool, or nil when the active tool is not an unwrap
// (or there is none).
func (s *Session) ActiveUnwrap() *UnwrapTool {
	if s.tool == nil {
		return nil
	}
	t, _ := s.tool.tool.(*UnwrapTool)
	return t
}
