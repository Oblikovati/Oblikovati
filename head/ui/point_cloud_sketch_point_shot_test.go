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
	"oblikovati.org/model/sketch"
)

// flatScanNearXY writes a flat grid of scan points just above z = 0, so projecting them onto the XY
// sketch plane lands sketch points right under the scanned features.
func flatScanNearXY(t *testing.T) (string, math.Point3) {
	t.Helper()
	var body string
	for ix := -6; ix <= 6; ix++ {
		for iy := -6; iy <= 6; iy++ {
			body += fmt.Sprintf("%d %d 1\n", ix, iy)
		}
	}
	peak := math.P3(4, 3, 1)
	body += fmt.Sprintf("%g %g %g\n", float64(peak.X), float64(peak.Y), float64(peak.Z))
	path := filepath.Join(t.TempDir(), "flat.xyz")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path, peak
}

// TestInWindowSketchPointFromScanRenders attaches a scan, enters a sketch, selects a scan point,
// projects it onto the sketch plane as a sketch point, and captures the live viewport — the visual
// confirmation that Project Scan Point anchors sketch geometry on the as-built data (#645). Skips
// without a display.
func TestInWindowSketchPointFromScanRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	path, peak := flatScanNearXY(t)
	pc, _, err := s.AttachPointCloud("Flat", path)
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("enter sketch: %v", err)
	}
	s.Select(app.PointCloudPointHandle{Cloud: pc, Point: peak})
	sp, err := s.CreateSketchPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("project scan point: %v", err)
	}
	t.Logf("placed sketch point at %v from scan peak %v", sp.Position(), peak)

	// The sketch environment drives its own plan view; this exercises the full attach → sketch →
	// select → project → render path without error (the placed position is asserted in the app
	// unit test, which is the authoritative geometric check).
	frameCameraOn(s, pc.RangeBox())
	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-sketch-point.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
