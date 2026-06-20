//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// tiltedPlanarScan writes an ASCII scan whose points lie on the tilted plane z = 0.4·x + 0.2·y,
// so a best-fit plane through them is visibly tilted (a flat sheet of points).
func tiltedPlanarScan(t *testing.T) string {
	t.Helper()
	var body string
	for ix := -10; ix <= 10; ix++ {
		for iy := -10; iy <= 10; iy++ {
			x, y := float64(ix), float64(iy)
			z := 0.4*x + 0.2*y
			body += fmt.Sprintf("%g %g %g\n", x, y, z)
		}
	}
	path := filepath.Join(t.TempDir(), "sheet.xyz")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path
}

// TestInWindowFitPlaneRenders attaches a flat sheet of scan points, fits a work plane to it, and
// captures the live viewport — the visual confirmation that Fit Work Plane builds a datum aligned
// with the scanned surface (#645). Skips without a display.
func TestInWindowFitPlaneRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, err := s.AttachPointCloud("Sheet", tiltedPlanarScan(t))
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	wp, plane, err := s.CreatePointCloudPlane("Sheet")
	if err != nil {
		t.Fatalf("fit plane: %v", err)
	}
	wp.SetVisible(true)
	t.Logf("fitted plane origin=%v normal=%v from %d points", plane.Origin, plane.Normal(), pc.TotalPointCount())

	frameCameraOn(s, pc.RangeBox())
	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-fit-plane.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
