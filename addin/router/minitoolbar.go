// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerMiniToolbarHandlers wires the in-canvas mini-toolbar methods (M05-F07, #614).
func (r *Router) registerMiniToolbarHandlers() {
	r.readOnly(wire.MethodMiniToolbarSet, typed(setMiniToolbar))
	r.readOnly(wire.MethodMiniToolbarUpdate, typed(updateMiniToolbar))
	r.readOnly(wire.MethodMiniToolbarRemove, typed(removeMiniToolbar))
	r.readOnly(wire.MethodMiniToolbarList, listMiniToolbars)
}

func setMiniToolbar(s *app.Session, in wire.SetMiniToolbarArgs) (wire.OKResult, error) {
	if err := s.SetMiniToolbar(in.Toolbar); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func updateMiniToolbar(s *app.Session, in wire.UpdateMiniToolbarArgs) (wire.OKResult, error) {
	if err := s.UpdateMiniToolbarControls(in.ID, in.Controls); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func removeMiniToolbar(s *app.Session, in wire.RemoveMiniToolbarArgs) (wire.OKResult, error) {
	if err := s.RemoveMiniToolbar(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func listMiniToolbars(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListMiniToolbarsResult{Toolbars: s.MiniToolbars().List()})
}
