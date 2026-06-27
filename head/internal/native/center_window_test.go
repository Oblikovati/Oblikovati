//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"
)

// TestCenterNextWindowCentersOnViewport pins the #1474 fix: CenterNextWindow must place
// the following window so its MIDPOINT sits on the main-viewport centre — i.e. top-left =
// centre − size/2 — rather than ImGui's default top-left cascade that left the save prompt
// lost in a corner. It drives a real offscreen ImGui frame (lavapipe+xvfb in CI) and reads
// the window's resulting position back through WindowPos.
func TestCenterNextWindowCentersOnViewport(t *testing.T) {
	w, err := CreateWindow(800, 600, "Oblikovati (#1474 center test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	const ww, wh float32 = 240, 140
	var px, py, cx, cy float32
	w.BeginFrame()
	cx, cy = MainViewportCenter()
	CenterNextWindow()
	SetNextWindowSize(ww, wh)
	if Begin("center-probe") {
		px, py = WindowPos()
	}
	End()
	w.EndFrame(0.10, 0.10, 0.12)

	wantX, wantY := cx-ww/2, cy-wh/2
	const tol = 1.0 // sub-pixel rounding in ImGui's pivot placement
	if abs32(px-wantX) > tol || abs32(py-wantY) > tol {
		t.Errorf("centered window at (%.1f,%.1f), want (%.1f,%.1f) — pivot centring broken (#1474)",
			px, py, wantX, wantY)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
