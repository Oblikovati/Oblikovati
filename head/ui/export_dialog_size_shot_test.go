//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"
)

// TestInWindowExportDialogFitsAtRaisedFontScale is the live confirmation for #1753: at a raised UI
// font scale the Export file dialog opens SCALED with the text, and its file table now fills the
// window minus a reserved footer — so the Export/Cancel row is pinned in view instead of falling off
// the bottom (the reported clip). It renders real DrawChrome frames with the global Export modal armed
// and saves a PNG to eyeball; the footer reserve scales with the font, so the guard below holds at any
// scale. Skips cleanly without display/Vulkan.
func TestInWindowExportDialogFitsAtRaisedFontScale(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	defer func() { uiFontScale = 1.0 }() // package global; leave it as other tests expect

	s := framedSession()
	if err := s.SetUIFontScale(1.6); err != nil { // the hi-dpi cohort the clip bit hardest
		t.Fatalf("SetUIFontScale: %v", err)
	}
	fileModal.openFor(dialogExport) // arm the global Export modal that DrawChrome renders
	defer fileModal.cancel()

	for i := 0; i < 6; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	if !fileModal.isOpen() {
		t.Fatal("the Export modal should still be open after rendering")
	}
	// The reserved footer scales with the (raised) font, so the layout always leaves room below the
	// table for the Export/Cancel row — the buttons can no longer be pushed off an undersized window.
	if r := fileFooterReserve(); r <= 6*15 { // 6 frame-heights must have grown past the 1.0-scale ~15px baseline
		t.Errorf("footer reserve = %v; expected it to scale up with the raised font", r)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "export-dialog-fontscale.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
