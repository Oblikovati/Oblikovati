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
	r.readOnly(wire.MethodUISearch, typed(searchCommands))
	r.readOnly(wire.MethodUIGetMarkingMenu, typed(getMarkingMenu))
	r.readOnly(wire.MethodUISetMarkingMenu, typed(setMarkingMenu))
	r.readOnly(wire.MethodUISetContextMenu, typed(setContextMenu))
	r.readOnly(wire.MethodUIGetObjectVisibility, getObjectVisibility)
	r.readOnly(wire.MethodUISetObjectVisibility, typed(setObjectVisibility))
	r.readOnly(wire.MethodUIRegisterEnvironment, typed(registerEnvironment))
	r.readOnly(wire.MethodUIActivateEnvironment, typed(activateEnvironment))
}

// searchCommands finds registered commands matching the query (wire ui.search).
func searchCommands(s *app.Session, in wire.SearchCommandsArgs) (wire.SearchCommandsResult, error) {
	hits := s.SearchCommands(in.Query)
	out := make([]wire.CommandInfo, len(hits))
	for i, c := range hits {
		out[i] = wire.CommandInfo{
			ID: c.ID(), DisplayName: c.DisplayName(), Tab: c.Tab(), Category: c.Category(),
			Alias: c.Alias(), Tooltip: c.Tooltip(), Icon: c.Icon(),
			ButtonStyle: c.ButtonStyle(), Enabled: c.IsEnabled(s),
		}
	}
	return wire.SearchCommandsResult{Commands: out}, nil
}

func getMarkingMenu(s *app.Session, in wire.GetMarkingMenuArgs) (wire.MarkingMenuView, error) {
	return s.MarkingMenu(in.Environment), nil
}

func setMarkingMenu(s *app.Session, in wire.SetMarkingMenuArgs) (wire.OKResult, error) {
	if err := s.SetMarkingMenu(in.Menu); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func setContextMenu(s *app.Session, in wire.SetContextMenuArgs) (wire.OKResult, error) {
	if err := s.SetContextMenuItems(in.AddIn, in.Kind, in.Items); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func getObjectVisibility(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(s.ObjectVisibility())
}

func setObjectVisibility(s *app.Session, in wire.SetObjectVisibilityArgs) (wire.OKResult, error) {
	s.SetObjectVisibility(in.Visibility)
	return wire.OKResult{OK: true}, nil
}
