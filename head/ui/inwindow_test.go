//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// True in-window verification: open the real GLFW+Vulkan+Dear ImGui window, drive the
// actual viewport navigation path (InvisibleButton → readNavInput → ApplyNavigation →
// camera) for real frames while injecting synthetic pointer input, and assert the live
// session camera moved. Skips cleanly when no display/Vulkan is available (e.g. CI).
package ui

import (
	stdmath "math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/scene"
)

const inWinW, inWinH = 800, 600

// newViewportWindow opens the head window, or skips the test if the environment has no
// display/Vulkan driver.
func newViewportWindow(t *testing.T) *native.Window {
	t.Helper()
	win, err := native.CreateWindow(inWinW, inWinH, "obk-inwindow-test")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	win.InitViewport()
	return win
}

// framedSession is a session whose camera looks down −Z from a known distance.
func framedSession() *app.Session {
	s := app.NewSession()
	_, _ = compdef.AddPart(s.Workspace(), "viewport-test.obk", true)
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	return s
}

// viewportFrame runs one real frame drawing only the viewport panel, forced to fill the
// window so the injected pointer lands on the navigation surface.
func viewportFrame(win *native.Window, s *app.Session) {
	win.BeginFrame()
	native.SetNextWindowPos(0, 0)
	native.SetNextWindowSize(inWinW, inWinH)
	drawViewportPanel(win, s)
	win.EndFrame(0.1, 0.1, 0.1)
}

// TestInWindowDockedViewportIsInteractive is the regression guard for the "barebones UI,
// no 3D viewport" bug: with no dock layout the viewport window auto-sized to a 1px sliver.
// It drives the full DrawChrome (dockspace + default layout) and drags over the window
// center — which falls in the docked viewport's central node — asserting the camera
// orbits. A collapsed viewport would not be hit there and the camera would not move.
func TestInWindowDockedViewportIsInteractive(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false // rebuild the default layout for this fresh window/context
	icons = nil         // rebind the icon cache to this fresh window (it holds GPU handles)
	s := framedSession()

	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	cx, cy := float32(inWinW/2), float32(inWinH/2) // center → the docked viewport's node

	// Settle the dock layout + establish hover, then press and drag.
	native.InjectMousePos(cx, cy)
	frame()
	frame()
	native.InjectMouseButton(native.MouseLeft, true)
	frame()
	before := s.Camera()
	for i := 1; i <= 3; i++ {
		native.InjectMousePos(cx+float32(15*i), cy)
		frame()
	}
	got := s.Camera()
	native.InjectMouseButton(native.MouseLeft, false)
	frame()

	if got.Eye.IsEqualTo(before.Eye, 1e-6) {
		t.Fatal("dragging over the docked viewport did not orbit — the 3D viewport is not present/sized")
	}
}

func TestInWindowLeftDragOrbitsCamera(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := framedSession()
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	// Hover the nav surface (pointer over it, button up), then press left → it activates.
	native.InjectMousePos(cx, cy)
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)

	before := s.Camera()

	// Drag right with the button held → orbit (button state persists across frames).
	for i := 1; i <= 3; i++ {
		native.InjectMousePos(cx+float32(20*i), cy)
		viewportFrame(win, s)
	}
	got := s.Camera()
	native.InjectMouseButton(native.MouseLeft, false) // release
	viewportFrame(win, s)

	if got.Eye.IsEqualTo(before.Eye, 1e-6) {
		t.Fatalf("left-drag through the live window did not orbit: eye stayed %v", got.Eye)
	}
	if d := stdmath.Abs(dist(got) - dist(before)); d > 1e-6 {
		t.Errorf("orbit changed the eye–target distance by %v, want it preserved", d)
	}
}

func TestInWindowMiddleDragPansCamera(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := framedSession()
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	// Hover, press middle button → active.
	native.InjectMousePos(cx, cy)
	native.InjectMouseButton(native.MouseMiddle, false)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseMiddle, true)
	viewportFrame(win, s)

	before := s.Camera()

	// Drag right with middle held → pan (eye and target slide together, distance kept).
	for i := 1; i <= 3; i++ {
		native.InjectMousePos(cx+float32(20*i), cy)
		viewportFrame(win, s)
	}
	got := s.Camera()
	native.InjectMouseButton(native.MouseMiddle, false)
	viewportFrame(win, s)

	if got.Target.IsEqualTo(before.Target, 1e-6) {
		t.Fatalf("middle-drag through the live window did not pan: target stayed %v", got.Target)
	}
	if !got.Forward().IsEqualTo(before.Forward(), 1e-6) {
		t.Errorf("pan changed the view direction: %v → %v", before.Forward(), got.Forward())
	}
	if d := stdmath.Abs(dist(got) - dist(before)); d > 1e-6 {
		t.Errorf("pan changed the eye–target distance by %v, want it preserved", d)
	}
}

func TestInWindowWheelZoomsCamera(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := framedSession()
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	// Two hover frames first: an ImGui item isn't reported hovered on the frame its
	// window first appears, so settle the layout before scrolling.
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	before := dist(s.Camera())

	native.InjectMousePos(cx, cy)
	native.InjectMouseWheel(3) // scroll up → zoom in
	viewportFrame(win, s)
	after := dist(s.Camera())

	if after >= before {
		t.Errorf("scroll-up over the live viewport should zoom in: %v → %v", before, after)
	}
}
