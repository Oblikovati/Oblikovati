//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// screenshotPath is where Tools ▸ Save Viewport PNG writes the current viewport image. A fixed
// path (overwritten each time) so the latest capture is always at a known location for
// inspection. screenshotRequested is set by the menu and serviced after the viewport renders.
const screenshotPath = "/tmp/oblikovati-viewport.png"

var screenshotRequested bool

// serviceScreenshot writes the viewport PNG if Tools ▸ Save Viewport PNG was clicked. Called at
// the end of drawViewportPanel — AFTER the viewport has rendered this frame, so the offscreen
// image readback reflects exactly what is on screen.
func serviceScreenshot(win *native.Window, s *app.Session) {
	if !screenshotRequested {
		return
	}
	screenshotRequested = false
	if err := win.SaveViewportPNG(screenshotPath); err != nil {
		fileNotice(s, "Save Viewport PNG failed: %v", err)
		return
	}
	fileNotice(s, "Saved viewport image to %s", screenshotPath)
}
