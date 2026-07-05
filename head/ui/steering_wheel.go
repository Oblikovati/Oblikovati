//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// SteeringWheels (#913 N26): a radial menu of navigation tools summoned at the cursor, for rapid
// switching between them. Like the Navigation Bar it re-exposes the existing app tools; here they are
// laid out in a ring so a flick of the cursor reaches any of them. The ring is PINNED where it is
// summoned so its wedges can be clicked (#1754); choosing a tool, or Esc, hides the wheel.

// steeringWheelTool is one wedge of the SteeringWheel: an icon and the session action it runs.
type steeringWheelTool struct {
	id, icon string
	run      func(*app.Session)
}

// steeringWheelTools are the wheel's tools, clockwise from the top.
var steeringWheelTools = []steeringWheelTool{
	{"wheel.home", "home", func(s *app.Session) { s.HomeView() }},
	{"wheel.fit", "zoom-all", func(s *app.Session) { s.FitView() }},
	{"wheel.zoomWindow", "zoom-window", func(s *app.Session) { s.ArmZoomWindow() }},
	{"wheel.orbit", "orbit-constrained", func(s *app.Session) { s.ToggleConstrainedOrbit() }},
	{"wheel.lookAt", "look-at", func(s *app.Session) { s.LookAtSelection() }},
	{"wheel.rewind", "rewind", func(s *app.Session) { s.PreviousView() }},
}

const (
	steeringIconPx  = 26
	steeringRadius  = 62
	steeringRingSeg = 40
)

// Pin state (#1754): the ring is SUMMONED at the cursor the first frame it is shown, then HELD there
// (in viewport-local coords) while active. A ring that re-centred on the live cursor every frame could
// never be reached — moving toward a wedge dragged the whole wheel by the same amount, so the icons
// fled the pointer and no wedge could ever be clicked.
var (
	steeringLocalX, steeringLocalY float32
	steeringPinned                 bool
)

// drawSteeringWheel draws the SteeringWheels menu as a ring of icon buttons, SUMMONED at the cursor
// when the tool is activated over the viewport and then PINNED there for as long as it stays active
// (#913 N26, #1754). ox,oy is the viewport panel's window-local origin. Choosing a tool runs it and
// hides the wheel; the wheel's wedges are discrete nav commands (Home/Fit/…), so a click is the right
// gesture once the ring holds still.
func drawSteeringWheel(s *app.Session, ox, oy float32) {
	if !s.SteeringWheelActive() {
		steeringPinned = false
		return
	}
	if !steeringPinned {
		if !native.IsItemHovered() {
			return // wait until the cursor is over the viewport, then summon the ring at that spot
		}
		vcx, vcy := viewportCursor()
		steeringLocalX, steeringLocalY, steeringPinned = float32(vcx), float32(vcy), true
	}
	cx, cy := ox+steeringLocalX, oy+steeringLocalY
	drawSteeringRing(cx, cy)
	iconPx := scaledIconPx(steeringIconPx)
	for i, tool := range steeringWheelTools {
		tex, ok := icons.texture(tool.icon, "", iconPx)
		if !ok {
			continue
		}
		bx, by := steeringWedgePos(cx, cy, i, len(steeringWheelTools), iconPx)
		native.SetCursorPos(bx, by)
		if native.ImageButton(tool.id, tex, float32(iconPx), float32(iconPx), identityTint) {
			tool.run(s)
			s.DisarmSteeringWheel()
		}
	}
}

// steeringWedgePos is the top-left of wedge i's icon button on the ring around (cx,cy): the i-th of n
// tools laid clockwise from the top. Pure geometry (no session), so it is unit-testable and adds no
// head→Session coupling.
func steeringWedgePos(cx, cy float32, i, n, iconPx int) (float32, float32) {
	a := 2*stdmath.Pi*float64(i)/float64(n) - stdmath.Pi/2 // start at the top
	bx := cx + float32(steeringRadius*stdmath.Cos(a)) - float32(iconPx)/2
	by := cy + float32(steeringRadius*stdmath.Sin(a)) - float32(iconPx)/2
	return bx, by
}

// drawSteeringRing draws the faint hub circle the tool icons sit on.
func drawSteeringRing(cx, cy float32) {
	col := [4]float32{0.85, 0.85, 0.92, 0.45}
	r := float32(steeringRadius)
	prevx, prevy := cx+r, cy
	for i := 1; i <= steeringRingSeg; i++ {
		a := 2 * stdmath.Pi * float64(i) / steeringRingSeg
		x := cx + r*float32(stdmath.Cos(a))
		y := cy + r*float32(stdmath.Sin(a))
		native.DrawLine(prevx, prevy, x, y, col, 1.2)
		prevx, prevy = x, y
	}
}
