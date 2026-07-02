//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/icon"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/theme"
)

// Theme-driven colors for the viewport overlays and ribbon icons. They live here in one
// place (rather than scattered across the overlay files) because they all share one
// owner: the active theme. They are seeded from the Dark built-in so they hold correct
// values before the first applyThemeIfChanged and in any headless head test, then
// overwritten from the active theme whenever it changes (ADR-0021). The overlay files
// read these vars by name exactly as before.
// themePalette is the theme cache the overlay builders and icon cache read each
// frame: every themed overlay/icon color in ONE object instead of ~16 mutable
// package arrays (#1617, audit B6). Single writer: applyThemeIfChanged, at the
// top of DrawChrome — the UI loop owns the palette's lifecycle; nothing else may
// mutate it. Seeded from the Dark built-in so it holds correct values before the
// first frame and in any headless head test.
type themePalette struct {
	gridMinorColor, gridMajorColor                         [4]float32
	sketchColor, sketchSelectedColor, sketchCandidateColor [4]float32
	previewColor                                           [4]float32 // rubber-band preview
	dimensionColor, dimensionDrivenColor                   [4]float32
	snapGlyphColor, pointMarkerColor                       [4]float32
	activeViewportBorderColor                              [4]float32 // focused split-view tile outline
	faintPlaneColor, hoverPlaneColor, selectedPlaneColor   [4]float32
	planeFillColor                                         [4]float32      // translucent plane fill (alpha = opacity)
	selectionHighlight                                     [4]float32      // picked-body highlight
	iconColors                                             icon.RoleColors // glyph layer colors (icon.* tokens)
	accentColor                                            [4]float32      // active/toggled ribbon button
	dangerColor                                            [4]float32      // required-but-empty selector / error affordances
	windowClearColor                                       [3]float32      // swapchain (chrome) clear
}

// chromeTheme is the UI loop's palette instance, constructed once from the Dark
// default (no init() side effects — #1617).
var chromeTheme = defaultThemePalette()

// defaultThemePalette seeds a palette from the built-in Dark theme.
func defaultThemePalette() themePalette {
	var p themePalette
	p.refresh(theme.DefaultDark())
	return p
}

// appliedThemeRevision / appliedColorSchemeRevision are the library + color-scheme revisions
// last pushed into Dear ImGui and the color vars, so applyThemeIfChanged is a cheap no-op on
// frames neither has changed.
var appliedThemeRevision uint64
var appliedColorSchemeRevision uint64

// applyThemeIfChanged re-styles the UI from the active theme when it (or the active color
// scheme) has changed since the last frame — pushing the chrome colors into Dear ImGui's
// persistent style (set once per change, not per widget), refreshing the overlay/icon color
// vars, and updating the 3D viewport's clear color. Called at the top of DrawChrome; this is
// the live-preview hook (a theme switch, a color edit, or a color-scheme switch bumps a revision).
func applyThemeIfChanged(win *native.Window, s *app.Session) {
	rev, schemeRev := s.ThemeRevision(), s.ColorSchemeRevision()
	if rev == appliedThemeRevision && schemeRev == appliedColorSchemeRevision {
		return
	}
	appliedThemeRevision, appliedColorSchemeRevision = rev, schemeRev
	active := s.Theme()
	applyChromeStyle(active)
	chromeTheme.refresh(active)
	applyColorSchemeColors(s)
	vp := viewportClear(s)
	win.SetViewportClear(vp.R, vp.G, vp.B)
}

// viewportClear is the 3D background color: the active color scheme owns the viewport
// background (M16-F06 #642), so its screen color overrides the theme's viewport-bg token.
func viewportClear(s *app.Session) theme.Rgba { return s.ActiveColorScheme().Screen.Rgba() }

// applyColorSchemeColors pushes the active scheme's highlight color into the palette
// so a scheme switch repaints selection highlighting along with the background.
func applyColorSchemeColors(s *app.Session) {
	h := s.ActiveColorScheme().Highlight.Rgba()
	chromeTheme.selectionHighlight = [4]float32{h.R, h.G, h.B, 1}
}

