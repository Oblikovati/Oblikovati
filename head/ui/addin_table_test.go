// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/wire"
)

// cellAt must tolerate a row shorter than the column count (a ragged member row) by returning ""
// rather than panicking, so a malformed catalog row can't crash the head.
func TestCellAt(t *testing.T) {
	row := wire.TableRow{Key: "k", Cells: []string{"10", "30"}}
	if got := cellAt(row, 1); got != "30" {
		t.Fatalf("cellAt col1 = %q, want 30", got)
	}
	if got := cellAt(row, 5); got != "" {
		t.Fatalf("cellAt out-of-range = %q, want empty", got)
	}
}

// TestTableSelectionChanged is the #1933 regression: a PanelTable must scroll a pre-selected row
// into view ONCE per programmatic selection change and NOT every frame (an unconditional scroll
// would fight the user's own scrolling). tableSelectionChanged gates the SetScrollHereY call, so it
// must return true the first time a value is seen, false while that value is unchanged, and true
// again when the add-in selects a different row.
func TestTableSelectionChanged(t *testing.T) {
	const win, ctl = "test-window-1933", "members"
	delete(tableScrolledValue, win+"/"+ctl) // isolate from other tests sharing the package map

	if !tableSelectionChanged(win, ctl, "row20") {
		t.Fatal("first selection of row20: want true (scroll it into view)")
	}
	if tableSelectionChanged(win, ctl, "row20") {
		t.Error("same selection on the next frame: want false (do not fight the user's scrolling)")
	}
	if tableSelectionChanged(win, ctl, "row20") {
		t.Error("same selection again: want false (still no re-scroll)")
	}
	if !tableSelectionChanged(win, ctl, "row35") {
		t.Error("add-in changed the selection to row35: want true (scroll the new row into view)")
	}
	if tableSelectionChanged(win, ctl, "row35") {
		t.Error("row35 unchanged next frame: want false")
	}
	// A different control keeps its own last-scrolled value, so one table's selection does not
	// suppress another's scroll.
	if !tableSelectionChanged(win, "other-control", "row35") {
		t.Error("a different control selecting row35: want true (independent scroll state)")
	}
}
