//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestDockablePanelRegistryMembership pins the set of built-in dockable windows that #1473 routes
// through the shared closable path + View menu: every panel the bug named (Materials, Preferences)
// plus the rest, each with a non-empty View-menu label and a unique title.
func TestDockablePanelRegistryMembership(t *testing.T) {
	want := []string{
		"Model", "Parameters", "Materials", "Lighting", "Color Styles", "Display Settings",
		"Document Settings — Units", "Named Views", "Bill of Materials", "History Browser",
		"Selection Filter", "Command", "Preferences",
	}
	byTitle := map[string]*dockablePanel{}
	for i := range dockablePanels {
		p := &dockablePanels[i]
		if _, dup := byTitle[p.title]; dup {
			t.Fatalf("duplicate dockable panel title %q", p.title)
		}
		byTitle[p.title] = p
		if panelMenuLabel(p) == "" {
			t.Errorf("panel %q has an empty View-menu label", p.title)
		}
		if p.isOpen == nil || p.setOpen == nil || p.draw == nil {
			t.Errorf("panel %q is missing an isOpen/setOpen/draw hook", p.title)
		}
	}
	for _, title := range want {
		if byTitle[title] == nil {
			t.Errorf("expected dockable panel %q to be registered", title)
		}
	}
	// The docked REPL is captioned "Command" but must read as "Command Window" in the menu, and the
	// model tree reads as "Model Browser" — the menuLabel override path.
	if got := panelMenuLabel(byTitle["Command"]); got != "Command Window" {
		t.Errorf("Command menu label = %q, want %q", got, "Command Window")
	}
	if got := panelMenuLabel(byTitle["Model"]); got != "Model Browser" {
		t.Errorf("Model menu label = %q, want %q", got, "Model Browser")
	}
}

// TestDockablePanelOpenCloseRoundTrips verifies every panel's isOpen/setOpen pair is consistent —
// the contract the View-menu toggle and the close-'X' routing both rely on. A panel whose setOpen
// did not actually flip its backing store would show a stale check mark or refuse to close.
func TestDockablePanelOpenCloseRoundTrips(t *testing.T) {
	// Snapshot the head-local visibility globals so this test leaves no residue for others.
	defer func(b, m, p bool) { showBrowser, showMaterials, showPreferences = b, m, p }(showBrowser, showMaterials, showPreferences)

	s := app.NewSession()
	for i := range dockablePanels {
		p := &dockablePanels[i]
		p.setOpen(s, true)
		if !p.isOpen(s) {
			t.Errorf("panel %q: setOpen(true) left it closed", p.title)
		}
		p.setOpen(s, false)
		if p.isOpen(s) {
			t.Errorf("panel %q: setOpen(false) left it open", p.title)
		}
	}
}
