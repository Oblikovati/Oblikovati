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

// navBarButton is one Navigation Bar tool: an icon key, the session action it runs, an optional
// active-state predicate (a toggle/armed tool renders accented while on), and an optional enabled
// predicate (a nil enabled means always enabled; false greys the button out and blocks the click).
type navBarButton struct {
	id, icon string
	run      func(*app.Session)
	active   func(*app.Session) bool
	enabled  func(*app.Session) bool
}

// navBarButtons are the bar's tools, top to bottom.
var navBarButtons = []navBarButton{
	{"navbar.home", "home", func(s *app.Session) { s.HomeView() }, nil, nil},
	{"navbar.fit", "zoom-all", func(s *app.Session) { s.FitView() }, nil, nil},
	{"navbar.zoomWindow", "zoom-window", func(s *app.Session) { s.ArmZoomWindow() }, (*app.Session).ZoomWindowArmed, nil},
	{"navbar.constrainedOrbit", "orbit-constrained", func(s *app.Session) { s.ToggleConstrainedOrbit() }, (*app.Session).ConstrainedOrbitActive, nil},
	// Look At only orients to a selected work plane / planar face, so it disables with nothing
	// suitable selected — a click would otherwise be a silent no-op (#1468 follow-up).
	{"navbar.lookAt", "look-at", func(s *app.Session) { s.LookAtSelection() }, nil, (*app.Session).CanLookAt},
}

const (
	navBarIconPx = 22
	navBarPad    = 4
	navBarMargin = 6
)

// drawNavigationBar draws the floating Navigation Bar as a vertical strip of icon buttons at the
// viewport's right edge, vertically centred (#913 N25). bx,by is the viewport's SCREEN origin; pw,ph
// its pixel size. The strip is its OWN borderless overlay window so its buttons receive clicks while
// the viewport keeps drag-to-orbit everywhere else — ImGui routes the cursor to the topmost window
// under it, which a same-window InvisibleButton used to swallow (#1468). Each button runs its session
// action; a toggle/armed tool renders accented. No-op when hidden.
func drawNavigationBar(s *app.Session, bx, by float32, pw, ph int) {
	if !s.ShowNavBar() {
		return
	}
	iconPx := scaledIconPx(navBarIconPx)
	x, y := navBarStripRect(bx, by, pw, ph)
	native.SetNextWindowPos(x, y)
	if native.BeginOverlayWindow("##navbar") {
		for _, b := range navBarButtons {
			drawNavBarButton(s, b, iconPx)
		}
	}
	native.End()
}

// navBarStripRect is the strip's top-left corner in the given coordinate frame whose origin (ox,oy)
// is the viewport's top-left: a right-edge vertical strip of len(navBarButtons) cells, vertically
// centred. The overlay window is positioned here.
func navBarStripRect(ox, oy float32, pw, ph int) (x, y float32) {
	cell := float32(scaledIconPx(navBarIconPx) + 2*navBarPad)
	x = ox + float32(pw) - cell - navBarMargin
	y = oy + (float32(ph)-cell*float32(len(navBarButtons)))/2
	return x, y
}

// drawNavBarButton draws one Navigation Bar button at the overlay window's current cursor: its icon
// at iconPx, accented while its tool is armed/toggled. Returns false (drawing nothing) when the icon
// asset is missing. Clicking it runs the tool — which now reaches it, since the strip is its own
// window rather than items under the viewport's input-capturing button (#1468).
func drawNavBarButton(s *app.Session, b navBarButton, iconPx int) bool {
	tex, ok := icons.texture(b.icon, "", iconPx)
	if !ok {
		return false
	}
	on := b.active != nil && b.active(s)
	if on {
		native.PushStyleColor("Button", chromeTheme.accentColor)
		native.PushStyleColor("ButtonHovered", chromeTheme.accentColor)
		native.PushStyleColor("ButtonActive", chromeTheme.accentColor)
		defer native.PopStyleColor(3)
	}
	if b.enabled != nil && !b.enabled(s) {
		native.BeginDisabled(true) // greys the icon and blocks the click when the tool can't run
		defer native.EndDisabled()
	}
	clicked := native.ImageButton(b.id, tex, float32(iconPx), float32(iconPx), identityTint)
	x0, y0 := native.ItemRectMin()
	x1, y1 := native.ItemRectMax()
	navBarButtonRect[b.id] = [4]float32{x0, y0, x1, y1} // screen rect, so a click test can target it
	if clicked {
		b.run(s)
	}
	return true
}

// navBarButtonRect holds each nav-bar button's screen rect (minX,minY,maxX,maxY) from the last
// frame it drew, keyed by id. It lets an interaction test click a button by id without recomputing
// the layout; production code does not read it (#1468 regression guard).
var navBarButtonRect = map[string][4]float32{}
