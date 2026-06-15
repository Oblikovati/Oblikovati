// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestOccurrenceEditOpensComponentTab checks the placed-component Edit path (#764): the
// occurrence context menu offers an enabled Edit that opens the component document in a
// visible, active tab.
func TestOccurrenceEditOpensComponentTab(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	occ := asm.Occurrences().Item(0)

	menu := occurrenceMenu(OccurrenceHandle{Occurrence: occ})
	edit, ok := findMenuItem(menu, "Edit")
	if !ok || !edit.Enabled {
		t.Fatalf("occurrence Edit item missing or disabled (ok=%v, %+v)", ok, edit)
	}
	if err := edit.Invoke(s); err != nil {
		t.Fatalf("Edit invoke: %v", err)
	}

	widget, found := s.Workspace().ByName("widget.obk")
	if !found {
		t.Fatal("widget.obk not in the workspace")
	}
	if s.ActiveDocument() != widget {
		t.Error("Edit did not open/activate the component document")
	}
	if !widget.Visible() {
		t.Error("the opened component document is not visible (no tab would show)")
	}
}
