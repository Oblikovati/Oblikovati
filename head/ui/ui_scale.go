//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// uiIconScale is the live icon-size factor (1.0 = 100%), refreshed each frame by applyUIScale
// from the session's persisted preference. Package-global to match the chrome's other per-window
// state (icons, theme); scaledIconPx reads it at every icon draw site (#1232 follow-up: icons
// too small on high-resolution monitors).
var uiIconScale = 1.0

// scaledIconPx scales a base icon pixel size by the live UI icon-scale preference, rounded to a
// whole pixel. A non-positive scale (shouldn't happen — guarded on read) falls back to the base
// so an icon can never collapse to zero.
//
//	scaledIconPx(18) // => 27 when uiIconScale is 1.5
func scaledIconPx(base int) int {
	if uiIconScale <= 0 {
		return base
	}
	return int(math.Round(float64(base) * uiIconScale))
}

// uiFontScale is the live text-scale factor (1.0 = 100%), refreshed each frame by applyUIScale.
// Dialog window sizes read it so a box grows WITH its text: a fixed pixel size clips its content —
// and its OK/Cancel row — once the user raises the font scale (#1232 follow-up / #1753).
var uiFontScale = 1.0

// applyUIScale pushes the user's persisted UI scale into the renderer once per frame: the text
// scale into ImGui (live, via style.FontScaleMain) and the icon scale into uiIconScale for the
// icon draw sites. Called from prepareChromeFrame so it precedes the ribbon, which sizes its band
// from the scaled icon size.
func applyUIScale(win *native.Window, s *app.Session) {
	win.SetUIFontScale(float32(s.UIFontScale()))
	uiIconScale = s.UIIconScale()
	uiFontScale = s.UIFontScale()
}

// scaleDim scales a base dialog dimension by the live UI font scale. A 0 dimension (ImGui's
// "auto-size this axis") is preserved, and a non-positive scale falls back to the base so a
// window can never collapse.
func scaleDim(v float32) float32 {
	if uiFontScale <= 0 || v == 0 {
		return v
	}
	return v * float32(uiFontScale)
}

// dialogMargin is the gap kept between a dialog and the host-window edge when clamping, so the
// window border (and, vertically, the menu bar) stays clear.
const dialogMargin = 24

// dialogFit scales a base dialog size by the font scale AND clamps it to the main viewport, so a
// scaled dialog never opens LARGER than the host window — which would push its footer/OK-Cancel row
// past the window edge, re-creating the very clip this fix removes (#1753). A 0 axis (auto-size) is
// left untouched. The vertical clamp leaves double the margin for the menu and title bars.
func dialogFit(w, h float32) (float32, float32) {
	w, h = scaleDim(w), scaleDim(h)
	vw, vh := native.MainViewportSize()
	if w > 0 && vw > 2*dialogMargin && w > vw-dialogMargin {
		w = vw - dialogMargin
	}
	if h > 0 && vh > 4*dialogMargin && h > vh-2*dialogMargin {
		h = vh - 2*dialogMargin
	}
	return w, h
}

// dialogSizeOnce sets a dialog's first-open size, scaled by the UI font scale and clamped to the host
// window, so the window and its content grow together yet never overflow — the OK/Cancel row no longer
// falls off the bottom at a raised font scale (#1753).
func dialogSizeOnce(w, h float32) { native.SetNextWindowSizeOnce(dialogFit(w, h)) }
