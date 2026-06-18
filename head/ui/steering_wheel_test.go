//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
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
