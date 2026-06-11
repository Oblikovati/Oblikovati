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
	r.handlers[wire.MethodAddInsList] = listAddIns
	r.handlers[wire.MethodAddInsGet] = getAddIn
	r.handlers[wire.MethodAddInsActivate] = activateAddIn
	r.handlers[wire.MethodAddInsDeactivate] = deactivateAddIn
	r.handlers[wire.MethodAddInsSetLoadBehavior] = setAddInLoadBehavior
	r.handlers[wire.MethodAddInsCallAutomation] = callAddInAutomation
	r.handlers[wire.MethodClientAppsRegister] = registerClientApp
	r.handlers[wire.MethodClientAppsUnregister] = unregisterClientApp
	r.handlers[wire.MethodClientAppsList] = listClientApps
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
func getAddIn(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var ref wire.AddInRefArgs
	if err := decode(args, &ref); err != nil {
		return nil, err
	}
	info, err := s.AddIns().Describe(ref.ID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(info)
}

// activateAddIn runs the add-in's activation (wire.MethodAddInsActivate).
func activateAddIn(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var ref wire.AddInRefArgs
	if err := decode(args, &ref); err != nil {
		return nil, err
	}
	if err := s.AddIns().Activate(s, ref.ID); err != nil {
		return nil, err
	}
	return ok()
}

// deactivateAddIn runs the add-in's shutdown (wire.MethodAddInsDeactivate).
func deactivateAddIn(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var ref wire.AddInRefArgs
	if err := decode(args, &ref); err != nil {
		return nil, err
	}
	if err := s.AddIns().Deactivate(s, ref.ID); err != nil {
		return nil, err
	}
	return ok()
}

// setAddInLoadBehavior persists when the host activates the add-in on startup
// (wire.MethodAddInsSetLoadBehavior).
func setAddInLoadBehavior(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetAddInLoadBehaviorArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.AddIns().SetLoadBehavior(req.ID, req.LoadBehavior); err != nil {
		return nil, err
	}
	return ok()
}

// callAddInAutomation routes a call to another add-in's automation surface
// (wire.MethodAddInsCallAutomation).
func callAddInAutomation(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.CallAddInAutomationArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	out, err := s.AddIns().CallAutomation(req.ID, req.Method, req.Args)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.CallAddInAutomationResult{Result: out})
}

// registerClientApp announces an external client application
// (wire.MethodClientAppsRegister).
func registerClientApp(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.RegisterClientApplicationArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	id, err := s.ClientApps().Register(req.Name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.RegisterClientApplicationResult{ID: id})
}

// unregisterClientApp removes an external client application
// (wire.MethodClientAppsUnregister).
func unregisterClientApp(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.UnregisterClientApplicationArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.ClientApps().Unregister(req.ID); err != nil {
		return nil, err
	}
	return ok()
}

// listClientApps returns the registered external clients (wire.MethodClientAppsList).
func listClientApps(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListClientApplicationsResult{Clients: s.ClientApps().List()})
}
