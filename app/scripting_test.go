// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestScriptConsoleCommandToggles checks the Manage ▸ Script Console command opens and
// closes the console panel and reports its open state (drives the head panel each frame).
func TestScriptConsoleCommandToggles(t *testing.T) {
	s := registeredSession(t)
	if s.ScriptConsoleOpen() {
		t.Fatal("console should start closed")
	}
	if err := s.Execute("Manage.ScriptConsole"); err != nil {
		t.Fatalf("execute Manage.ScriptConsole: %v", err)
	}
	cmd, _ := s.Commands().ByID("Manage.ScriptConsole")
	if !s.ScriptConsoleOpen() || !cmd.IsActive(s) {
		t.Errorf("console should be open and command active after toggle")
	}
	if err := s.Execute("Manage.ScriptConsole"); err != nil {
		t.Fatalf("re-toggle: %v", err)
	}
	if s.ScriptConsoleOpen() {
		t.Errorf("console should be closed after second toggle")
	}
}

// TestScriptConsoleOnManageTab checks the Script Console lands on the Manage tab's Scripts
// panel (the ribbon exposes the control).
func TestScriptConsoleOnManageTab(t *testing.T) {
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab("Manage")
	if !ok {
		t.Fatal("no Manage tab")
	}
	if _, ok := tab.Panel("Scripts"); !ok {
		t.Errorf("Manage tab missing the Scripts panel")
	}
}
