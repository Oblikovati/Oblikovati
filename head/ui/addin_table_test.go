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
