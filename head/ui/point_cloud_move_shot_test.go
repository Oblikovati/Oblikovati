//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestInWindowCloudMoveDragFollows arms the Move tool, drags the cloud across the viewport through
// the session drag the head wires (BeginCloudDrag/UpdateCloudDrag/CommitCloudDrag), and captures the
// result — the visual confirmation that an interactive drag moves the cloud and the fitted plane
// follows it (#645). Skips without a display.
func TestInWindowCloudMoveDragFollows(t *testing.T) {
	win := newShotWindow(t)
	defer win.Destroy()
	s := framedSession()
	pc, wp := attachSheetWithFitPlane(t, s)
	x0 := float64(wp.Plane().Origin().X)

	// Arm Move and drag from the viewport centre across the screen (the head's updateCloudDrag
	// drives exactly these calls on press / hold / release).
	s.Select(app.PointCloudHandle{Clouds: nil, Cloud: pc})
	if err := s.StartMoveSelectedCloud(); err != nil {
		t.Fatalf("StartMoveSelectedCloud: %v", err)
	}
	if !s.BeginCloudDrag(400, 300) {
		t.Fatal("BeginCloudDrag should start")
	}
	s.UpdateCloudDrag(560, 300) // drag right
	s.CommitCloudDrag()
	x1 := float64(wp.Plane().Origin().X)
	t.Logf("plane origin X: %.3f before drag, %.3f after (cloud dragged right)", x0, x1)
	if x1 == x0 {
		t.Fatalf("plane did not follow the cloud drag: X stayed %.3f", x0)
	}

	captureFramed(t, win, s, pc.RangeBox(), "point-cloud-move")
}
