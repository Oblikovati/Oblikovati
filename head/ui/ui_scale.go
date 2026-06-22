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

// applyUIScale pushes the user's persisted UI scale into the renderer once per frame: the text
// scale into ImGui (live, via style.FontScaleMain) and the icon scale into uiIconScale for the
// icon draw sites. Called from prepareChromeFrame so it precedes the ribbon, which sizes its band
// from the scaled icon size.
func applyUIScale(win *native.Window, s *app.Session) {
	win.SetUIFontScale(float32(s.UIFontScale()))
	uiIconScale = s.UIIconScale()
}
