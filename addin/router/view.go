// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// getDisplayMode returns the viewport's current display mode and label
// (wire.MethodViewGetDisplayMode).
func getDisplayMode(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(displayModeView(s.DisplayMode()))
}

// setDisplayMode switches the viewport to the requested display mode and echoes the result,
// erroring on an unknown mode (wire.MethodViewSetDisplayMode).
func setDisplayMode(s *app.Session, a wire.SetDisplayModeArgs) (wire.DisplayModeView, error) {
	if err := s.SetDisplayMode(a.Mode); err != nil {
		return wire.DisplayModeView{}, err
	}
	return displayModeView(s.DisplayMode()), nil
}

// listDisplayModes enumerates every display mode in gallery order, flagging the active one
// (wire.MethodViewListDisplayModes).
func listDisplayModes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := s.DisplayMode()
	modes := types.AllDisplayModes()
	out := make([]wire.DisplayModeInfo, len(modes))
	for i, m := range modes {
		out[i] = wire.DisplayModeInfo{Mode: m, Name: m.String(), Active: m == active}
	}
	return json.Marshal(wire.ListDisplayModesResult{Modes: out})
}

// displayModeView projects a mode + its label onto the shared DisplayModeView reply.
func displayModeView(m types.DisplayModeEnum) wire.DisplayModeView {
	return wire.DisplayModeView{Mode: m, Name: m.String()}
}
