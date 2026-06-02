// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
)

// themeActive returns the host's active theme — name, kind, and every semantic color as
// hex — so an add-in can paint its own panels to match the host (wire.MethodThemeActive).
func themeActive(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := s.Themes().Active()
	colors := make(map[string]string, len(types.AllThemeTokens()))
	for _, tok := range types.AllThemeTokens() {
		colors[string(tok)] = active.Color(tok).Hex()
	}
	return json.Marshal(wire.ThemeView{
		Name:   active.Name(),
		Kind:   string(active.Kind()),
		Colors: colors,
	})
}

// themeList returns a summary of every available theme (built-in and custom), flagging
// the active one (wire.MethodThemeList).
func themeList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	lib := s.Themes()
	activeName := lib.ActiveName()
	themes := lib.Themes()
	out := make([]wire.ThemeSummary, len(themes))
	for i, t := range themes {
		out[i] = wire.ThemeSummary{
			Name:   t.Name(),
			Kind:   string(t.Kind()),
			Active: t.Name() == activeName,
		}
	}
	return json.Marshal(wire.ListThemesResult{Themes: out})
}
