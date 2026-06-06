//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"bytes"

	"oblikovati/api/types"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/theme"
)

// themeNameBuf backs the "new theme name" field across frames (ImGui edits it in place);
// themeStatus shows the last action's error, or "" when fine.
var (
	themeNameBuf [64]byte
	themeStatus  string
)

// drawAppearanceTab is the Preferences ▸ Appearance pane: pick a theme, duplicate the
// active one into an editable custom, and recolor its tokens with a live preview (every
// edit bumps the theme revision, so the whole UI restyles next frame). Built-in themes
// are read-only; the user duplicates to customize (ADR-0021).
func drawAppearanceTab(s *app.Session) {
	drawThemeSelector(s)
	drawThemeActions(s)
	native.Separator()
	drawColorRows(s)
}

// drawThemeSelector renders the theme dropdown; choosing an entry activates it (and
// persists the choice).
func drawThemeSelector(s *app.Session) {
	lib := s.Themes()
	if !native.BeginCombo("Theme", lib.ActiveName()) {
		return
	}
	for _, t := range lib.Themes() {
		if native.Selectable(t.Name(), t.Name() == lib.ActiveName()) {
			themeStatus = errText(s.SetActiveTheme(t.Name()))
		}
	}
	native.EndCombo()
}

// drawThemeActions renders the duplicate field/button plus Save/Delete for a custom
// active theme. Save writes the active theme's current colors to disk; Delete removes it.
func drawThemeActions(s *app.Session) {
	native.InputText("New name", themeNameBuf[:])
	native.SameLine()
	if native.Button("Duplicate") {
		duplicateActiveTheme(s)
	}
	if active := s.Themes().Active(); active.Kind().Editable() {
		native.SameLine()
		if native.Button("Save") {
			themeStatus = errText(s.SaveActiveTheme())
		}
		native.SameLine()
		if native.Button("Delete") {
			themeStatus = errText(s.DeleteTheme(active.Name()))
		}
	}
	if themeStatus != "" {
		native.Text(themeStatus)
	}
}

// duplicateActiveTheme copies the active theme into a new custom under the typed name,
// reporting the library's name/uniqueness error in the status line.
func duplicateActiveTheme(s *app.Session) {
	name := bufString(themeNameBuf[:])
	themeStatus = errText(s.DuplicateTheme(s.Themes().ActiveName(), name))
	if themeStatus == "" {
		clearBuf(themeNameBuf[:]) // consumed the typed name
	}
}

// drawColorRows lists every token as a color swatch, grouped by area, disabled (read-only)
// for a built-in theme. Editing a swatch recolors the active custom theme live.
func drawColorRows(s *app.Session) {
	active := s.Themes().Active()
	native.BeginDisabled(!active.Kind().Editable())
	var group theme.Group
	for _, tok := range types.AllThemeTokens() {
		info := theme.InfoFor(tok)
		if info.Group != group {
			group = info.Group
			native.SeparatorText(string(group))
		}
		drawColorRow(s, active, tok, info.Label)
	}
	native.EndDisabled()
}

// drawColorRow draws one token's swatch+picker; on change it recolors the active theme.
// The label carries a hidden "##token" suffix so each row has a stable, unique ImGui id.
func drawColorRow(s *app.Session, active *theme.Theme, tok types.ThemeToken, label string) {
	c := active.Color(tok).Array()
	if native.ColorEdit4(label+"##"+string(tok), &c) {
		s.Themes().EditActiveColor(tok, types.Rgba{R: c[0], G: c[1], B: c[2], A: c[3]})
	}
}

// errText renders an action error for the status line ("" on success).
func errText(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// bufString reads the NUL-terminated text ImGui wrote into an InputText buffer.
func bufString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

// clearBuf zeroes an InputText buffer (so the field empties after a successful action).
func clearBuf(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
