//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestInWindowNamedViewsPanelDraws opens the real window with the Named Views panel visible and
// a saved view, then runs frames — so a mismatched ImGui Begin/End or a bad widget call in the
// name field / per-view Restore/Delete rows would trip Dear ImGui's assertions (M16-F03 #404).
func TestInWindowNamedViewsPanelDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	if _, err := s.CaptureNamedView("Iso"); err != nil {
		t.Fatalf("CaptureNamedView: %v", err)
	}
	s.OpenNamedViewsPanel()
	defer s.CloseNamedViewsPanel()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if len(s.NamedViews()) != 1 {
		t.Errorf("named views = %d, want 1 during panel draw", len(s.NamedViews()))
	}
}
