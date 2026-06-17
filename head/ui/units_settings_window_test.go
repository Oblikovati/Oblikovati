//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestInWindowUnitsSettingsDraws opens the real window with the Document Settings ▸
// Units dialog visible and runs frames — exercising the unit combos, precision
// inputs and format controls without tripping Dear ImGui's Begin/End assertions
// (#146). It skips cleanly where no display/Vulkan is available.
func TestInWindowUnitsSettingsDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	s.OpenUnitsSettings()
	defer s.CloseUnitsSettings()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if !s.UnitsSettingsOpen() {
		t.Error("Units settings dialog should stay open across frames")
	}
}
