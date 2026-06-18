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
)

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
	Modal            NavMode // a held F2/F3/F4 navigation mode (drives a left-drag)
	Left             bool    // left button down — only consulted in a Modal mode
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
		cam = cam.Orbit(float64(-in.DX)*orbitRadPerPixel, float64(-in.DY)*orbitRadPerPixel)
	case NavZoom:
		cam = cam.Dolly(stdmath.Pow(zoomDragPerPixel, float64(in.DY)))
	}
	return cam
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
