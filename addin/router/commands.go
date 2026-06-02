// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
)

// listCommands returns every registered command and whether it can run now.
func listCommands(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	all := s.Commands().All()
	out := make([]wire.CommandInfo, len(all))
	for i, c := range all {
		out[i] = wire.CommandInfo{
			ID: c.ID(), DisplayName: c.DisplayName(), Tab: c.Tab(),
			Category: c.Category(), Alias: c.Alias(), Tooltip: c.Tooltip(),
			Icon: c.Icon(), ButtonStyle: c.ButtonStyle(),
			Enabled: c.IsEnabled(s),
		}
	}
	return json.Marshal(wire.ListCommandsResult{Commands: out})
}

// createCommand registers a ribbon button on behalf of an add-in (wire commands.create
// / Inventor's ButtonDefinition). The command runs no host logic — clicking it fires a
// command.ended event the add-in handles in its Notify entry point — so the host stays
// agnostic to the add-in's action. It errors on a missing id/displayName or a duplicate id.
func createCommand(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.CreateCommandArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if in.ID == "" || in.DisplayName == "" {
		return nil, fmt.Errorf("commands.create: id and displayName are required, got id=%q displayName=%q", in.ID, in.DisplayName)
	}
	cmd := app.NewCommand(in.ID, in.DisplayName, in.Category, func(*app.Session) error { return nil }).
		WithTab(in.Tab).WithAlias(in.Alias).WithTooltip(in.Tooltip).
		WithIcon(in.Icon).WithButtonStyle(in.ButtonStyle)
	if err := s.Commands().Add(cmd); err != nil {
		return nil, err
	}
	return ok()
}

// executeCommand runs the command with the given id (the same path a ribbon click
// takes), surfacing a disabled/unknown command as an error.
func executeCommand(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ExecuteCommandArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if in.ID == "" {
		return nil, errors.New("commands.execute: id is required")
	}
	if err := s.Execute(in.ID); err != nil {
		return nil, err
	}
	return ok()
}
