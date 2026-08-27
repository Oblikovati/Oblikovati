// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Replace Face tool's property window.

// ActiveReplaceFace returns the running Replace Face tool, or nil when the active tool is not
// a replace-face (or there is none).
func (s *Session) ActiveReplaceFace() *ReplaceFaceTool {
	return s.activeTool[*ReplaceFaceTool]()
}
