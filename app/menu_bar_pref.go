// SPDX-License-Identifier: GPL-2.0-only

package app

// ShowMenuBar reports whether the legacy File/Edit/View/Tools/Help menu bar is
// visible. It defaults to visible so File and Preferences remain discoverable.
func (s *Session) ShowMenuBar() bool {
	return s.appOptions.UI.ShowMenuBar
}

// SetShowMenuBar persists the menu-bar visibility toggle.
func (s *Session) SetShowMenuBar(v bool) error {
	s.appOptions.UI.ShowMenuBar = v
	return s.saveOptions()
}
