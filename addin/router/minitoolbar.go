// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerMiniToolbarHandlers wires the in-canvas mini-toolbar methods (M05-F07, #614).
func (r *Router) registerMiniToolbarHandlers() {
	r.handlers[wire.MethodMiniToolbarSet] = setMiniToolbar
	r.handlers[wire.MethodMiniToolbarUpdate] = updateMiniToolbar
	r.handlers[wire.MethodMiniToolbarRemove] = removeMiniToolbar
	r.handlers[wire.MethodMiniToolbarList] = listMiniToolbars
}

func setMiniToolbar(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetMiniToolbarArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.SetMiniToolbar(req.Toolbar); err != nil {
		return nil, err
	}
	return ok()
}

func updateMiniToolbar(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.UpdateMiniToolbarArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.UpdateMiniToolbarControls(req.ID, req.Controls); err != nil {
		return nil, err
	}
	return ok()
}

func removeMiniToolbar(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.RemoveMiniToolbarArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.RemoveMiniToolbar(req.ID); err != nil {
		return nil, err
	}
	return ok()
}

func listMiniToolbars(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListMiniToolbarsResult{Toolbars: s.MiniToolbars().List()})
}
