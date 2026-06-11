// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerUIShellHandlers wires the M05-F12 shell methods: command search,
// marking menus, context-menu injection, and object visibility (#619).
func (r *Router) registerUIShellHandlers() {
	r.handlers[wire.MethodUISearch] = searchCommands
	r.handlers[wire.MethodUIGetMarkingMenu] = getMarkingMenu
	r.handlers[wire.MethodUISetMarkingMenu] = setMarkingMenu
	r.handlers[wire.MethodUISetContextMenu] = setContextMenu
	r.handlers[wire.MethodUIGetObjectVisibility] = getObjectVisibility
	r.handlers[wire.MethodUISetObjectVisibility] = setObjectVisibility
	r.handlers[wire.MethodUIRegisterEnvironment] = registerEnvironment
	r.handlers[wire.MethodUIActivateEnvironment] = activateEnvironment
}

// searchCommands finds registered commands matching the query (wire ui.search).
func searchCommands(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SearchCommandsArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	hits := s.SearchCommands(req.Query)
	out := make([]wire.CommandInfo, len(hits))
	for i, c := range hits {
		out[i] = wire.CommandInfo{
			ID: c.ID(), DisplayName: c.DisplayName(), Tab: c.Tab(), Category: c.Category(),
			Alias: c.Alias(), Tooltip: c.Tooltip(), Icon: c.Icon(),
			ButtonStyle: c.ButtonStyle(), Enabled: c.IsEnabled(s),
		}
	}
	return json.Marshal(wire.SearchCommandsResult{Commands: out})
}

func getMarkingMenu(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.GetMarkingMenuArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	return json.Marshal(s.MarkingMenu(req.Environment))
}

func setMarkingMenu(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetMarkingMenuArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.SetMarkingMenu(req.Menu); err != nil {
		return nil, err
	}
	return ok()
}

func setContextMenu(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetContextMenuArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.SetContextMenuItems(req.AddIn, req.Kind, req.Items); err != nil {
		return nil, err
	}
	return ok()
}

func getObjectVisibility(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(s.ObjectVisibility())
}

func setObjectVisibility(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetObjectVisibilityArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	s.SetObjectVisibility(req.Visibility)
	return ok()
}
