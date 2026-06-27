//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawLightingWindow renders the Lighting settings panel while it is open: global exposure /
// brightness / ambience sliders and per-light color, intensity and direction editing on the
// active rig (M16/F03, ADR-0026). Edits write straight back to the session's live lighting.
func drawLightingBody(s *app.Session) {
	exposure := s.Exposure()
	if native.SliderFloat("Exposure", &exposure, 0.1, 4) {
		s.SetExposure(exposure)
	}
	brightness := s.Brightness()
	if native.SliderFloat("Brightness", &brightness, 0, 3) {
		s.SetBrightness(brightness)
	}
	ambience := s.Ambience()
	if native.SliderFloat("Ambience", &ambience, 0, 1) {
		s.SetAmbience(ambience)
	}
	native.Separator()
	drawLightRows(s)
	native.Separator()
	if native.Button("Done") {
		s.CloseLightingPanel()
	}
}

// drawLightRows draws an editor block per active light (color, intensity, direction); a changed
// control writes the whole light back through the session.
func drawLightRows(s *app.Session) {
	for i, l := range s.Lights() {
		native.Text(fmt.Sprintf("Light %d", i+1))
		edited := false
		col := [4]float32{l.Color[0], l.Color[1], l.Color[2], 1}
		if native.ColorEdit4(fmt.Sprintf("Color##l%d", i), &col) {
			l.Color = [3]float32{col[0], col[1], col[2]}
			edited = true
		}
		if native.SliderFloat(fmt.Sprintf("Intensity##l%d", i), &l.Intensity, 0, 6) {
			edited = true
		}
		if native.SliderFloat(fmt.Sprintf("Dir X##l%d", i), &l.Direction[0], -1, 1) {
			edited = true
		}
		if native.SliderFloat(fmt.Sprintf("Dir Y##l%d", i), &l.Direction[1], -1, 1) {
			edited = true
		}
		if native.SliderFloat(fmt.Sprintf("Dir Z##l%d", i), &l.Direction[2], -1, 1) {
			edited = true
		}
		if edited {
			_ = s.SetLight(i, l)
		}
		native.Separator()
	}
}
