// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Full Round Fillet tool's property window.

// ActiveFullRoundFillet returns the running Full Round Fillet tool, or nil when the active tool is
// not a full round (or there is none).
func (s *Session) ActiveFullRoundFillet() *FullRoundFilletTool {
	if s.tool == nil {
		return nil
	}
	f, _ := s.tool.tool.(*FullRoundFilletTool)
	return f
}
