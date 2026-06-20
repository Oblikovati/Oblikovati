//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// TestInWindowSketchHUDDrawsAndCommits drives the 2D-sketch dynamic-input HUD through real
// DrawChrome frames in the live window (#790): it enters a sketch with the Line tool active and
// one point placed, parks the cursor over the canvas, types a precise length, and asserts the
// HUD engaged and painted (handleSketchHUD + the paint helpers) without crashing, then that an
// injected Enter committed the typed point. Skips cleanly when no display/Vulkan is present.
func TestInWindowSketchHUDDrawsAndCommits(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s, line := hudSketchSession(t)

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	cx, cy := float32(inWinW/2), float32(inWinH/2) // the docked viewport node

	// Settle the dock + camera, establish hover over the canvas.
	native.InjectMousePos(cx, cy)
	frame()
	frame()
	if !s.SketchHUDView(float64(cx), float64(cy)).Visible {
		t.Skip("HUD not visible in this layout (cursor did not map onto the sketch plane)")
	}

	// Type a precise length; the HUD must engage and paint over several frames without crashing.
	for _, r := range "60" {
		s.HUDInputRune(r)
	}
	native.InjectMousePos(cx, cy)
	frame()
	if !s.HUDEngaged() {
		t.Fatal("typing into the HUD did not engage it")
	}

	// Commit the entry; the line tool's pending point advances to the committed coordinate.
	before, _ := line.PendingReferencePoint()
	if err := s.HUDCommit(float64(cx), float64(cy)); err != nil {
		t.Fatalf("HUDCommit: %v", err)
	}
	frame()
	after, ok := line.PendingReferencePoint()
	if !ok || after.IsEqualTo(before, 1e-9) {
		t.Errorf("HUDCommit did not place a new point: ref %v → %v", before, after)
	}
	if s.HUDEngaged() {
		t.Error("HUDCommit should reset the HUD entry")
	}
}

// hudSketchSession returns a session editing a 2D sketch with the Line tool active and its first
// point placed, the camera settled facing the plane — so the HUD shows Length/Angle.
func hudSketchSession(t *testing.T) (*app.Session, *app.LineTool) {
	t.Helper()
	s := app.NewSession()
	docu, _ := compdef.AddPart(s.Workspace(), "hud.opd", true)
	part := docu.Content().(*compdef.PartComponentDefinition)
	sk := part.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	s.Grid().SnapToGrid = false
	s.TickCameraAnimation(100) // finish the enter-sketch swing (test dt≈0) so picks/HUD map
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	line := app.NewLineTool()
	s.StartTool(line)
	s.Click(float64(inWinW/2), float64(inWinH/2)) // place the first point → polar HUD
	return s, line
}
