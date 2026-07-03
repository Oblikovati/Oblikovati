//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// tiltedSheetScan writes a flat sheet of scan points on the tilted plane z = 0.3·x + 0.15·y.
func tiltedSheetScan(t *testing.T) string {
	t.Helper()
	var body string
	for ix := -10; ix <= 10; ix++ {
		for iy := -10; iy <= 10; iy++ {
			x, y := float64(ix), float64(iy)
			body += fmt.Sprintf("%g %g %g\n", x, y, 0.3*x+0.15*y)
		}
	}
	path := filepath.Join(t.TempDir(), "sheet.xyz")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path
}

// liftZ is a +dz translation matrix (row-major, translation in the last column).
func liftZ(dz float64) math.Matrix4 {
	m := math.Identity4()
	c := m.Cells()
	c[11] = math.Scalar(dz)
	return math.Matrix4FromCells(c)
}

// TestInWindowProvenancePlaneFollowsCloud fits a work plane to a scan, moves the cloud up, and
// captures the recomputed scene — the visual confirmation that the fitted plane keeps a live link
// to the cloud and follows it (provenance), rather than staying behind as a frozen datum (#645).
func TestInWindowProvenancePlaneFollowsCloud(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, _, err := s.AttachPointCloud("Sheet", tiltedSheetScan(t))
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	wp, _, err := s.CreatePointCloudPlane("Sheet")
	if err != nil {
		t.Fatalf("fit plane: %v", err)
	}
	wp.SetVisible(true)
	z0 := float64(wp.Plane().Origin().Z)

	// Move the cloud up; a recompute must re-fit the plane to the cloud's new position.
	pc.SetTransform(liftZ(10))
	if part, ok := s.ActiveDocument().Content().(*compdef.PartComponentDefinition); ok {
		part.Recompute()
	}
	z1 := float64(wp.Plane().Origin().Z)
	t.Logf("plane origin Z: %.3f before move, %.3f after (cloud lifted +10)", z0, z1)
	if z1-z0 < 9 {
		t.Fatalf("plane did not follow the cloud: Z %0.3f → %0.3f", z0, z1)
	}

	frameCameraOn(s, pc.RangeBox())
	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-provenance.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
