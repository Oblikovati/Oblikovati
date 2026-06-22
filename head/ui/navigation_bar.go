//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Navigation Bar (#913 N25): a floating vertical strip of icon buttons at the viewport's right
// edge, re-exposing the View tab's camera tools on-canvas. The tools themselves live in app
// (HomeView/FitView/ArmZoomWindow/ToggleConstrainedOrbit/LookAtSelection); this only draws the
// buttons and routes clicks, so it stays thin.

// navBarButton is one Navigation Bar tool: an icon key, the session action it runs, and an optional
// active-state predicate (a toggle/armed tool renders accented while on).
type navBarButton struct {
	id, icon string
	run      func(*app.Session)
	active   func(*app.Session) bool
}

// navBarButtons are the bar's tools, top to bottom.
var navBarButtons = []navBarButton{
	{"navbar.home", "home", func(s *app.Session) { s.HomeView() }, nil},
	{"navbar.fit", "zoom-all", func(s *app.Session) { s.FitView() }, nil},
	{"navbar.zoomWindow", "zoom-window", func(s *app.Session) { s.ArmZoomWindow() }, (*app.Session).ZoomWindowArmed},
	{"navbar.constrainedOrbit", "orbit-constrained", func(s *app.Session) { s.ToggleConstrainedOrbit() }, (*app.Session).ConstrainedOrbitActive},
	{"navbar.lookAt", "look-at", func(s *app.Session) { s.LookAtSelection() }, nil},
}

const (
	navBarIconPx = 22
	navBarPad    = 4
	navBarMargin = 6
)

// drawNavigationBar draws the floating Navigation Bar as a vertical strip of icon buttons at the
// viewport's right edge, vertically centred (#913 N25). ox,oy is the viewport panel's window-local
// origin (the cursor position captured before the viewport's InvisibleButton); pw,ph its pixel size.
// Each button runs its session action; a toggle/armed tool renders accented. No-op when hidden.
func drawNavigationBar(s *app.Session, ox, oy float32, pw, ph int) {
	if !s.ShowNavBar() {
		return
	}
	iconPx := scaledIconPx(navBarIconPx)
	cell := float32(iconPx + 2*navBarPad)
	x := ox + float32(pw) - cell - navBarMargin
	y := oy + (float32(ph)-cell*float32(len(navBarButtons)))/2
	for _, b := range navBarButtons {
		if drawNavBarButton(s, b, x, y, iconPx) {
			y += cell // a button with no icon asset consumes no slot (keeps the strip gapless)
		}
	}
}

// drawNavBarButton draws one Navigation Bar button at (x,y): its icon at iconPx, accented while
// its tool is armed/toggled. Returns false (drawing nothing) when the icon asset is missing.
func drawNavBarButton(s *app.Session, b navBarButton, x, y float32, iconPx int) bool {
	tex, ok := icons.texture(b.icon, "", iconPx)
	if !ok {
		return false
	}
	native.SetCursorPos(x, y)
	on := b.active != nil && b.active(s)
	if on {
		native.PushStyleColor("Button", accentColor)
		native.PushStyleColor("ButtonHovered", accentColor)
		native.PushStyleColor("ButtonActive", accentColor)
		defer native.PopStyleColor(3)
	}
	if native.ImageButton(b.id, tex, float32(iconPx), float32(iconPx), identityTint) {
		b.run(s)
	}
	return true
}
