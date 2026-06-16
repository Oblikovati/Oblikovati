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
	"oblikovati.org/model/sketch"
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
	_, _ = compdef.AddPart(s.Workspace(), "viewport-test.opd", true)
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

	// Settle the dock layout + establish hover, then Shift+middle-drag to orbit.
	native.InjectMousePos(cx, cy)
	frame()
	frame()
	native.InjectKeyShift(true)
	native.InjectMouseButton(native.MouseMiddle, true)
	frame()
	before := s.Camera()
	for i := 1; i <= 3; i++ {
		native.InjectMousePos(cx+float32(15*i), cy)
		frame()
	}
	got := s.Camera()
	native.InjectMouseButton(native.MouseMiddle, false)
	native.InjectKeyShift(false)
	frame()

	if got.Eye.IsEqualTo(before.Eye, 1e-6) {
		t.Fatal("Shift+middle-drag over the docked viewport did not orbit — the 3D viewport is not present/sized")
	}
}

// Left-drag must NOT orbit the camera — Inventor reserves it for selection / box-select,
// and the old left-drag-orbit collided with the sketch editor's left-click select/drag (#916).
func TestInWindowLeftDragDoesNotOrbit(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := framedSession()
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	native.InjectMousePos(cx, cy)
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)

	before := s.Camera()
	for i := 1; i <= 3; i++ {
		native.InjectMousePos(cx+float32(20*i), cy)
		viewportFrame(win, s)
	}
	got := s.Camera()
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)

	if !got.Eye.IsEqualTo(before.Eye, 1e-6) {
		t.Fatalf("left-drag must not orbit: eye moved from %v to %v", before.Eye, got.Eye)
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

// fakeEmptyPicker reports a miss for every point pick — so a viewport press lands on empty
// space and arms box-select (the head's begin condition).
type fakeEmptyPicker struct{}

func (fakeEmptyPicker) Pick(_, _ float64, _ *app.SelectionFilter) (app.Selectable, bool) {
	return nil, false
}

// fakeRegionHit returns one canned selectable for any box, so the test asserts the head
// drove Begin→Update→Commit without needing real projected geometry.
type fakeRegionHit struct{ sel app.Selectable }

func (f fakeRegionHit) PickRegion(_, _, _, _ float64, _ bool, _ *app.SelectionFilter) []app.Selectable {
	return []app.Selectable{f.sel}
}

// TestInWindowBoxSelectDragSelects drives a real left-drag from empty space across the live
// viewport and asserts the head ran the box-select state machine to completion: nothing is
// selected mid-drag, and on release the region hit joins the selection (#916).
func TestInWindowBoxSelectDragSelects(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := framedSession()
	s.SetPicker(fakeEmptyPicker{}) // every press is on empty space → box-select arms
	s.SetRegionPicker(fakeRegionHit{sel: app.BodyHandle{}})
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)

	native.InjectMouseButton(native.MouseLeft, true) // press on empty → begin box
	viewportFrame(win, s)
	if !s.BoxSelectActive() {
		t.Fatal("a left press on empty space did not begin box-select")
	}
	for i := 1; i <= 3; i++ { // drag out the rectangle
		native.InjectMousePos(cx+float32(40*i), cy+float32(25*i))
		viewportFrame(win, s)
	}
	if s.Selection().Count() != 0 {
		t.Fatalf("box-select must not commit mid-drag: count=%d", s.Selection().Count())
	}

	native.InjectMouseButton(native.MouseLeft, false) // release → commit
	viewportFrame(win, s)
	if s.BoxSelectActive() {
		t.Error("releasing the button must end the box-select drag")
	}
	if s.Selection().Count() != 1 {
		t.Errorf("box-select release should select the region hit: count=%d, want 1", s.Selection().Count())
	}
}

