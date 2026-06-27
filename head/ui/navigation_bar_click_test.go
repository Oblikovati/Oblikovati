//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// clickNavBarButton renders the viewport, locates the named nav-bar button's screen rect, and
// injects a full left click at its centre (hover one frame first so the allow-overlap button can
// take the press), then settles. Reports false if the button was not drawn.
func clickNavBarButton(win *native.Window, s *app.Session, id string) bool {
	viewportFrame(win, s) // draw once to record the button rects
	r, ok := navBarButtonRect[id]
	if !ok {
		return false
	}
	cx, cy := (r[0]+r[2])/2, (r[1]+r[3])/2
	native.InjectMousePos(cx, cy)
	viewportFrame(win, s) // establish hover before the press (allow-overlap settles over one frame)
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
	return true
}

// TestNavBarButtonClicksReachActions is the #1468 regression guard: clicking a Navigation Bar button
// must fire its action. The bug was that the full-viewport InvisibleButton (submitted first) swallowed
// the click via ImGui's overlap rule, so the overlay buttons never fired. We click two buttons whose
// effect is a deterministic session flag and assert the flag flipped — which only happens if the
// click actually reached the button.
func TestNavBarButtonClicksReachActions(t *testing.T) {
	cases := []struct {
		id    string
		state func(*app.Session) bool // the flag the button's action sets, false until clicked
	}{
		{"navbar.zoomWindow", (*app.Session).ZoomWindowArmed},
		{"navbar.constrainedOrbit", (*app.Session).ConstrainedOrbitActive},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			win := newViewportWindow(t)
			defer win.Destroy()
			dockLaidOut = false // fresh layout for this window/context
			icons = nil
			s := framedSession()
			if !s.ShowNavBar() {
				t.Skip("nav bar hidden")
			}
			if c.state(s) {
				t.Fatalf("%s state should start false", c.id)
			}
			if !clickNavBarButton(win, s, c.id) {
				t.Fatalf("%s button was not drawn", c.id)
			}
			if !c.state(s) {
				t.Errorf("clicking %s did not fire its action — the viewport swallowed the click (#1468)", c.id)
			}
			native.InjectMouseButton(native.MouseLeft, false)
		})
	}
}
