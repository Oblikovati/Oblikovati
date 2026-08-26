//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
)

// gridScan writes a small ASCII scan: a flat grid of points with one raised peak, so the placed
// work point on the peak is easy to spot.
func gridScan(t *testing.T) string {
	t.Helper()
	var body string
	for ix := -6; ix <= 6; ix++ {
		for iy := -6; iy <= 6; iy++ {
			body += fmt.Sprintf("%d %d 0\n", ix, iy)
		}
	}
	body += "0 0 6\n" // a single raised point (the peak) to snap the work point onto
	path := filepath.Join(t.TempDir(), "grid.xyz")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path
}

// TestInWindowSnapWorkPointRenders attaches a scan, selects its raised peak point, places a work
// point there, and captures the live viewport — the visual confirmation that Work Point anchors a
// datum on the as-built scan data (#645). Skips without a display.
func TestInWindowSnapWorkPointRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, _, err := s.AttachPointCloud("Grid", gridScan(t))
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	peak := math.P3(0, 0, 6)
	s.Select(app.PointCloudPointHandle{Cloud: pc, Point: peak})
	wp, err := s.CreateWorkPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("create work point: %v", err)
	}
	t.Logf("placed work point at %v on the snapped peak", wp.Point())

	frameCameraOn(s, pc.RangeBox())
	for range 10 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-snap-workpoint.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
