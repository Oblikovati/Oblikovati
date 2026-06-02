// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"

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
			Enabled: c.IsEnabled(s),
		}
	}
	return json.Marshal(wire.ListCommandsResult{Commands: out})
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
