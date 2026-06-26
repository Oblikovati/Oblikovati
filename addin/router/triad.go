// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerTriadHandlers wires the gizmo methods (M05-F13, #620).
func (r *Router) registerTriadHandlers() {
	r.readOnly(wire.MethodTriadShow, showTriad)
	r.readOnly(wire.MethodTriadUpdate, updateTriad)
	r.readOnly(wire.MethodTriadHide, hideTriad)
	r.readOnly(wire.MethodTriadGet, getTriad)
	r.readOnly(wire.MethodManipulatorsSet, setManipulators)
	r.readOnly(wire.MethodManipulatorsRemove, removeManipulators)
}

func showTriad(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowTriadArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	req.Triad.Visible = true
	if err := s.ShowTriad(req.Triad); err != nil {
		return nil, err
	}
	return ok()
}

func updateTriad(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowTriadArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.ShowTriad(req.Triad); err != nil {
		return nil, err
	}
	return ok()
}

func hideTriad(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	s.HideTriad()
	return ok()
}

func getTriad(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(s.TriadSpec())
}

func setManipulators(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetManipulatorsArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.SetManipulators(req.ID, req.Handles, req.Command); err != nil {
		return nil, err
	}
	return ok()
}

func removeManipulators(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.RemoveManipulatorsArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.RemoveManipulators(req.ID); err != nil {
		return nil, err
	}
	return ok()
}
