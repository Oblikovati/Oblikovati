// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati/api/wire"
	"oblikovati/app"
	"oblikovati/model/clientgraphics"
)

// setClientGraphics submits or replaces a named client-graphics group (idempotent by
// clientId) and echoes its node/primitive counts (wire.MethodClientGraphicsSet).
func setClientGraphics(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetClientGraphicsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	g, err := clientgraphics.DecodeGroup(a)
	if err != nil {
		return nil, err
	}
	s.Graphics().Set(g)
	return json.Marshal(wire.SetClientGraphicsResult{
		ClientId: g.Name(), NodeCount: g.NodeCount(), PrimitiveCount: g.PrimitiveCount(),
	})
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
func deleteClientGraphics(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.DeleteClientGraphicsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	s.Graphics().Delete(a.ClientId)
	return ok()
}

// setClientGraphicsVisible toggles a group's visibility without resubmitting its geometry
// (wire.MethodClientGraphicsSetVisible).
func setClientGraphicsVisible(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetClientGraphicsVisibleArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.Graphics().SetVisible(a.ClientId, a.Visible); err != nil {
		return nil, err
	}
	return ok()
}

// updateInteractionGraphics replaces a transient interaction lane's nodes — the
// rubber-band/manipulator update path (wire.MethodInteractionGraphicsUpdate).
func updateInteractionGraphics(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.UpdateInteractionGraphicsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	g, err := clientgraphics.DecodeGroup(wire.SetClientGraphicsArgs{ClientId: interactionClientID(a.Lane), Lane: a.Lane, Nodes: a.Nodes})
	if err != nil {
		return nil, err
	}
	s.Graphics().ReplaceLane(clientgraphics.Lane(a.Lane), []clientgraphics.Group{g})
	return ok()
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
