// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/clientgraphics"
)

// setClientGraphics submits or replaces a named client-graphics group (idempotent by
// clientId) and echoes its node/primitive counts (wire.MethodClientGraphicsSet).
func setClientGraphics(s *app.Session, in wire.SetClientGraphicsArgs) (wire.SetClientGraphicsResult, error) {
	g, err := clientgraphics.DecodeGroup(in)
	if err != nil {
		return wire.SetClientGraphicsResult{}, err
	}
	s.Graphics().Set(g)
	return wire.SetClientGraphicsResult{
		ClientId: g.Name(), NodeCount: g.NodeCount(), PrimitiveCount: g.PrimitiveCount(),
	}, nil
}

// listClientGraphics enumerates the live graphics groups across all lanes, sorted by
// client id (wire.MethodClientGraphicsList).
func listClientGraphics(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	groups := s.Graphics().Groups()
	out := make([]wire.ClientGraphicsInfo, len(groups))
	for i, g := range groups {
		out[i] = wire.ClientGraphicsInfo{
			ClientId: g.Name(), Lane: g.Lane(), Visible: g.Visible(),
			NodeCount: g.NodeCount(), PrimitiveCount: g.PrimitiveCount(),
		}
	}
	return json.Marshal(wire.ListClientGraphicsResult{Groups: out})
}

// deleteClientGraphics removes a named group (wire.MethodClientGraphicsDelete).
func deleteClientGraphics(s *app.Session, in wire.DeleteClientGraphicsArgs) (wire.OKResult, error) {
	s.Graphics().Delete(in.ClientId)
	return wire.OKResult{OK: true}, nil
}

// setClientGraphicsVisible toggles a group's visibility without resubmitting its geometry
// (wire.MethodClientGraphicsSetVisible).
func setClientGraphicsVisible(s *app.Session, in wire.SetClientGraphicsVisibleArgs) (wire.OKResult, error) {
	if err := s.Graphics().SetVisible(in.ClientId, in.Visible); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// updateInteractionGraphics replaces a transient interaction lane's nodes — the
// rubber-band/manipulator update path (wire.MethodInteractionGraphicsUpdate).
func updateInteractionGraphics(s *app.Session, in wire.UpdateInteractionGraphicsArgs) (wire.OKResult, error) {
	g, err := clientgraphics.DecodeGroup(wire.SetClientGraphicsArgs{ClientId: interactionClientID(in.Lane), Lane: in.Lane, Nodes: in.Nodes})
	if err != nil {
		return wire.OKResult{}, err
	}
	s.Graphics().ReplaceLane(clientgraphics.Lane(in.Lane), []clientgraphics.Group{g})
	return wire.OKResult{OK: true}, nil
}

// clearInteractionGraphics drops every transient interaction lane — what a command calls
// on commit/cancel (wire.MethodInteractionGraphicsClear).
func clearInteractionGraphics(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	s.Graphics().ClearInteraction()
	return ok()
}

// interactionClientID is the reserved group key for one interaction lane, so an Update
// replaces (rather than accumulates) the lane's content each call.
func interactionClientID(lane string) string { return "__interaction__/" + lane }
