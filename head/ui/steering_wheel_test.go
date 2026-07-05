//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// TestSteeringWheelToolsRunActions: each wheel tool runs its session action — spot-check that the
// zoom-window wheel arms Zoom Window and the orbit wheel toggles Constrained Orbit (#913 N26).
func TestSteeringWheelToolsRunActions(t *testing.T) {
	byID := map[string]steeringWheelTool{}
	for _, w := range steeringWheelTools {
		if w.run == nil {
			t.Fatalf("wheel tool %q has no action", w.id)
		}
		byID[w.id] = w
	}
	for _, id := range []string{"wheel.home", "wheel.fit", "wheel.zoomWindow", "wheel.orbit", "wheel.lookAt", "wheel.rewind"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("steering wheel is missing tool %q", id)
		}
	}

	s := app.NewSession()
	byID["wheel.zoomWindow"].run(s)
	if !s.ZoomWindowArmed() {
		t.Error("the zoom-window wheel tool should arm Zoom Window")
	}
	byID["wheel.orbit"].run(s)
	if !s.ConstrainedOrbitActive() {
		t.Error("the orbit wheel tool should toggle Constrained Orbit on")
	}
}

// TestSteeringWedgePos checks the pure ring geometry: wedge 0 is laid at the top (angle -π/2), so its
// icon's top-left is radius above (cx,cy), inset by half the icon. No window needed.
func TestSteeringWedgePos(t *testing.T) {
	const cx, cy, iconPx = 100.0, 100.0, 20
	bx, by := steeringWedgePos(cx, cy, 0, 6, iconPx)
	wantX := float32(cx) - float32(iconPx)/2
	wantY := float32(cy) - float32(steeringRadius) - float32(iconPx)/2
	if d := bx - wantX; d > 1e-3 || d < -1e-3 {
		t.Errorf("wedge 0 x = %v, want %v (top of ring, centred)", bx, wantX)
	}
	if d := by - wantY; d > 1e-3 || d < -1e-3 {
		t.Errorf("wedge 0 y = %v, want %v (radius above centre)", by, wantY)
	}
}

// TestInWindowSteeringWheelDraws activates the SteeringWheels menu over a part and runs frames, so a
// mismatched ImGui Begin/End or a bad ImageButton call would trip an assertion.
func TestInWindowSteeringWheelDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()
	s.ToggleSteeringWheel()
	if !s.SteeringWheelActive() {
		t.Fatal("SteeringWheels should be active after toggle")
	}
	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// TestInWindowSteeringWheelPinsAtSummonPoint is the #1754 regression: once the wheel is summoned it
// must STAY where it appeared, not re-centre on the live cursor every frame. The old ring chased the
// cursor, so moving toward a wedge dragged the whole wheel by the same amount and no wedge could ever
// be clicked. Here we summon the ring, then move the cursor far away and assert the ring's centre did
// not budge — the property that makes the wedges reachable. Saves a PNG for eyeball confirmation.
func TestInWindowSteeringWheelPinsAtSummonPoint(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	defer func() { steeringPinned = false }() // package global; reset for other tests

	s := framedSession()
	s.ToggleSteeringWheel()
	frame := func() {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	cx, cy := float32(inWinW/2), float32(inWinH/2)

	// Hover the viewport centre so the ring summons + pins there (an ImGui item is not reported hovered
	// on its first appearance, so settle a few frames).
	for i := 0; i < 4; i++ {
		native.InjectMousePos(cx, cy)
		frame()
	}
	if !steeringPinned {
		t.Fatal("the wheel should have pinned at the summon point after hovering the viewport")
	}
	px, py := steeringLocalX, steeringLocalY
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "steering-wheel-pinned.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}

	// Move the cursor well away and render: the ring must NOT follow it.
	native.InjectMousePos(cx+140, cy+90)
	frame()
	if steeringLocalX != px || steeringLocalY != py {
		t.Errorf("the summoned ring must stay put, not follow the cursor: (%v,%v) -> (%v,%v)",
			px, py, steeringLocalX, steeringLocalY)
	}
}
