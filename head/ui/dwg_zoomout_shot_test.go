//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// dwgCorpus loads a real .dwg from the git-ignored experiments tree, skipping when absent.
func dwgCorpus(t *testing.T, name string) string {
	t.Helper()
	dir := os.Getenv("DWG_TESTFILES_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "experiments", "dwg-reverse-engineering")
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}
	return path
}

// TestInWindowLargeDWGZoomedOutRenders is the live regression for "zooming out a large-dimension
// DWG stopped rendering the sketch": testfile-7 is an ~100k-unit oval-plaza plan, so framing the
// whole drawing puts the camera tens of thousands of units back — well beyond the old fixed 5000
// far plane, which clipped the entire sketch to a blank viewport. With the adaptive far plane the
// drawing must produce a non-trivial draw list that frames within the frustum. Saves a PNG for a
// human/Claude to eyeball the rendered linework.
func TestInWindowLargeDWGZoomedOutRenders(t *testing.T) {
	win := newShotWindow(t)
	defer win.Destroy()
	s := framedSession()

	plane, err := sketch.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	res, err := s.ImportDWGFile(dwgCorpus(t, "testfile-7.dwg"), plane)
	if err != nil {
		t.Fatalf("ImportDWGFile: %v", err)
	}
	if res.EntityCount == 0 {
		t.Fatal("imported zero entities")
	}

	// Frame on the imported drawing's real extent (the line work), which is the whole ~100k-unit
	// plan, so the camera sits far past the legacy 5000 far plane. The sketch overlay is all OnTop
	// lines, which DrawListBounds excludes, so the far plane uses the cached overlay bounds.
	cachedPartSketchOverlays(s) // populate the overlay + bounds cache
	mn, mx, ok := cachedSketchOverlayBounds()
	if !ok {
		t.Fatal("imported sketch produced no overlay bounds (nothing to render)")
	}
	box := math.NewBox(math.P3(float64(mn[0]), float64(mn[1]), float64(mn[2])),
		math.P3(float64(mx[0]), float64(mx[1]), float64(mx[2])))
	span := box.Diagonal().Length()
	if span < 5000 {
		t.Skipf("drawing span %v is within the legacy far plane; not the zoom-out case", span)
	}

	// The adaptive far plane must enclose the framed camera's view of this box.
	frameCameraOn(s, box)
	far := viewportClipFar(s.Camera(), mn, mx, true)
	if far <= viewportFar {
		t.Errorf("far plane %v did not extend for a %v-unit drawing (sketch would clip on zoom-out)", far, span)
	}
	captureFramed(t, win, s, box, "dwg_zoomout_testfile7")
}
