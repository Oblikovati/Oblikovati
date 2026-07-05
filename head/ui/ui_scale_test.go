//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

func TestScaledIconPx(t *testing.T) {
	defer func() { uiIconScale = 1.0 }() // package global shared with the live chrome
	cases := []struct {
		scale float64
		base  int
		want  int
	}{
		{1.0, 18, 18},  // identity
		{2.0, 18, 36},  // doubled
		{1.5, 18, 27},  // 1.5x rounds cleanly
		{1.5, 40, 60},  // large icon scales too
		{0, 18, 18},    // corrupt scale → base, never zero
		{-1, 40, 40},   // negative → base
		{1.25, 18, 23}, // 22.5 rounds to 23
	}
	for _, c := range cases {
		uiIconScale = c.scale
		if got := scaledIconPx(c.base); got != c.want {
			t.Errorf("scaledIconPx(%d) at scale %v = %d, want %d", c.base, c.scale, got, c.want)
		}
	}
}

// TestScaleDim covers the font-scale dialog-dimension math without a window: a 0 axis (ImGui's
// auto-size) is preserved, a positive dimension multiplies by the live UI font scale, and a
// non-positive scale falls back to the base so a window can never collapse (#1753).
func TestScaleDim(t *testing.T) {
	defer func() { uiFontScale = 1.0 }() // package global; restore for other tests

	uiFontScale = 1.5
	if got := scaleDim(200); got != 300 {
		t.Errorf("scaleDim(200) at 1.5x = %v, want 300", got)
	}
	if got := scaleDim(0); got != 0 {
		t.Errorf("scaleDim(0) = %v, want 0 (auto-size axis preserved)", got)
	}

	uiFontScale = 0 // guard: a non-positive scale must fall back to the base dimension
	if got := scaleDim(200); got != 200 {
		t.Errorf("scaleDim(200) at 0 scale = %v, want 200 (fallback)", got)
	}
}

// TestInWindowApplyUIScale drives applyUIScale in a real frame and asserts it copies the session's
// persisted icon scale into the live uiIconScale the draw sites read. The font scale goes straight
// to ImGui (style.FontScaleMain) and is exercised here for its draw path.
func TestInWindowApplyUIScale(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	defer func() { uiIconScale = 1.0 }()
	s := app.NewSession()
	if err := s.SetUIIconScale(1.75); err != nil {
		t.Fatalf("SetUIIconScale: %v", err)
	}
	if err := s.SetUIFontScale(1.25); err != nil {
		t.Fatalf("SetUIFontScale: %v", err)
	}

	win.BeginFrame()
	applyUIScale(win, s)
	win.EndFrame(0.1, 0.1, 0.1)

	if uiIconScale != 1.75 {
		t.Errorf("applyUIScale left uiIconScale = %v, want 1.75", uiIconScale)
	}
}
