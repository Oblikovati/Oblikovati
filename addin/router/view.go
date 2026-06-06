// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati/api/types"
	"oblikovati/api/wire"
	"oblikovati/app"
)

// getDisplayMode returns the viewport's current display mode and label
// (wire.MethodViewGetDisplayMode).
func getDisplayMode(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return marshalDisplayMode(s.DisplayMode())
}

// setDisplayMode switches the viewport to the requested display mode and echoes the result,
// erroring on an unknown mode (wire.MethodViewSetDisplayMode).
func setDisplayMode(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetDisplayModeArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.SetDisplayMode(a.Mode); err != nil {
		return nil, err
	}
	return marshalDisplayMode(s.DisplayMode())
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

// marshalDisplayMode encodes a mode + its label as the shared DisplayModeView reply.
func marshalDisplayMode(m types.DisplayModeEnum) (json.RawMessage, error) {
	return json.Marshal(wire.DisplayModeView{Mode: m, Name: m.String()})
}
