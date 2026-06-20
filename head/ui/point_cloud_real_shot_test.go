//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// realScanPath is a sample Artec PLY scan; the test skips when it is not present (e.g. CI).
const realScanPath = "/home/vmiguel/git/oblikovati-workspace/vini-scan-examples/vini scan examples/capstan vertraging.ply"

// realE57Path is the same scan in ASTM E57 (the bit-packed CompressedVector export).
const realE57Path = "/home/vmiguel/git/oblikovati-workspace/vini-scan-examples/vini scan examples/capstan vertraging.e57"

// TestInWindowRealScanRenders attaches a real binary-PLY laser scan to the active part, budgets and
// frames it, and captures the live viewport — the visual confirmation that the PLY reader + render
// path handle real-world scan data (M17-F06, #645). Skips without the sample file or a display.
func TestInWindowRealScanRenders(t *testing.T) {
	if _, err := os.Stat(realScanPath); err != nil {
		t.Skipf("sample scan not present: %v", err)
	}
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, err := s.AttachPointCloud("Capstan", realScanPath)
	if err != nil {
		t.Fatalf("attach real scan: %v", err)
	}
	t.Logf("loaded %d scan points", pc.TotalPointCount())
	pc.SetMaximumPointCount(50000) // budget the 266k-point scan for display

	frameCameraOn(s, pc.RangeBox())

	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-real-scan.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}

// TestInWindowRealE57Renders attaches a real E57 laser scan and captures the live viewport — the
// visual confirmation that the E57 reader + render path handle real-world scan data (#645). Skips
// without the sample file or a display.
func TestInWindowRealE57Renders(t *testing.T) {
	if _, err := os.Stat(realE57Path); err != nil {
		t.Skipf("sample E57 not present: %v", err)
	}
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, err := s.AttachPointCloud("Capstan E57", realE57Path)
	if err != nil {
		t.Fatalf("attach real E57 scan: %v", err)
	}
	t.Logf("loaded %d E57 scan points", pc.TotalPointCount())
	pc.SetMaximumPointCount(50000)

	frameCameraOn(s, pc.RangeBox())

	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-real-e57.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}

// frameCameraOn points the session camera at a box's centre from a 3/4 view at a distance scaled
// to its diagonal, so the whole cloud sits in frame.
func frameCameraOn(s *app.Session, box math.Box) {
	center := box.Center()
	d := float64(box.Diagonal().Length())
	if d <= 0 {
		d = 1
	}
	cam := scene.NewCamera(inWinW, inWinH)
	off := math.V3(0.6, -0.8, 0.6).Scale(math.Scalar(d * 1.3))
	cam.Eye = center.TranslateBy(off)
	cam.Target = center
	cam.Up = math.V3(0, 0, 1)
	s.SetCamera(cam)
}
