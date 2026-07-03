// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerAddInHandlers wires the add-in registry/lifecycle/automation methods and
// the external client-application registry (M05-F01: #245, #251, #252).
func (r *Router) registerAddInHandlers() {
	r.readOnly(wire.MethodAddInsList, listAddIns)
	r.readOnly(wire.MethodAddInsGet, typed(getAddIn))
	r.readOnly(wire.MethodAddInsActivate, typed(activateAddIn))
	r.readOnly(wire.MethodAddInsDeactivate, typed(deactivateAddIn))
	r.readOnly(wire.MethodAddInsSetLoadBehavior, typed(setAddInLoadBehavior))
	r.readOnly(wire.MethodAddInsCallAutomation, typed(callAddInAutomation))
	r.readOnly(wire.MethodClientAppsRegister, typed(registerClientApp))
	r.readOnly(wire.MethodClientAppsUnregister, typed(unregisterClientApp))
	r.readOnly(wire.MethodClientAppsList, listClientApps)
}

// listAddIns returns every registered add-in in registration order
// (wire.MethodAddInsList).
func listAddIns(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	registry := s.AddIns()
	ids := registry.Registered()
	infos := make([]wire.AddInInfo, 0, len(ids))
	for _, id := range ids {
		info, err := registry.Describe(id)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return json.Marshal(wire.ListAddInsResult{AddIns: infos})
}

// getAddIn returns one registry entry by id (wire.MethodAddInsGet).
func getAddIn(s *app.Session, in wire.AddInRefArgs) (wire.AddInInfo, error) {
	info, err := s.AddIns().Describe(in.ID)
	if err != nil {
		return wire.AddInInfo{}, err
	}
	return info, nil
}

// activateAddIn runs the add-in's activation (wire.MethodAddInsActivate).
func activateAddIn(s *app.Session, in wire.AddInRefArgs) (wire.OKResult, error) {
	if err := s.AddIns().Activate(s, in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// deactivateAddIn runs the add-in's shutdown (wire.MethodAddInsDeactivate).
func deactivateAddIn(s *app.Session, in wire.AddInRefArgs) (wire.OKResult, error) {
	if err := s.AddIns().Deactivate(s, in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setAddInLoadBehavior persists when the host activates the add-in on startup
// (wire.MethodAddInsSetLoadBehavior).
func setAddInLoadBehavior(s *app.Session, in wire.SetAddInLoadBehaviorArgs) (wire.OKResult, error) {
	if err := s.AddIns().SetLoadBehavior(in.ID, in.LoadBehavior); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// callAddInAutomation routes a call to another add-in's automation surface
// (wire.MethodAddInsCallAutomation).
func callAddInAutomation(s *app.Session, in wire.CallAddInAutomationArgs) (wire.CallAddInAutomationResult, error) {
	out, err := s.AddIns().CallAutomation(in.ID, in.Method, in.Args)
	if err != nil {
		return wire.CallAddInAutomationResult{}, err
	}
	return wire.CallAddInAutomationResult{Result: out}, nil
}

// registerClientApp announces an external client application
// (wire.MethodClientAppsRegister).
func registerClientApp(s *app.Session, in wire.RegisterClientApplicationArgs) (wire.RegisterClientApplicationResult, error) {
	id, err := s.ClientApps().Register(in.Name)
	if err != nil {
		return wire.RegisterClientApplicationResult{}, err
	}
	return wire.RegisterClientApplicationResult{ID: id}, nil
}

// unregisterClientApp removes an external client application
// (wire.MethodClientAppsUnregister).
func unregisterClientApp(s *app.Session, in wire.UnregisterClientApplicationArgs) (wire.OKResult, error) {
	if err := s.ClientApps().Unregister(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// listClientApps returns the registered external clients (wire.MethodClientAppsList).
func listClientApps(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListClientApplicationsResult{Clients: s.ClientApps().List()})
}
