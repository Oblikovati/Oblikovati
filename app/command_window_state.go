// SPDX-License-Identifier: GPL-2.0-only

package app

// Command Window visibility (M26 F04). The docked command line is on by default — the
// user chose an always-on panel — so the flag is stored inverted: the Session zero value
// is "open", and SetCommandWindowOpen(false) hides it.

// CommandWindowOpen reports whether the docked Command Window panel is shown.
func (s *Session) CommandWindowOpen() bool { return !s.commandWindowHidden }

// SetCommandWindowOpen shows (true) or hides (false) the Command Window panel.
func (s *Session) SetCommandWindowOpen(open bool) { s.commandWindowHidden = !open }

// ToggleCommandWindow flips the Command Window's visibility — the View/Tools menu item and
// its hotkey call this.
func (s *Session) ToggleCommandWindow() { s.commandWindowHidden = !s.commandWindowHidden }