// TestInWindowBoxSelectCrossingShift drives a right→left (crossing) Shift+drag, exercising the
// crossing-mode rubber-band colour and the Shift-add modifier path of the head box-select
// handler (#916).
func TestInWindowBoxSelectCrossingShift(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := framedSession()
	s.SetPicker(fakeEmptyPicker{})
	s.SetRegionPicker(fakeRegionHit{sel: app.BodyHandle{}})
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	native.InjectMousePos(cx+150, cy+100) // start lower-right
	viewportFrame(win, s)
	native.InjectMousePos(cx+150, cy+100)
	viewportFrame(win, s)

	native.InjectKeyShift(true)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)
	for i := 1; i <= 3; i++ { // drag up-left → crossing select
		native.InjectMousePos(cx+150-float32(60*i), cy+100-float32(40*i))
		viewportFrame(win, s)
	}
	if _, _, _, _, crossing := s.BoxSelectRect(); !crossing {
		t.Error("a right→left drag should be a crossing select (drives the green rubber-band)")
	}
	native.InjectMouseButton(native.MouseLeft, false) // release with Shift STILL held...
	viewportFrame(win, s)                             // ...so the commit reads ShiftMod (add)
	native.InjectKeyShift(false)
	if s.Selection().Count() != 1 {
		t.Errorf("shift+crossing box-select should add the region hit: count=%d, want 1", s.Selection().Count())
	}
}

// TestInWindowSketchEntityDrag drives a real left-drag on an unconstrained sketch point in the
// live window and asserts the head moved it — the sketch-editor drag-to-move path (#916).
func TestInWindowSketchEntityDrag(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := app.NewSession()
	docu, _ := compdef.AddPart(s.Workspace(), "drag.opd", true)
	part := docu.Content().(*compdef.PartComponentDefinition)
	sk := part.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	s.Grid().SnapToGrid = false
	p := sk.Points().Add(math.P2(2, 0)) // a free point, off the (fixed) origin

	// Force-finish the enter-sketch camera swing (test frames report ~0 dt, so it never
	// completes on its own and input would be skipped while animating), then aim a known camera
	// so (2,0) sits at the viewport centre.
	s.TickCameraAnimation(100)
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(2, 0, 10), math.P3(2, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	// The point (= camera target) projects to the viewport CONTENT centre, which sits a little
	// below window-centre because of the "Viewport" window title bar; scan a few rows to find
	// the exact pixel that picks it (ImGui metrics vary), then press there.
	native.InjectMousePos(float32(inWinW/2), float32(inWinH/2))
	viewportFrame(win, s)
	viewportFrame(win, s)
	cx, cy, found := pressFindsSketchPoint(win, s)
	if !found {
		t.Fatal("could not find the on-screen pixel for the sketch point")
	}
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, true) // press on the point → begin drag
	viewportFrame(win, s)
	if !s.EntityDragActive() {
		t.Fatal("pressing an unconstrained sketch point did not begin a drag")
	}
	for i := 1; i <= 4; i++ { // drag to the right
		native.InjectMousePos(cx+float32(15*i), cy)
		viewportFrame(win, s)
	}
	native.InjectMouseButton(native.MouseLeft, false) // release → commit
	viewportFrame(win, s)

	if s.EntityDragActive() {
		t.Error("releasing should end the sketch drag")
	}
	if p.Position().X <= 2.0 {
		t.Errorf("the dragged sketch point did not move in +x: now at %v", p.Position())
	}
}

// pressFindsSketchPoint scans a few pixel rows around window-centre for the screen position
// that picks the sketch point (the viewport content centre is offset below window-centre by the
// title bar, and the exact inset depends on ImGui metrics). It presses without moving (zero
// drag, so the point does not shift), checks whether a drag armed, then cancels — returning the
// first pixel that works.
func pressFindsSketchPoint(win *native.Window, s *app.Session) (float32, float32, bool) {
	x := float32(inWinW / 2)
	for dy := float32(-6); dy <= 36; dy += 3 {
		y := float32(inWinH/2) + dy
		native.InjectMousePos(x, y)
		viewportFrame(win, s)
		native.InjectMouseButton(native.MouseLeft, true)
		viewportFrame(win, s)
		armed := s.EntityDragActive()
		native.InjectMouseButton(native.MouseLeft, false)
		viewportFrame(win, s)
		s.CancelEntityDrag()
		if armed {
			return x, y, true
		}
	}
	return x, float32(inWinH / 2), false
}