// refresh copies the active theme's overlay/icon colors into the palette
// the overlay builders read each frame.
func (p *themePalette) refresh(t contract.Theme) {
	arr := func(tok types.ThemeToken) [4]float32 { return t.Color(tok).Array() }
	p.gridThemeColors(arr)
	p.sketchThemeColors(arr)
	p.planeThemeColors(arr)
	p.iconThemeColors(arr)
	p.selectionHighlight = arr(types.TokenSelectionHighlight)
	p.accentColor = arr(types.TokenChromeAccent)
	p.dangerColor = arr(types.TokenChromeDanger)
	c := t.Color(types.TokenChromeWindowBg)
	p.windowClearColor = [3]float32{c.R, c.G, c.B}
}

func (p *themePalette) gridThemeColors(arr func(types.ThemeToken) [4]float32) {
	p.gridMinorColor = arr(types.TokenGridMinor)
	p.gridMajorColor = arr(types.TokenGridMajor)
}

func (p *themePalette) sketchThemeColors(arr func(types.ThemeToken) [4]float32) {
	p.sketchColor = arr(types.TokenSketchGeometry)
	p.sketchSelectedColor = arr(types.TokenSketchSelected)
	p.sketchCandidateColor = arr(types.TokenSketchCandidate)
	p.previewColor = arr(types.TokenSketchPreview)
	p.dimensionColor = arr(types.TokenDimensionDriving)
	p.dimensionDrivenColor = arr(types.TokenDimensionDriven)
	p.snapGlyphColor = arr(types.TokenSnapGlyph)
	p.activeViewportBorderColor = arr(types.TokenViewportActiveBorder)
	p.pointMarkerColor = p.sketchColor // placed points match the sketch wireframe
}

func (p *themePalette) planeThemeColors(arr func(types.ThemeToken) [4]float32) {
	p.faintPlaneColor = arr(types.TokenPlaneFaint)
	p.hoverPlaneColor = arr(types.TokenPlaneHover)
	p.selectedPlaneColor = arr(types.TokenPlaneSelected)
	p.planeFillColor = arr(types.TokenPlaneFill)
}

// refreshIconThemeColors maps the icon.* tokens onto the glyph color roles. The icon
// cache composes its masks with these on the next texture lookup after a theme change
// (the revision flush in iconCache.beginFrame).
func (p *themePalette) iconThemeColors(arr func(types.ThemeToken) [4]float32) {
	p.iconColors[icon.RolePrimary] = arr(types.TokenIconPrimary)
	p.iconColors[icon.RoleSecondary] = arr(types.TokenIconSecondary)
	p.iconColors[icon.RoleTertiary] = arr(types.TokenIconTertiary)
	p.iconColors[icon.RoleBackground] = arr(types.TokenIconBackground)
}

// WindowClearColor is the themed background the frame loop clears the swapchain to (the
// area behind/around the docked panels). The main loop passes it to Window.EndFrame.
func WindowClearColor() (r, g, b float32) {
	return chromeTheme.windowClearColor[0], chromeTheme.windowClearColor[1], chromeTheme.windowClearColor[2]
}

// chromeBinding maps each chrome token to the Dear ImGui color slots it drives, by name
// (native.SetStyleColor resolves the name to the version-specific enum). One token may
// feed several slots, which is how a small palette styles the whole shell — e.g. the
// accent paints the active tab, checkmark, slider grab, and selection. Each ImGui slot
// appears under exactly one token, so apply order does not matter.
var chromeBinding = map[types.ThemeToken][]string{
	types.TokenChromeWindowBg:     {"WindowBg"},
	types.TokenChromePanelBg:      {"ChildBg"},
	types.TokenChromePopupBg:      {"PopupBg"},
	types.TokenChromeMenuBarBg:    {"MenuBarBg"},
	types.TokenChromeHeaderBg:     {"TitleBg", "TitleBgCollapsed", "Header"},
	types.TokenChromeText:         {"Text", "InputTextCursor"},
	types.TokenChromeTextDisabled: {"TextDisabled"},
	types.TokenChromeBorder:       {"Border", "Separator"},
	// ScrollbarBg (the track groove) shares the recessed control background so the grab — painted
	// by TokenChromeScrollbar below — actually contrasts against it. Binding both the track and the
	// grab to one token left the grab invisible until dragged (Oblikovati#1471 follow-up).
	types.TokenChromeControlBg:     {"FrameBg", "ScrollbarBg"},
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
	// The scrollbar GRAB (the draggable slider) — the track is TokenChromeControlBg above and the
	// dragged state is ScrollbarGrabActive under the accent, so the grab is visible at rest, not
	// only while dragging (Oblikovati#1471 follow-up).
	types.TokenChromeScrollbar: {"ScrollbarGrab", "ScrollbarGrabHovered"},
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
