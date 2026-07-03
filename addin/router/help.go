// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerHelpHandlers wires the help-routing and language methods (M05-F14, #621).
func (r *Router) registerHelpHandlers() {
	r.readOnly(wire.MethodHelpRegisterContext, typed(registerHelpContext))
	r.readOnly(wire.MethodHelpDisplay, typed(displayHelp))
	r.readOnly(wire.MethodHelpPath, typed(helpPath))
	r.readOnly(wire.MethodLanguageInfo, languageInfo)
}

func registerHelpContext(s *app.Session, in wire.RegisterHelpContextArgs) (wire.OKResult, error) {
	if err := s.RegisterHelpContext(in.Source, in.Base); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func displayHelp(s *app.Session, in wire.DisplayHelpArgs) (wire.OKResult, error) {
	if err := s.DisplayHelpTopic(in.Source, in.Topic); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func helpPath(s *app.Session, in wire.DisplayHelpArgs) (wire.HelpPathResult, error) {
	base, err := s.HelpPath(in.Source)
	if err != nil {
		return wire.HelpPathResult{}, err
	}
	return wire.HelpPathResult{Source: in.Source, Base: base}, nil
}

func languageInfo(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.LanguageInfoResult{Locale: app.Locale()})
}
