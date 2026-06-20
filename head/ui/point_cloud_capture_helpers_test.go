//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
)

// captureFramed frames the camera on box, renders a few chrome frames, and saves the viewport to
// name.png — the shared tail of the point-cloud in-window shot tests.
func captureFramed(t *testing.T, win *native.Window, s *app.Session, box math.Box, name string) {
	t.Helper()
	frameCameraOn(s, box)
	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), name+".png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
