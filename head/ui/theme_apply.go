//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/theme"
)

// Theme-driven colors for the viewport overlays and ribbon icons. They live here in one
// place (rather than scattered across the overlay files) because they all share one
// owner: the active theme. They are seeded from the Dark built-in so they hold correct
// values before the first applyThemeIfChanged and in any headless head test, then
// overwritten from the active theme whenever it changes (ADR-0021). The overlay files
// read these vars by name exactly as before.
var (
	gridMinorColor, gridMajorColor, gridAxisColor          [4]float32
	sketchColor, sketchSelectedColor, sketchCandidateColor [4]float32
	previewColor                                           [4]float32 // rubber-band preview
	dimensionColor, dimensionDrivenColor                   [4]float32
	snapGlyphColor, pointMarkerColor                       [4]float32
	activeViewportBorderColor                              [4]float32 // focused split-view tile outline
	faintPlaneColor, hoverPlaneColor, selectedPlaneColor   [4]float32
	planeFillColor                                         [4]float32 // translucent plane fill (alpha = opacity)
	selectionHighlight                                     [4]float32 // picked-body highlight
	iconTint                                               [4]float32 // ribbon glyph tint
	accentColor                                            [4]float32 // active/toggled ribbon button
	windowClearColor                                       [3]float32 // swapchain (chrome) clear
)

func init() { refreshThemeColors(theme.DefaultDark()) }

// appliedThemeRevision is the library revision last pushed into Dear ImGui and the color
// vars, so applyThemeIfChanged is a cheap no-op on the frames the theme has not changed.
var appliedThemeRevision uint64

// applyThemeIfChanged re-styles the UI from the active theme when it has changed since
// the last frame — pushing the chrome colors into Dear ImGui's persistent style (set
// once per change, not per widget), refreshing the overlay/icon color vars, and updating
// the 3D viewport's clear color. Called at the top of DrawChrome; this is the live-preview
// hook (a theme switch or a color edit bumps the library revision).
func applyThemeIfChanged(win *native.Window, s *app.Session) {
	rev := s.ThemeRevision()
	if rev == appliedThemeRevision {
		return
	}
	appliedThemeRevision = rev
	active := s.Theme()
	applyChromeStyle(active)
	refreshThemeColors(active)
	vp := active.Color(types.TokenViewportBg)
	win.SetViewportClear(vp.R, vp.G, vp.B)
}

// refreshThemeColors copies the active theme's overlay/icon colors into the package vars
// the overlay builders read each frame.
func refreshThemeColors(t contract.Theme) {
	arr := func(tok types.ThemeToken) [4]float32 { return t.Color(tok).Array() }
	refreshGridThemeColors(arr)
	refreshSketchThemeColors(arr)
	refreshPlaneThemeColors(arr)
	selectionHighlight = arr(types.TokenSelectionHighlight)
	iconTint = arr(types.TokenIconTint)
	accentColor = arr(types.TokenChromeAccent)
	c := t.Color(types.TokenChromeWindowBg)
	windowClearColor = [3]float32{c.R, c.G, c.B}
}

func refreshGridThemeColors(arr func(types.ThemeToken) [4]float32) {
	gridMinorColor = arr(types.TokenGridMinor)
	gridMajorColor = arr(types.TokenGridMajor)
	gridAxisColor = arr(types.TokenGridAxis)
}

func refreshSketchThemeColors(arr func(types.ThemeToken) [4]float32) {
	sketchColor = arr(types.TokenSketchGeometry)
	sketchSelectedColor = arr(types.TokenSketchSelected)
	sketchCandidateColor = arr(types.TokenSketchCandidate)
	previewColor = arr(types.TokenSketchPreview)
	dimensionColor = arr(types.TokenDimensionDriving)
	dimensionDrivenColor = arr(types.TokenDimensionDriven)
	snapGlyphColor = arr(types.TokenSnapGlyph)
	activeViewportBorderColor = arr(types.TokenViewportActiveBorder)
	pointMarkerColor = sketchColor // placed points match the sketch wireframe
}

func refreshPlaneThemeColors(arr func(types.ThemeToken) [4]float32) {
	faintPlaneColor = arr(types.TokenPlaneFaint)
	hoverPlaneColor = arr(types.TokenPlaneHover)
	selectedPlaneColor = arr(types.TokenPlaneSelected)
	planeFillColor = arr(types.TokenPlaneFill)
}

// WindowClearColor is the themed background the frame loop clears the swapchain to (the
// area behind/around the docked panels). The main loop passes it to Window.EndFrame.
func WindowClearColor() (r, g, b float32) {
	return windowClearColor[0], windowClearColor[1], windowClearColor[2]
}

// chromeBinding maps each chrome token to the Dear ImGui color slots it drives, by name
// (native.SetStyleColor resolves the name to the version-specific enum). One token may
// feed several slots, which is how a small palette styles the whole shell — e.g. the
// accent paints the active tab, checkmark, slider grab, and selection. Each ImGui slot
// appears under exactly one token, so apply order does not matter.
var chromeBinding = map[types.ThemeToken][]string{
	types.TokenChromeWindowBg:      {"WindowBg"},
	types.TokenChromePanelBg:       {"ChildBg"},
	types.TokenChromePopupBg:       {"PopupBg"},
	types.TokenChromeMenuBarBg:     {"MenuBarBg"},
	types.TokenChromeHeaderBg:      {"TitleBg", "TitleBgCollapsed", "Header"},
	types.TokenChromeText:          {"Text", "InputTextCursor"},
	types.TokenChromeTextDisabled:  {"TextDisabled"},
	types.TokenChromeBorder:        {"Border", "Separator"},
	types.TokenChromeControlBg:     {"FrameBg"},
	types.TokenChromeControlHover:  {"FrameBgHovered", "SeparatorHovered"},
	types.TokenChromeControlActive: {"FrameBgActive"},
	types.TokenChromeButton:        {"Button", "Tab", "TabDimmed"},
	types.TokenChromeButtonHover:   {"ButtonHovered"},
	types.TokenChromeButtonActive:  {"ButtonActive", "TabDimmedSelected"},
	types.TokenChromeAccent: {
		"TitleBgActive", "TabSelected", "TabHovered", "CheckMark",
		"SliderGrab", "SliderGrabActive", "HeaderHovered", "HeaderActive",
		"SeparatorActive", "ScrollbarGrabActive", "TextSelectedBg",
	},
	types.TokenChromeScrollbar: {"ScrollbarBg", "ScrollbarGrab", "ScrollbarGrabHovered"},
}

// applyChromeStyle pushes every chrome token's color into the ImGui slots it binds to.
func applyChromeStyle(t contract.Theme) {
	for tok, slots := range chromeBinding {
		c := t.Color(tok).Array()
		for _, slot := range slots {
			native.SetStyleColor(slot, c)
		}
	}
}
