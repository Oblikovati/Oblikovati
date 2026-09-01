// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestPopupControlResolvesRegistryItems checks a PopupControl's ribbon button lists
// its item commands (resolved live from the registry), skipping ids that don't exist
// — the CommandBarPopUp behavior of M05-F03.
func TestPopupControlResolvesRegistryItems(t *testing.T) {
	t.Parallel()
	s := NewSession() // no document open ⇒ the ZeroDoc ribbon, so commands target it
	noop := func(*Session) error { return nil }
	if err := s.Commands().Add(NewCommand("x.a", "Alpha", "Demo", noop).WithTooltip("first").WithRibbons(ZeroDocRibbon)); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Commands().Add(NewCommand("x.b", "Beta", "Demo", noop).WithRibbons(ZeroDocRibbon)); err != nil {
		t.Fatalf("add: %v", err)
	}
	popup := NewCommand("x.menu", "Tools Menu", "Demo", noop).WithRibbons(ZeroDocRibbon).
		WithPopupItems("x.a", "x.ghost", "x.b")
	if err := s.Commands().Add(popup); err != nil {
		t.Fatalf("add popup: %v", err)
	}
	if popup.Kind() != PopupControl {
		t.Fatalf("WithPopupItems should set PopupControl, got %v", popup.Kind())
	}

	panel, ok := BuildRibbon(s).Panel("Demo")
	if !ok {
		t.Fatal("Demo panel missing from ribbon")
	}
	var menu *RibbonButton
	for i := range panel.Buttons {
		if panel.Buttons[i].Command.ID() == "x.menu" {
			menu = &panel.Buttons[i]
		}
	}
	if menu == nil {
		t.Fatal("popup button missing from panel")
	}
	if len(menu.Variants) != 2 || menu.Variants[0].CommandID != "x.a" || menu.Variants[1].CommandID != "x.b" {
		t.Fatalf("popup items = %+v, want x.a and x.b (ghost skipped)", menu.Variants)
	}
	if menu.Variants[0].Tooltip != "first" || !menu.Variants[0].Enabled {
		t.Errorf("item[0] = %+v, want tooltip and live enabled state carried", menu.Variants[0])
	}
}
