//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// styledRibbonButton builds a minimal ribbon button of the given style for layout tests.
func styledRibbonButton(id string, style app.ButtonStyle) app.RibbonButton {
	cmd := app.NewCommand(id, id, "Test", func(*app.Session) error { return nil }).
		WithButtonStyle(style)
	return app.RibbonButton{Command: cmd, Enabled: true}
}

// TestPackPanelColumns locks the ribbon's column-major flow: small/compact buttons
// stack ribbonMaxRows deep, and a large button always stands alone as its own column —
// even when it interrupts a partially filled stack.
func TestPackPanelColumns(t *testing.T) {
	buttons := []app.RibbonButton{
		styledRibbonButton("s1", app.SmallIconButton),
		styledRibbonButton("s2", app.SmallIconButton),
		styledRibbonButton("L1", app.LargeIconButton), // interrupts the 2-deep stack
		styledRibbonButton("c1", app.CompactIconButton),
		styledRibbonButton("c2", app.CompactIconButton),
		styledRibbonButton("c3", app.CompactIconButton),
		styledRibbonButton("s3", app.SmallIconButton), // overflow starts a new column
	}
	cols := packPanelColumns(buttons)
	wantLens := []int{2, 1, 3, 1}
	if len(cols) != len(wantLens) {
		t.Fatalf("packPanelColumns returned %d columns, want %d", len(cols), len(wantLens))
	}
	for i, want := range wantLens {
		if len(cols[i]) != want {
			t.Errorf("column %d has %d buttons, want %d", i, len(cols[i]), want)
		}
	}
	if got := cols[1][0].Command.ID(); got != "L1" {
		t.Errorf("column 1 holds %q, want the large button \"L1\" alone", got)
	}
}

// TestPackPanelColumnsEmpty guards the degenerate panel (no buttons → no columns).
func TestPackPanelColumnsEmpty(t *testing.T) {
	if cols := packPanelColumns(nil); len(cols) != 0 {
		t.Errorf("packPanelColumns(nil) = %d columns, want 0", len(cols))
	}
}
