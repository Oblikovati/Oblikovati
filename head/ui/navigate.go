// SPDX-License-Identifier: GPL-2.0-only

// This file has no cgo build tag: the input→camera mapping is pure Go so it can be
// unit-tested without the native (Vulkan/GLFW/ImGui) stack. chrome.go (cgo) reads the
// raw pointer state from the platform layer into NavInput and calls ApplyNavigation.
package ui

import (
	stdmath "math"

	"oblikovati.org/scene"
)

const (
	// orbitRadPerPixel turns a drag pixel into an orbit angle; zoomPerNotch is the dolly
	// factor per wheel notch (<1 ⇒ a scroll-up zooms in); zoomDragPerPixel is the per-pixel
	// dolly for hold-F3 realtime zoom (>1 so dragging down zooms out, up zooms in).
	orbitRadPerPixel = 0.01
	zoomPerNotch     = 0.9
	zoomDragPerPixel = 1.01
	// orbitRingFraction sizes the Free-Orbit ring (radius = fraction of the smaller viewport side);
	// orbitRimFraction is where the rim band begins (inside it is free orbit, the band splits
	// yaw/pitch, outside the ring rolls). See classifyOrbitZone (#913 N5–N8).
	orbitRingFraction = 0.40
	orbitRimFraction  = 0.85
)

// OrbitZone is which region of the Free-Orbit ring a drag began in; it selects the rotation kind
// (#913 N5–N8). The zone is latched at the start of the drag and held until release.
type OrbitZone int

const (
	OrbitFree  OrbitZone = iota // inside the inner disc: free (yaw+pitch) orbit
	OrbitYaw                    // left/right rim band: rotate about the screen vertical axis
	OrbitPitch                  // top/bottom rim band: rotate about the screen horizontal axis
	OrbitRoll                   // outside the ring: roll about the view axis
)

// orbitRingRadius is the Free-Orbit ring radius for a w×h viewport — a fraction of the smaller side.
func orbitRingRadius(w, h float64) float64 { return orbitRingFraction * stdmath.Min(w, h) }

// classifyOrbitZone picks the orbit zone for a drag starting at (px,py) relative to a ring centred at
// (cx,cy) with the given radius: the inner disc is free orbit; the rim band splits into yaw (nearer
// the left/right of the ring, |dx|≥|dy|) and pitch (nearer top/bottom); outside the ring rolls.
func classifyOrbitZone(px, py, cx, cy, radius float64) OrbitZone {
	if radius <= 0 {
		return OrbitFree // no ring (degenerate viewport) → plain free orbit
	}
	dx, dy := px-cx, py-cy
	r := stdmath.Hypot(dx, dy)
	switch {
	case r > radius:
		return OrbitRoll
	case r < radius*orbitRimFraction:
		return OrbitFree
	case stdmath.Abs(dx) >= stdmath.Abs(dy):
		return OrbitYaw
	default:
		return OrbitPitch
	}
}

// NavMode is a held function-key navigation mode (Inventor's F2 pan / F3 zoom / F4 orbit): while
// the key is down, a left-drag drives that gesture.
type NavMode uint8

const (
	NavNone  NavMode = iota
	NavPan           // F2
	NavZoom          // F3
	NavOrbit         // F4
)

// NavInput is one frame of viewport pointer input. Hovered/Active come from the drag
// surface (Active stays true through a drag even if the cursor leaves it); the rest are
// the raw wheel/delta/button/modifier state. Modal+Left carry the hold-F-key navigation.
type NavInput struct {
	Hovered          bool
	Active           bool
	Wheel            float32
	DX, DY           float32
	CursorX, CursorY float32 // viewport-local cursor pixels (for zoom-to-cursor, N2)
	Middle           bool
	Shift            bool
	Modal            NavMode   // a held F2/F3/F4 navigation mode (drives a left-drag)
	Left             bool      // left button down — only consulted in a Modal mode
	OrbitZone        OrbitZone // the Free-Orbit ring zone latched at drag start (#913 N5–N8)
}

// ApplyNavigation maps one frame of pointer input to a camera move, mirroring Inventor:
// wheel zooms (when hovered), middle-drag pans, Shift+middle-drag orbits. Left-drag is
// deliberately NOT a navigation gesture — Inventor reserves it for selection (box-select
// on empty space, drag-to-move on an entity), and orbiting on left-drag collided with the
// sketch editor's left-click select/drag (#916). The camera math lives in scene.
func ApplyNavigation(cam scene.Camera, in NavInput) scene.Camera {
	if in.Hovered && in.Wheel != 0 {
		cam = cam.DollyToCursor(stdmath.Pow(zoomPerNotch, float64(in.Wheel)), float64(in.CursorX), float64(in.CursorY))
	}
	if !in.Active || (in.DX == 0 && in.DY == 0) {
		return cam
	}
	switch navGesture(in) {
	case NavPan:
		cam = cam.Pan(float64(in.DX), float64(in.DY))
	case NavOrbit:
		cam = applyOrbit(cam, in)
	case NavZoom:
		cam = cam.Dolly(stdmath.Pow(zoomDragPerPixel, float64(in.DY)))
	}
	return cam
}

// applyOrbit applies one frame of the Free-Orbit ring gesture: the drag-start zone (latched by the
// head into in.OrbitZone) selects free orbit, yaw-only (about the screen vertical axis), pitch-only
// (about the horizontal axis), or roll about the view axis (#913 N5–N8). The default zone (OrbitFree)
// is also what a Shift+middle orbit uses, so that gesture keeps its existing free-orbit behaviour.
func applyOrbit(cam scene.Camera, in NavInput) scene.Camera {
	switch in.OrbitZone {
	case OrbitYaw:
		return cam.Orbit(float64(-in.DX)*orbitRadPerPixel, 0)
	case OrbitPitch:
		return cam.Orbit(0, float64(-in.DY)*orbitRadPerPixel)
	case OrbitRoll:
		return cam.Roll(ringRollAngle(cam, in))
	default:
		return cam.Orbit(float64(-in.DX)*orbitRadPerPixel, float64(-in.DY)*orbitRadPerPixel)
	}
}

// ringRollAngle returns this frame's roll from a perimeter drag: the tangential component of the
// (DX,DY) move about the ring centre (the viewport centre), per unit radius — a small-angle arc.
// Screen y is down, so a counter-clockwise drag yields a positive roll.
func ringRollAngle(cam scene.Camera, in NavInput) float64 {
	rx := float64(in.CursorX) - float64(cam.Width)/2
	ry := float64(in.CursorY) - float64(cam.Height)/2
	r := stdmath.Hypot(rx, ry)
	if r < 1 {
		return 0
	}
	tangential := (float64(in.DX)*(-ry) + float64(in.DY)*rx) / r // drag · tangent unit
	return tangential / r
}

// navGesture resolves a drag into a navigation gesture: a held F-key (F2 pan / F3 zoom / F4
// orbit) drives a left-drag; otherwise the middle button pans, Shift+middle orbits. A plain
// left-drag (no modal key) is NavNone — it belongs to selection, not navigation.
func navGesture(in NavInput) NavMode {
	if in.Left {
		return in.Modal
	}
	switch {
	case in.Middle && in.Shift:
		return NavOrbit
	case in.Middle:
		return NavPan
	default:
		return NavNone
	}
}
