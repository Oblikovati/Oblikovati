//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// shiftX is a +dx translation matrix (row-major, translation in the last column).
func shiftX(dx float64) math.Matrix4 {
	m := math.Identity4()
	c := m.Cells()
	c[3] = math.Scalar(dx)
	return math.Matrix4FromCells(c)
}

// TestInWindowSketchPointProvenanceFollowsCloud anchors a sketch point on a scan point, moves the
// cloud, re-projects, and captures — the visual confirmation that the scan-anchored sketch point
// keeps a live link to the cloud and follows it (provenance) (#645). Skips without a display.
func TestInWindowSketchPointProvenanceFollowsCloud(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	path, peak := flatScanNearXY(t) // reuses the grid+peak scan from the sketch-point shot test
	pc, _, err := s.AttachPointCloud("Flat", path)
	if err != nil {
		t.Fatalf("attach scan: %v", err)
	}
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("enter sketch: %v", err)
	}
	s.Select(app.PointCloudPointHandle{Cloud: pc, Point: peak})
	p, err := s.CreateSketchPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("project scan point: %v", err)
	}
	x0 := float64(p.Position().X)

	pc.SetTransform(shiftX(20)) // move the cloud; the anchored sketch point must re-project with it
	sk.UpdateProjections()
	x1 := float64(p.Position().X)
	t.Logf("sketch point X: %.3f before move, %.3f after (cloud shifted +20)", x0, x1)
	if x1-x0 < 19 {
		t.Fatalf("sketch point did not follow the cloud: X %.3f → %.3f", x0, x1)
	}

	frameCameraOn(s, pc.RangeBox())
	for range 10 {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-sketchpoint-provenance.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
