//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/viewport"
)

// applyShadow enables the sun shadow map when object or ground shadows are on and the scene has
// a light, fitting the light frustum to the model bounds (min/max, excluding the ground plane so
// the shadow map stays high-resolution on the model). The primary light (index 0) casts;
// density/softness come from the active shadow settings (ADR-0026 §6).
func applyShadow(win *native.Window, s *app.Session, min, max [3]float32, ok bool) {
	sh := s.ShadowSettings()
	lights := s.SceneLighting().ActiveLights()
	mapOn := sh.ObjectShadows || sh.GroundShadows || sh.AmbientShadows
	if !ok || !mapOn || len(lights) == 0 {
		win.SetViewportShadow(nil, false, 0, 0, false, false)
		return
	}
	lvp := viewport.LightMatrix(min, max, lights[0].Direction)
	castDirect := sh.ObjectShadows || sh.GroundShadows
	win.SetViewportShadow(lvp[:], true, sh.Density, sh.Softness, castDirect, sh.AmbientShadows)
}
