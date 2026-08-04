// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerApplicationHandlers wires the host-application info methods an add-in can
// query at runtime (the ThisApplication version surface).
func (r *Router) registerApplicationHandlers() {
	r.readOnly(wire.MethodApplicationApiVersion, applicationApiVersion)
	r.readOnly(wire.MethodApplicationGetHUDOptions, getHUDOptions)
	r.mutating(wire.MethodApplicationSetHUDOptions, "", typed(setHUDOptions))
}

// getHUDOptions serves wire.MethodApplicationGetHUDOptions: the in-canvas sketch input
// configuration (#2014).
func getHUDOptions(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(hudOptionsView(s.HUDOptions()))
}

// setHUDOptions serves wire.MethodApplicationSetHUDOptions. Every field is replaced, so a
// caller changing one setting reads first.
func setHUDOptions(s *app.Session, in wire.HeadsUpDisplayOptionsView) (wire.HeadsUpDisplayOptionsView, error) {
	opts := types.HeadsUpDisplayOptions{
		Enabled:                              in.Enabled,
		PointerInputEnabled:                  in.PointerInputEnabled,
		PointerInputInCartesianCoordinates:   in.PointerInputInCartesianCoordinates,
		DimensionInputEnabled:                in.DimensionInputEnabled,
		DimensionInputInCartesianCoordinates: in.DimensionInputInCartesianCoordinates,
		CreateDimensionsOnValueInput:         in.CreateDimensionsOnValueInput,
	}
	if err := s.SetHUDOptions(opts); err != nil {
		return wire.HeadsUpDisplayOptionsView{}, err
	}
	return hudOptionsView(s.HUDOptions()), nil
}

// hudOptionsView renders the session's configuration as its wire view.
func hudOptionsView(o types.HeadsUpDisplayOptions) wire.HeadsUpDisplayOptionsView {
	return wire.HeadsUpDisplayOptionsView{
		Enabled:                              o.Enabled,
		PointerInputEnabled:                  o.PointerInputEnabled,
		PointerInputInCartesianCoordinates:   o.PointerInputInCartesianCoordinates,
		DimensionInputEnabled:                o.DimensionInputEnabled,
		DimensionInputInCartesianCoordinates: o.DimensionInputInCartesianCoordinates,
		CreateDimensionsOnValueInput:         o.CreateDimensionsOnValueInput,
	}
}

// applicationApiVersion reports the semantic version of the api contract this host
// implements (wire.MethodApplicationApiVersion). It is sourced from api.Version, the
// same constant the load-time ObkAddInApiMajor handshake gates on, so the runtime
// answer can never disagree with the compatibility check.
func applicationApiVersion(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ApplicationApiVersionResult{
		Version: api.Version,
		Major:   api.Major(),
		Minor:   api.Minor(),
	})
}
