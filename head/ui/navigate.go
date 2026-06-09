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
	// factor per wheel notch (<1 ⇒ a scroll-up zooms in).
	orbitRadPerPixel = 0.01
	zoomPerNotch     = 0.9
)

// NavInput is one frame of viewport pointer input. Hovered/Active come from the drag
// surface (Active stays true through a drag even if the cursor leaves it); the rest are
// the raw wheel/delta/button/modifier state.
type NavInput struct {
	Hovered bool
	Active  bool
	Wheel   float32
	DX, DY  float32
	Middle  bool
	Left    bool
	Shift   bool
}

// ApplyNavigation maps one frame of pointer input to a camera move, mirroring Inventor:
// wheel zooms (when hovered), middle-drag pans, Shift+middle-drag orbits; a plain
// left-drag also orbits for discoverability. The camera math lives in scene.
func ApplyNavigation(cam scene.Camera, in NavInput) scene.Camera {
	if in.Hovered && in.Wheel != 0 {
		cam = cam.Dolly(stdmath.Pow(zoomPerNotch, float64(in.Wheel)))
	}
	if !in.Active || (in.DX == 0 && in.DY == 0) {
		return cam
	}
	switch {
	case in.Middle && !in.Shift: // Inventor: pan with the middle button
		cam = cam.Pan(float64(in.DX), float64(in.DY))
	case (in.Middle && in.Shift) || in.Left: // Shift+middle orbits; left-drag orbits too
		cam = cam.Orbit(float64(-in.DX)*orbitRadPerPixel, float64(-in.DY)*orbitRadPerPixel)
	}
	return cam
}
