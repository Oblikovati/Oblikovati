//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// TestInWindowUITabDraws renders the Preferences UI tab in a real frame so the slider draw paths
// (drawUITab → editUITextScale/editUIIconScale) are exercised; the slider apply branches are
// covered by the app-level SetUIFontScale/SetUIIconScale tests.
func TestInWindowUITabDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := app.NewSession()

	win.BeginFrame()
	if native.Begin("##ui-tab-test") {
		drawUITab(s)
	}
	native.End()
	win.EndFrame(0.1, 0.1, 0.1)

	// A no-click render leaves the scales at their defaults.
	if s.UIFontScale() != 1 || s.UIIconScale() != 1 {
		t.Errorf("UI scales drifted without interaction: font=%v icon=%v", s.UIFontScale(), s.UIIconScale())
	}
}
