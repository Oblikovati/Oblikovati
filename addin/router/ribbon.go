// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// ribbonList returns the ribbon currently shown for the active document (wire ribbon.list):
// the typed ribbon object model of M05-F03 — tabs, panels (with their selector when they
// render as a selection box), and each control's kind, look, live state and dropdown items.
// It mirrors exactly what the shell renders — the ribbon for the active document's type
// (ZeroDoc when none is open), with contextual tabs present only when their environment is
// active.
func ribbonList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	r := app.BuildRibbon(s)
	tabs := make([]wire.RibbonTabInfo, len(r.Tabs))
	for i, t := range r.Tabs {
		tabs[i] = wire.RibbonTabInfo{Name: t.Name, Panels: panelInfos(t.Panels)}
	}
	return json.Marshal(wire.ListRibbonResult{Key: r.Key, Tabs: tabs})
}

// panelInfos maps a tab's panels (and their buttons or selector) to the wire shape.
func panelInfos(panels []app.RibbonPanel) []wire.RibbonPanelInfo {
	out := make([]wire.RibbonPanelInfo, len(panels))
	for i, p := range panels {
		info := wire.RibbonPanelInfo{Name: p.Name}
		if p.Selector != nil {
			info.Selector = selectorInfo(p.Selector)
		} else {
			info.Controls = controlInfos(p.Buttons)
		}
		out[i] = info
	}
	return out
}

// controlInfos maps a panel's buttons to the wire control shape.
func controlInfos(buttons []app.RibbonButton) []wire.RibbonControlInfo {
	out := make([]wire.RibbonControlInfo, len(buttons))
	for i, b := range buttons {
		out[i] = wire.RibbonControlInfo{
			CommandID:   b.Command.ID(),
			DisplayName: b.Command.DisplayName(),
			Kind:        b.Command.Kind(),
			ButtonStyle: b.Command.ButtonStyle(),
			Icon:        b.Command.Icon(),
			Tooltip:     b.Command.Tooltip(),
			Alias:       b.Command.Alias(),
			Enabled:     b.Enabled,
			Active:      b.Active,
			Items:       itemInfos(b.Variants),
		}
	}
	return out
}

// itemInfos maps a button's resolved dropdown entries (split-button variants or popup
// items) to the wire shape.
func itemInfos(variants []app.RibbonVariant) []wire.RibbonItemInfo {
	if len(variants) == 0 {
		return nil
	}
	out := make([]wire.RibbonItemInfo, len(variants))
	for i, v := range variants {
		out[i] = wire.RibbonItemInfo{CommandID: v.CommandID, Label: v.Label, Tooltip: v.Tooltip, Enabled: v.Enabled}
	}
	return out
}

// selectorInfo maps a selection-box panel to the wire shape.
func selectorInfo(sel *app.RibbonSelector) *wire.RibbonSelectorInfo {
	options := make([]wire.RibbonItemInfo, len(sel.Options))
	for i, o := range sel.Options {
		options[i] = wire.RibbonItemInfo{CommandID: o.CommandID, Label: o.Label, Tooltip: o.Tooltip, Enabled: true}
	}
	return &wire.RibbonSelectorInfo{Options: options, SelectedIndex: sel.SelectedIndex}
}

// listEnvironments returns the UI environments the command framework scopes by —
// the built-ins plus every registered add-in environment — flagging the active one
// (wire ui.listEnvironments, M05-F12/F16).
func listEnvironments(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := app.CurrentEnvironment(s)
	envs := []wire.EnvironmentInfo{
		{Environment: types.BaseEnvironment, Name: "Base", Active: active == types.BaseEnvironment},
		{Environment: types.SketchEnvironment, Name: "Sketch", Active: active == types.SketchEnvironment},
	}
	for env, name := range s.AddInEnvironments() {
		envs = append(envs, wire.EnvironmentInfo{Environment: env, Name: name, Active: active == env})
	}
	return json.Marshal(wire.ListEnvironmentsResult{Environments: envs})
}

// registerEnvironment declares an add-in environment (wire ui.registerEnvironment).
func registerEnvironment(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.RegisterEnvironmentArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.RegisterEnvironment(req.Environment, req.Name); err != nil {
		return nil, err
	}
	return ok()
}

// activateEnvironment enters/leaves an add-in environment (wire ui.activateEnvironment).
func activateEnvironment(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ActivateEnvironmentArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.ActivateEnvironment(req.Environment); err != nil {
		return nil, err
	}
	return ok()
}
