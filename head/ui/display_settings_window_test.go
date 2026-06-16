//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestColorFromVec converts an ImGui rgba vector to an opaque-override color, clamping.
func TestColorFromVec(t *testing.T) {
	c := colorFromVec([4]float32{1, 0, 0.5, 1})
	if c.R != 255 || c.G != 0 || c.B != 128 || c.Opacity != 1 {
		t.Errorf("colorFromVec = %+v, want (255,0,128) opaque", c)
	}
	if toByte(-1) != 0 || toByte(2) != 255 {
		t.Errorf("toByte clamp wrong: %d %d", toByte(-1), toByte(2))
	}
}

// TestInWindowDisplaySettingsPanelDraws opens the real window with the Display Settings dialog
// visible and runs frames — exercising the color editors / checkboxes without tripping Dear
// ImGui's Begin/End assertions (M16-F07 #643).
func TestInWindowDisplaySettingsPanelDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	s.OpenDisplaySettings()
	defer s.CloseDisplaySettings()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if !s.DisplaySettingsOpen() {
		t.Error("Display Settings dialog should stay open across frames")
	}
}
