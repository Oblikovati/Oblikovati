// SPDX-License-Identifier: GPL-2.0-only

package app

// ShowStatusBar reports whether the bottom status bar is visible. The status bar
// mirrors the active command prompt and defaults to visible for Inventor parity.
func (s *Session) ShowStatusBar() bool {
	return s.appOptions.UI.ShowStatusBar
}

// SetShowStatusBar persists the status-bar visibility toggle.
func (s *Session) SetShowStatusBar(v bool) error {
	s.appOptions.UI.ShowStatusBar = v
	return s.saveOptions()
}
