// SPDX-License-Identifier: GPL-2.0-only

package app_test

import (
	"testing"

	"oblikovati.org/app"
)

// TestRibbonDropdownEntriesAreExecutable is the structural "exposed-but-unwired UI" guard for the
// id-dispatched parts of the ribbon (#1468 investigation). A split-button head exposes VARIANTS in
// its dropdown; clicking one routes through Session.Execute(variantID) → CommandManager.ByID. But
// WithVariants only FLAGS its entries (isVariant=true) — it does not register them — so a variant
// that was attached but never added to the registry is a visible dropdown row whose click silently
// fails ByID. Likewise a PopupControl lists other commands by string id and silently drops any that
// do not resolve. This asserts every dropdown entry resolves to a registered, executable command, so
// an unwired entry fails CI instead of being a dead control the user discovers.
//
// (Plain ribbon buttons and selector options are built FROM the registry — their command is a live
// *CommandDefinition — so they cannot dangle. Menu-bar items call session verbs directly and are
// compile-checked. On-canvas overlay buttons hold direct func pointers, so their gap is input
// plumbing, not a missing id — covered by the Navigation Bar click test in head/ui.)
func TestRibbonDropdownEntriesAreExecutable(t *testing.T) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	cmds := s.Commands()
	variants := 0
	for _, c := range cmds.All() {
		for _, v := range c.Variants() {
			variants++
			if _, ok := cmds.ByID(v.ID()); !ok {
				t.Errorf("command %q exposes variant %q in its dropdown, but it is not registered — a "+
					"click routes through Execute→ByID and silently fails (exposed-but-unwired UI)", c.ID(), v.ID())
			}
		}
		if c.Kind() == app.PopupControl {
			for _, id := range c.PopupItems() {
				if _, ok := cmds.ByID(id); !ok {
					t.Errorf("popup command %q lists item id %q that resolves to no registered command — "+
						"the flyout entry is silently dropped (exposed-but-unwired UI)", c.ID(), id)
				}
			}
		}
	}
	if variants == 0 {
		t.Fatal("no command variants found — the guard is checking nothing; has the registry changed?")
	}
}
