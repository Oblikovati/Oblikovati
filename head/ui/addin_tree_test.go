// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// treeFirstUse must report true the first time a control id is seen and false thereafter, so the
// Expanded seed only applies on first render (afterwards imgui owns open/closed state).
func TestTreeFirstUse(t *testing.T) {
	delete(treeSeeded, "w/catalog")
	if !treeFirstUse("w/catalog") {
		t.Fatal("first call = false, want true")
	}
	if treeFirstUse("w/catalog") {
		t.Fatal("second call = true, want false")
	}
}
