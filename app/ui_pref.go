// SPDX-License-Identifier: GPL-2.0-only

package app

// User-facing UI scale surface (#1232 follow-up: icons/text too small on high-resolution
// monitors). The session owns the persisted FontScale/IconScale preferences; the head reads
// them live each frame (font scale → ImGui style.FontScaleMain, icon scale → icon rasterization
// size). Values are stored as a factor where 1.0 = 100%.

// UI scale bounds. The lower bound keeps the interface usable (never invisibly small, even from
// a hand-edited options file); the upper bounds cap growth at a sane maximum per surface — text
// to 2x, icons to 2.5x (icons read crisply larger because they re-rasterize from vector SVG).
const (
	minUIScale     = 0.5
	maxUIFontScale = 2.0
	maxUIIconScale = 2.5
)

// UIFontScale reports the persisted UI text scale (1.0 = 100%). A non-positive stored value
// (absent or corrupt key) reads back as 1.0 so the interface can never become invisible.
//
//	s.UIFontScale() // => 1.2 after SetUIFontScale(1.2)
func (s *Session) UIFontScale() float64 {
	return scaleOrDefault(s.appOptions.UI.FontScale)
}

// SetUIFontScale clamps v to [minUIScale, maxUIFontScale] and persists it to the user's
// options file. Out-of-range and non-positive inputs are clamped, never rejected.
func (s *Session) SetUIFontScale(v float64) error {
	s.appOptions.UI.FontScale = clampScale(v, minUIScale, maxUIFontScale)
	return s.saveOptions()
}

// UIIconScale reports the persisted icon scale (1.0 = 100%), with the same non-positive guard
// as UIFontScale.
func (s *Session) UIIconScale() float64 {
	return scaleOrDefault(s.appOptions.UI.IconScale)
}

// SetUIIconScale clamps v to [minUIScale, maxUIIconScale] and persists it.
func (s *Session) SetUIIconScale(v float64) error {
	s.appOptions.UI.IconScale = clampScale(v, minUIScale, maxUIIconScale)
	return s.saveOptions()
}

// scaleOrDefault returns v, or 1.0 when v is non-positive — the guard that keeps a missing or
// corrupt stored scale from hiding the UI.
func scaleOrDefault(v float64) float64 {
	if v <= 0 {
		return 1
	}
	return v
}

// clampScale confines v to [lo, hi], treating a non-positive v as the low bound rather than
// snapping it to 1.0 — an explicit setter call honors the user's intent up to the safe minimum.
func clampScale(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
