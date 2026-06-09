// SPDX-License-Identifier: GPL-2.0-only

// Package theme owns the application UI's color themes: the semantic-token palette the
// shell styles itself from, the shipped Light and Dark built-ins, the in-memory library
// of built-ins plus user customs with an active selection, and load/save of customs to
// the user config directory. It is the GPL implementation of the Apache-2.0
// [oblikovati.org/api/contract.Theme] contract (ADR-0021).
//
// Themes cover the application shell only — window, menus, viewport 2D overlays, and 3D
// gizmos. 3D body appearance belongs to the (future) material/appearance subsystem.
//
// This package is pure data + IO with no cgo or ImGui dependency, so it is fully
// headless-testable; the head (head/ui) maps tokens onto Dear ImGui colors and the
// renderer's overlay colors.
package theme

import "oblikovati.org/api/types"

// Token, Rgba, and Kind alias the canonical Apache-2.0 definitions so callers program
// against the theme package without importing api/types directly (the rest of this
// module follows the same aliasing rule as model/param — ADR-0018).
type (
	Token = types.ThemeToken
	Rgba  = types.Rgba
	Kind  = types.ThemeKind
)

const (
	KindLight  = types.ThemeLight
	KindDark   = types.ThemeDark
	KindCustom = types.ThemeCustom
)

// Group is the editor heading a token is shown under — the four areas a theme styles.
type Group string

const (
	GroupChrome   Group = "Chrome"
	GroupViewport Group = "Viewport 2D"
	GroupGizmos   Group = "Gizmos 3D"
	GroupIcons    Group = "Icons"
)

// TokenInfo is the editor metadata for one token: a human label and the group it shows
// under. The token's color lives in the [Palette], not here.
type TokenInfo struct {
	Token Token
	Label string
	Group Group
}

// tokenInfos is the display table for every token, in [types.AllThemeTokens] order. The
// editor iterates it to lay out grouped color rows; a missing entry falls back to the
// token's raw string (see [InfoFor]), so a new token is never invisible.
var tokenInfos = map[Token]TokenInfo{
	types.TokenChromeWindowBg:       {types.TokenChromeWindowBg, "Window background", GroupChrome},
	types.TokenChromePanelBg:        {types.TokenChromePanelBg, "Panel background", GroupChrome},
	types.TokenChromePopupBg:        {types.TokenChromePopupBg, "Popup background", GroupChrome},
	types.TokenChromeMenuBarBg:      {types.TokenChromeMenuBarBg, "Menu bar", GroupChrome},
	types.TokenChromeHeaderBg:       {types.TokenChromeHeaderBg, "Header / title", GroupChrome},
	types.TokenChromeText:           {types.TokenChromeText, "Text", GroupChrome},
	types.TokenChromeTextDisabled:   {types.TokenChromeTextDisabled, "Text (disabled)", GroupChrome},
	types.TokenChromeBorder:         {types.TokenChromeBorder, "Border", GroupChrome},
	types.TokenChromeControlBg:      {types.TokenChromeControlBg, "Control background", GroupChrome},
	types.TokenChromeControlHover:   {types.TokenChromeControlHover, "Control (hover)", GroupChrome},
	types.TokenChromeControlActive:  {types.TokenChromeControlActive, "Control (active)", GroupChrome},
	types.TokenChromeButton:         {types.TokenChromeButton, "Button", GroupChrome},
	types.TokenChromeButtonHover:    {types.TokenChromeButtonHover, "Button (hover)", GroupChrome},
	types.TokenChromeButtonActive:   {types.TokenChromeButtonActive, "Button (active)", GroupChrome},
	types.TokenChromeAccent:         {types.TokenChromeAccent, "Accent", GroupChrome},
	types.TokenChromeScrollbar:      {types.TokenChromeScrollbar, "Scrollbar", GroupChrome},
	types.TokenViewportBg:           {types.TokenViewportBg, "Background", GroupViewport},
	types.TokenGridMinor:            {types.TokenGridMinor, "Grid (minor)", GroupViewport},
	types.TokenGridMajor:            {types.TokenGridMajor, "Grid (major)", GroupViewport},
	types.TokenGridAxis:             {types.TokenGridAxis, "Grid (axis)", GroupViewport},
	types.TokenSketchGeometry:       {types.TokenSketchGeometry, "Sketch geometry", GroupViewport},
	types.TokenSketchSelected:       {types.TokenSketchSelected, "Sketch (selected)", GroupViewport},
	types.TokenSketchCandidate:      {types.TokenSketchCandidate, "Sketch (candidate)", GroupViewport},
	types.TokenSketchPreview:        {types.TokenSketchPreview, "Sketch preview", GroupViewport},
	types.TokenDimensionDriving:     {types.TokenDimensionDriving, "Dimension (driving)", GroupViewport},
	types.TokenDimensionDriven:      {types.TokenDimensionDriven, "Dimension (driven)", GroupViewport},
	types.TokenSnapGlyph:            {types.TokenSnapGlyph, "Snap glyph", GroupViewport},
	types.TokenViewportActiveBorder: {types.TokenViewportActiveBorder, "Active view border", GroupViewport},
	types.TokenPlaneFaint:           {types.TokenPlaneFaint, "Work plane", GroupGizmos},
	types.TokenPlaneHover:           {types.TokenPlaneHover, "Work plane (hover)", GroupGizmos},
	types.TokenPlaneSelected:        {types.TokenPlaneSelected, "Work plane (selected)", GroupGizmos},
	types.TokenPlaneFill:            {types.TokenPlaneFill, "Work plane fill (alpha = opacity)", GroupGizmos},
	types.TokenSelectionHighlight:   {types.TokenSelectionHighlight, "Selection highlight", GroupGizmos},
	types.TokenIconTint:             {types.TokenIconTint, "Icon tint", GroupIcons},
	types.TokenIconDisabled:         {types.TokenIconDisabled, "Icon (disabled)", GroupIcons},
}

// InfoFor returns the editor metadata for a token, falling back to a self-labeled entry
// in the Chrome group for any token without an explicit row (so the editor stays robust
// to a token added without a label).
func InfoFor(t Token) TokenInfo {
	if info, ok := tokenInfos[t]; ok {
		return info
	}
	return TokenInfo{Token: t, Label: string(t), Group: GroupChrome}
}
