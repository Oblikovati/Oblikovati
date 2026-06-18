//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// navButton returns the nav-bar button with the given id (fatal if absent).
func navButton(t *testing.T, id string) navBarButton {
	t.Helper()
	for _, b := range navBarButtons {
		if b.id == id {
			return b
		}
	}
	t.Fatalf("no nav-bar button %q", id)
	return navBarButton{}
}

// TestNavBarButtonActiveReflectsState: the toggle/armed tools' buttons report active when their tool
// is on, so the bar renders them accented (#913 N25).
func TestNavBarButtonActiveReflectsState(t *testing.T) {
	s := app.NewSession()

	zw := navButton(t, "navbar.zoomWindow")
	if zw.active(s) {
		t.Error("zoom-window button active before arming")
	}
	s.ArmZoomWindow()
	if !zw.active(s) {
		t.Error("zoom-window button should be active once armed")
	}

	co := navButton(t, "navbar.constrainedOrbit")
	if co.active(s) {
		t.Error("constrained-orbit button active before enabling")
	}
	s.ToggleConstrainedOrbit()
	if !co.active(s) {
		t.Error("constrained-orbit button should be active once on")
	}

	// The momentary tools have no active predicate.
	if navButton(t, "navbar.home").active != nil || navButton(t, "navbar.fit").active != nil {
		t.Error("momentary tools (home/fit) should have no active predicate")
	}
}

// TestInWindowNavigationBarDraws renders the viewport with the Navigation Bar visible over a part and
// runs frames, so a mismatched ImGui Begin/End or a bad ImageButton call would trip an assertion.
func TestInWindowNavigationBarDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()
	if !s.ShowNavBar() {
		t.Fatal("the Navigation Bar should be shown by default")
	}
	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}
