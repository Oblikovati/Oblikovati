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
	native.Separator()
	// Mouse-navigation preferences (Inventor-parity, 2026-08-17): defaults are middle-drag pans,
	// Shift+middle-drag orbits, scroll-up zooms in, and the wheel zooms to the cursor.
	native.Text("Mouse navigation (defaults match Inventor):")
	editNavModeCombo("Middle button drag", s.MMBMode(), s.SetMMBMode, navModeOptions("pan"))
	editNavModeCombo("Shift + middle button drag", s.ShiftMMBMode(), s.SetShiftMMBMode, navModeOptions("orbit"))
	editNavModeCombo("Ctrl + middle button drag", s.CtrlMMBMode(), s.SetCtrlMMBMode, navModeOptions("pan"))
	prefCheckbox("Invert wheel zoom direction", s.WheelInvert, s.SetWheelInvert)
	prefCheckbox("Zoom to cursor", s.ZoomToCursor, s.SetZoomToCursor)
	native.Separator()
	// Classic menu bar + status bar visibility (also on View ▸ Windows).
	prefCheckbox("Show classic menu bar", s.ShowMenuBar, s.SetShowMenuBar)
	prefCheckbox("Show status bar", s.ShowStatusBar, s.SetShowStatusBar)
}

// prefCheckbox draws a boolean preference checkbox: get supplies the current value, set persists a
// toggle (its error surfaces through reportPrefError). It takes closures rather than the whole
// session, so the preferences tab does not widen head/ui's *app.Session coupling (audit I5).
func prefCheckbox(label string, get func() bool, set func(bool) error) {
	v := get()
	if native.Checkbox(label, &v) {
		reportPrefError(set(v))
	}
}

// navModeOptions lists the three middle-button gestures, with this binding's
// Inventor default first in the combo.
func navModeOptions(preferred string) []string {
	opts := []string{preferred}
	for _, mode := range []string{"pan", "orbit", "zoom"} {
		if mode != preferred {
			opts = append(opts, mode)
		}
	}
	return opts
}

// editNavModeCombo draws one pan/orbit/zoom mode selector (BeginCombo + Selectable
// pattern, same as the add-in grid combo cells).
func editNavModeCombo(label, current string, set func(string) error, options []string) {
	if !native.BeginCombo(label, current) {
		return
	}
	for _, opt := range options {
		if native.Selectable(opt, opt == current) {
			reportPrefError(set(opt))
		}
	}
	native.EndCombo()
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
