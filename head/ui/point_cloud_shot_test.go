//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// TestInWindowPointCloudRenders attaches a recognizable grid of scan points to the active part and
// captures the live viewport, confirming the cloud's markers render through the real cgo path
// (M17-F06, #645). It is the visual confirmation for the render slice; skips when no display.
func TestInWindowPointCloudRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	cam := s.Camera()
	cam.Eye = math.P3(3, -4, 6) // a 3/4 view so the in-plane crosses read clearly
	s.SetCamera(cam)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("grid")})
	if _, err := def.PointClouds().Add("Grid", "grid.xyz", rid, gridPoints()); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if len(s.PointCloudItems(s.Camera(), 0.5)) != 1 {
		t.Fatalf("expected one point-cloud draw item, got %d", len(s.PointCloudItems(s.Camera(), 0.5)))
	}
	// Select a scan point so its orange snap-highlight reads among the cyan grid.
	s.Select(app.PointCloudPointHandle{Point: math.P3(0, 0, 0)})

	for range 8 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.12, 0.12, 0.14)
	}
	out := filepath.Join(outDir(), "point-cloud-grid.png")
	if err := win.SaveWindowPNG(out); err != nil {
		t.Logf("SaveWindowPNG(%s): %v", out, err)
	}
}

// gridPoints returns a 7×7 grid of points in the XY plane spanning ±4, a pattern that reads
// clearly as a point cloud in the capture.
func gridPoints() []math.Point3 {
	var pts []math.Point3
	for i := -3; i <= 3; i++ {
		for j := -3; j <= 3; j++ {
			pts = append(pts, math.P3(math.Scalar(i)*1.3, math.Scalar(j)*1.3, 0))
		}
	}
	return pts
}
