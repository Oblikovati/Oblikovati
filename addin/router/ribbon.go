// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
)

// ribbonList returns the ribbon currently shown for the active document (wire ribbon.list /
// Inventor's "list the contents of the ribbon"), so an add-in can discover the tab and panel
// internal names to insert its controls into. It mirrors exactly what the shell renders — the
// ribbon for the active document's type (ZeroDoc when none is open), with contextual tabs
// present only when their environment is active.
func ribbonList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	r := app.BuildRibbon(s)
	tabs := make([]wire.RibbonTabInfo, len(r.Tabs))
	for i, t := range r.Tabs {
		tabs[i] = wire.RibbonTabInfo{Name: t.Name, Panels: panelInfos(t.Panels)}
	}
	return json.Marshal(wire.ListRibbonResult{Key: r.Key, Tabs: tabs})
}

// panelInfos maps a tab's panels (and their buttons) to the wire shape.
func panelInfos(panels []app.RibbonPanel) []wire.RibbonPanelInfo {
	out := make([]wire.RibbonPanelInfo, len(panels))
	for i, p := range panels {
		controls := make([]wire.RibbonControlInfo, len(p.Buttons))
		for j, b := range p.Buttons {
			controls[j] = wire.RibbonControlInfo{
				CommandID:   b.Command.ID(),
				DisplayName: b.Command.DisplayName(),
			}
		}
		out[i] = wire.RibbonPanelInfo{Name: p.Name, Controls: controls}
	}
	return out
}
