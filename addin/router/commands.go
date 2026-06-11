// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// listCommands returns every registered command and whether it can run now.
func listCommands(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	all := s.Commands().All()
	out := make([]wire.CommandInfo, len(all))
	for i, c := range all {
		out[i] = wire.CommandInfo{
			ID: c.ID(), DisplayName: c.DisplayName(), Ribbon: c.Ribbons()[0],
			Tab: c.Tab(), Category: c.Category(), Environment: c.Environment(),
			Alias: c.Alias(), Tooltip: c.Tooltip(),
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
	if in.Ribbon != "" && !in.Ribbon.Valid() {
		return nil, fmt.Errorf("commands.create: unknown ribbon %q (one of ZeroDoc/Part/Assembly/Drawing/Presentation/iFeatures/UnknownDocument)", in.Ribbon)
	}
	cmd := app.NewCommand(in.ID, in.DisplayName, in.Category, func(*app.Session) error { return nil }).
		WithTab(in.Tab).WithEnvironment(in.Environment).WithAlias(in.Alias).WithTooltip(in.Tooltip).
		WithIcon(in.Icon).WithButtonStyle(in.ButtonStyle).WithKind(in.Kind)
	if in.Ribbon != "" {
		cmd.WithRibbons(in.Ribbon)
	}
	// A PopupControl lists other registered commands by id (the CommandBarPopUp
	// equivalent, M05-F03); unknown ids are skipped at ribbon-build time.
	if in.Kind == app.PopupControl || len(in.Items) > 0 {
		cmd.WithPopupItems(in.Items...)
	}
	if err := s.Commands().Add(cmd); err != nil {
		return nil, err
	}
	return ok()
}

// setCommandState updates one of an add-in's commands' live ribbon state: its active
// (highlighted/accent) flag and an optional relabel (wire.MethodCommandsSetState).
func setCommandState(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.SetCommandStateArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if in.ID == "" {
		return nil, errors.New("commands.setState: id is required")
	}
	cmd, found := s.Commands().ByID(in.ID)
	if !found {
		return nil, fmt.Errorf("commands.setState: unknown command %q", in.ID)
	}
	cmd.SetActiveState(in.Active)
	if in.Enabled != nil {
		cmd.SetEnabledState(*in.Enabled)
	}
	if in.DisplayName != "" {
		cmd.SetDisplayName(in.DisplayName)
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
