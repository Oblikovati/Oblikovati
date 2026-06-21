//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// TestInWindowPrivacyTabDraws renders the Preferences Privacy tab in a real frame so the
// telemetry opt-out toggle's draw path is exercised (#1182). It asserts the default stays on
// when no click is injected; the toggle behavior itself is covered by the app-level
// SetTelemetryEnabled test. Skips when no display/Vulkan is available (e.g. some CI lanes).
func TestInWindowPrivacyTabDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := app.NewSession()

	win.BeginFrame()
	if native.Begin("##privacy-tab-test") {
		drawPrivacyTab(s)
	}
	native.End()
	win.EndFrame(0.1, 0.1, 0.1)

	if !s.TelemetryEnabled() {
		t.Error("telemetry must remain on by default when no opt-out click is injected")
	}
}

// TestInWindowPreferencesWindowOpensWithPrivacyTab drives the full Preferences window (the tab
// wiring, including the new Privacy tab) for a few real frames, so drawPreferencesWindow's
// branch that mounts the Privacy tab is exercised. It is a smoke render — it asserts no panic
// and that the window state flag round-trips.
func TestInWindowPreferencesWindowOpensWithPrivacyTab(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := app.NewSession()

	showPreferences = true
	defer func() { showPreferences = false }()
	for i := 0; i < 3; i++ { // a few frames so the tab bar settles
		win.BeginFrame()
		drawPreferencesWindow(s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if !showPreferences {
		t.Error("Preferences window should still be open")
	}
}
