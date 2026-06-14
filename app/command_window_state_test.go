// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestCommandWindowOpenByDefault(t *testing.T) {
	if !NewSession().CommandWindowOpen() {
		t.Error("Command Window should be open by default (always-on panel)")
	}
}

func TestCommandWindowSetAndToggle(t *testing.T) {
	s := NewSession()
	s.SetCommandWindowOpen(false)
	if s.CommandWindowOpen() {
		t.Error("CommandWindowOpen should be false after SetCommandWindowOpen(false)")
	}
	s.ToggleCommandWindow()
	if !s.CommandWindowOpen() {
		t.Error("ToggleCommandWindow should reopen the panel")
	}
	s.ToggleCommandWindow()
	if s.CommandWindowOpen() {
		t.Error("ToggleCommandWindow should hide the panel again")
	}
}
