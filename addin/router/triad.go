// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerTriadHandlers wires the gizmo methods (M05-F13, #620).
func (r *Router) registerTriadHandlers() {
	r.readOnly(wire.MethodTriadShow, typed(showTriad))
	r.readOnly(wire.MethodTriadUpdate, typed(updateTriad))
	r.readOnly(wire.MethodTriadHide, hideTriad)
	r.readOnly(wire.MethodTriadGet, getTriad)
	r.readOnly(wire.MethodManipulatorsSet, typed(setManipulators))
	r.readOnly(wire.MethodManipulatorsRemove, typed(removeManipulators))
}

func showTriad(s *app.Session, in wire.ShowTriadArgs) (wire.OKResult, error) {
	in.Triad.Visible = true
	if err := s.ShowTriad(in.Triad); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func updateTriad(s *app.Session, in wire.ShowTriadArgs) (wire.OKResult, error) {
	if err := s.ShowTriad(in.Triad); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func hideTriad(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	s.HideTriad()
	return ok()
}

func getTriad(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(s.TriadSpec())
}

func setManipulators(s *app.Session, in wire.SetManipulatorsArgs) (wire.OKResult, error) {
	if err := s.SetManipulators(in.ID, in.Handles, in.Command); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func removeManipulators(s *app.Session, in wire.RemoveManipulatorsArgs) (wire.OKResult, error) {
	if err := s.RemoveManipulators(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}
