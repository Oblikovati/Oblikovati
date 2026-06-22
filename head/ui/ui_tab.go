//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// UI-scale slider bounds in percent (the session clamps to the same factor range). Icons reach a
// higher maximum than text because they re-rasterize from vector SVG and stay crisp when enlarged.
const (
	uiFontPercentMin = 75
	uiFontPercentMax = 200
	uiIconPercentMin = 75
	uiIconPercentMax = 250
)

// drawUITab renders the UI-scale preferences (#1232 follow-up: icons/text too small on
// high-resolution monitors): a text-size and an icon-size percentage slider, applied live and
// persisted. Edits write straight back to the session, so the next frame renders at the new scale.
func drawUITab(s *app.Session) {
	native.Text("Adjust the interface scale for your monitor's resolution.")
	native.Separator()
	editUITextScale(s)
	editUIIconScale(s)
	native.Separator()
	native.Text("Text size also scales the Script Console code editor.")
}

// editUITextScale draws the UI text-size slider, reading and writing the scale as a percentage.
func editUITextScale(s *app.Session) {
	pct := float32(s.UIFontScale() * 100)
	if native.SliderPercent("Text size", &pct, uiFontPercentMin, uiFontPercentMax) {
		reportPrefError(s.SetUIFontScale(float64(pct) / 100))
	}
}

// editUIIconScale draws the ribbon/tool icon-size slider.
func editUIIconScale(s *app.Session) {
	pct := float32(s.UIIconScale() * 100)
	if native.SliderPercent("Icon size", &pct, uiIconPercentMin, uiIconPercentMax) {
		reportPrefError(s.SetUIIconScale(float64(pct) / 100))
	}
}
