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
	"oblikovati.org/model/compdef"
)

// flatScanWithPeak writes a flat grid with one raised peak point, so the work point placed on the
// peak is easy to spot as it follows the cloud.
func flatScanWithPeak(t *testing.T) (string, math.Point3) {
	t.Helper()
	var body string
	for ix := -6; ix <= 6; ix++ {
		for iy := -6; iy <= 6; iy++ {
			body += fmt.Sprintf("%d %d 0\n", ix, iy)
		}
	}
	peak := math.P3(0, 0, 6)
	body += fmt.Sprintf("%g %g %g\n", float64(peak.X), float64(peak.Y), float64(peak.Z))
	path := filepath.Join(t.TempDir(), "peak.xyz")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path, peak
}

// TestInWindowWorkPointProvenanceFollowsCloud places a work point on a snapped scan peak, moves the
// cloud, and captures the recomputed scene — the visual confirmation that the anchored work point
// keeps a live link to the cloud and follows it (provenance) (#645). Skips without a display.
func TestInWindowWorkPointProvenanceFollowsCloud(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	path, peak := flatScanWithPeak(t)
	pc, err := s.AttachPointCloud("Peak", path)
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	s.Select(app.PointCloudPointHandle{Cloud: pc, Point: peak})
	wp, err := s.CreateWorkPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("create work point: %v", err)
	}
	z0 := float64(wp.Point().Z)

	pc.SetTransform(liftZ(8)) // move the cloud up; the anchored point must follow on recompute
	if part, ok := s.ActiveDocument().Content().(*compdef.PartComponentDefinition); ok {
		part.Recompute()
	}
	z1 := float64(wp.Point().Z)
	t.Logf("work point Z: %.3f before move, %.3f after (cloud lifted +8)", z0, z1)
	if z1-z0 < 7 {
		t.Fatalf("work point did not follow the cloud: Z %.3f → %.3f", z0, z1)
	}

	frameCameraOn(s, pc.RangeBox())
	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-workpoint-provenance.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
