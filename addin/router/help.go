// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerHelpHandlers wires the help-routing and language methods (M05-F14, #621).
func (r *Router) registerHelpHandlers() {
	r.handlers[wire.MethodHelpRegisterContext] = registerHelpContext
	r.handlers[wire.MethodHelpDisplay] = displayHelp
	r.handlers[wire.MethodHelpPath] = helpPath
	r.handlers[wire.MethodLanguageInfo] = languageInfo
}

func registerHelpContext(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.RegisterHelpContextArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.RegisterHelpContext(req.Source, req.Base); err != nil {
		return nil, err
	}
	return ok()
}

func displayHelp(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.DisplayHelpArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.DisplayHelpTopic(req.Source, req.Topic); err != nil {
		return nil, err
	}
	return ok()
}

func helpPath(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.DisplayHelpArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	base, err := s.HelpPath(req.Source)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.HelpPathResult{Source: req.Source, Base: base})
}

func languageInfo(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.LanguageInfoResult{Locale: app.Locale()})
}
