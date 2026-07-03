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
	"oblikovati.org/renderer"
)

// wideGridScan writes a flat 25×25 grid of scan points spanning a wide area, so a crop box around
// the centre visibly drops the outer points.
func wideGridScan(t *testing.T) string {
	t.Helper()
	var body string
	for ix := -12; ix <= 12; ix++ {
		for iy := -12; iy <= 12; iy++ {
			body += fmt.Sprintf("%d %d 0\n", ix, iy)
		}
	}
	path := filepath.Join(t.TempDir(), "wide.xyz")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path
}

// TestInWindowCropBoxRenders attaches a wide grid scan, frames it, then drives the crop-box tool
// with two viewport clicks around the centre and captures the cropped cloud — the visual
// confirmation that boxing a region in the viewport limits the cloud's display (#645). Skips
// without a display.
func TestInWindowCropBoxRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, _, err := s.AttachPointCloud("Wide", wideGridScan(t))
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	frameCameraOn(s, pc.RangeBox())

	// Box the central half of the cloud's actual extent by projecting those corners to viewport
	// pixels with the same projection the crop tool uses, so the clicks reliably enclose the inner
	// points. The extent is read from the attached cloud's RangeBox rather than the raw file
	// coordinates because import rescales the cloud to the document's working unit (#1636) — a
	// hard-coded world box would no longer straddle the (now smaller) cloud.
	rb := pc.RangeBox()
	sx0, sy0, ok0 := renderer.Project(s.Camera(), 0.1, 5000, math.P3(rb.Min.X/2, rb.Min.Y/2, 0))
	sx1, sy1, ok1 := renderer.Project(s.Camera(), 0.1, 5000, math.P3(rb.Max.X/2, rb.Max.Y/2, 0))
	if !ok0 || !ok1 {
		t.Fatal("central corners must project on-screen")
	}
	full := pc.DisplayedPointCount()
	s.StartTool(app.NewCropBoxTool(pc))
	s.Click(sx0, sy0)
	s.Click(sx1, sy1)
	cropped := pc.DisplayedPointCount()
	t.Logf("displayed points: %d before crop, %d after", full, cropped)
	if cropped >= full || cropped == 0 {
		t.Fatalf("crop should reduce the displayed set to a non-empty subset, got %d of %d", cropped, full)
	}

	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-crop.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
