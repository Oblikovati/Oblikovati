//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// SteeringWheels (#913 N26): a radial menu of navigation tools drawn around the cursor, for rapid
// switching between them. Like the Navigation Bar it re-exposes the existing app tools; here they are
// laid out in a ring so a flick of the cursor reaches any of them. Choosing a tool hides the wheel.

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

// drawSteeringWheel draws the SteeringWheels menu as a ring of icon buttons centred on the cursor
// while the tool is active and the viewport is hovered (#913 N26). ox,oy is the viewport panel's
// window-local origin. Choosing a tool runs it and hides the wheel. No-op when inactive/unhovered.
func drawSteeringWheel(s *app.Session, ox, oy float32) {
	if !s.SteeringWheelActive() || !native.IsItemHovered() {
		return
	}
	vcx, vcy := viewportCursor()
	cx, cy := ox+float32(vcx), oy+float32(vcy)
	drawSteeringRing(cx, cy)
	for i, tool := range steeringWheelTools {
		tex, ok := icons.texture(tool.icon, steeringIconPx)
		if !ok {
			continue
		}
		a := 2*stdmath.Pi*float64(i)/float64(len(steeringWheelTools)) - stdmath.Pi/2 // start at the top
		bx := cx + float32(steeringRadius*stdmath.Cos(a)) - steeringIconPx/2
		by := cy + float32(steeringRadius*stdmath.Sin(a)) - steeringIconPx/2
		native.SetCursorPos(bx, by)
		if native.ImageButton(tool.id, tex, steeringIconPx, steeringIconPx, identityTint) {
			tool.run(s)
			s.DisarmSteeringWheel()
		}
	}
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
