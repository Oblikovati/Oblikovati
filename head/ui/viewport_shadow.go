//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/head/viewport"
)

// applyShadow enables the sun shadow map when object shadows are on and the scene has a light
// and geometry, fitting the light frustum to the drawn mesh's bounds (ADR-0026 §6). The primary
// light (index 0) casts; density/softness come from the active shadow settings.
func applyShadow(win *native.Window, s *app.Session, m viewport.Mesh) {
	sh := s.ShadowSettings()
	lights := s.SceneLighting().ActiveLights()
	if !sh.ObjectShadows || len(lights) == 0 {
		win.SetViewportShadow(nil, false, 0, 0)
		return
	}
	min, max, ok := viewport.SceneBounds(m)
	if !ok {
		win.SetViewportShadow(nil, false, 0, 0)
		return
	}
	lvp := viewport.LightMatrix(min, max, lights[0].Direction)
	win.SetViewportShadow(lvp[:], true, sh.Density, sh.Softness)
}
