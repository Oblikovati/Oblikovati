// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerColorSchemeHandlers wires the application color-scheme methods (M16-F06, #642).
func (r *Router) registerColorSchemeHandlers() {
	r.handlers[wire.MethodColorSchemesList] = listColorSchemes
	r.handlers[wire.MethodColorSchemesGetActive] = getActiveColorScheme
	r.handlers[wire.MethodColorSchemesSetActive] = setActiveColorScheme
}

// listColorSchemes returns every application color scheme, flagging the active one
// (wire.MethodColorSchemesList).
func listColorSchemes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	schemes := s.ColorSchemes()
	active := schemes.Active().Name()
	out := make([]wire.ColorSchemeView, schemes.Count())
	for i := range out {
		out[i] = colorSchemeView(schemes, schemes.Item(i), active)
	}
	return json.Marshal(wire.ColorSchemesResult{Schemes: out})
}

// getActiveColorScheme returns the active color scheme (wire.MethodColorSchemesGetActive).
func getActiveColorScheme(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	schemes := s.ColorSchemes()
	return json.Marshal(colorSchemeView(schemes, schemes.Active(), schemes.Active().Name()))
}

// setActiveColorScheme activates a scheme by name and echoes it, erroring on an unknown name
// (wire.MethodColorSchemesSetActive).
func setActiveColorScheme(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetColorSchemeArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.SetColorScheme(a.Name); err != nil {
		return nil, err
	}
	schemes := s.ColorSchemes()
	return json.Marshal(colorSchemeView(schemes, schemes.Active(), schemes.Active().Name()))
}

// colorSchemeView projects one contract scheme into its wire shape, carrying the registry's
// background type and flagging the active scheme.
func colorSchemeView(schemes contract.ColorSchemes, c contract.ColorScheme, active string) wire.ColorSchemeView {
	return wire.ColorSchemeView{
		Name:           c.Name(),
		Active:         c.Name() == active,
		BackgroundType: schemes.BackgroundType(),
		ScreenColor:    c.ScreenColor(),
		TopScreen:      c.TopScreenColor(),
		BottomScreen:   c.BottomScreenColor(),
		Highlight:      c.HighlightColor(),
		PrimarySelect:  c.PrimarySelectColor(),
		SecondSelect:   c.SecondarySelectColor(),
	}
}
