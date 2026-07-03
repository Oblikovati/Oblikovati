//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/pointcloud"
)

// newShotWindow opens a viewport window and resets the per-window dock/icon state, the common
// preamble of the point-cloud in-window shot tests. The caller defers win.Destroy().
func newShotWindow(t *testing.T) *native.Window {
	win := newViewportWindow(t)
	dockLaidOut = false
	icons = nil
	return win
}

// attachSheetWithFitPlane attaches the tilted sheet scan and fits a visible work plane to it — the
// shared setup for the provenance / move / auto-recompute shot tests.
func attachSheetWithFitPlane(t *testing.T, s *app.Session) (*pointcloud.PointCloud, *feature.WorkPlane) {
	t.Helper()
	pc, _, err := s.AttachPointCloud("Sheet", tiltedSheetScan(t))
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	wp, _, err := s.CreatePointCloudPlane("Sheet")
	if err != nil {
		t.Fatalf("fit plane: %v", err)
	}
	wp.SetVisible(true)
	return pc, wp
}

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
