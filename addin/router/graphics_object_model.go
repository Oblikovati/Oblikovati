// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/clientgraphics"
)

// registerGraphicsObjectModelHandlers wires the retained-mode node mutations and the named
// color-mapper registry of the client-graphics object model (M16-F05, #641).
func (r *Router) registerGraphicsObjectModelHandlers() {
	r.readOnly(wire.MethodGraphicsNodeSetTransform, typed(setGraphicsNodeTransform))
	r.readOnly(wire.MethodGraphicsNodeSetVisible, typed(setGraphicsNodeVisible))
	r.readOnly(wire.MethodGraphicsNodeSetSelectable, typed(setGraphicsNodeSelectable))
	r.readOnly(wire.MethodClientGraphicsRegisterMapper, typed(registerColorMapper))
	r.readOnly(wire.MethodClientGraphicsListMappers, listColorMappers)
}

// setGraphicsNodeTransform moves one node without resubmitting its mesh
// (wire.MethodGraphicsNodeSetTransform).
func setGraphicsNodeTransform(s *app.Session, in wire.SetNodeTransformArgs) (wire.OKResult, error) {
	xf, has, err := clientgraphics.TransformFromWire(in.Transform)
	if err != nil {
		return wire.OKResult{}, err
	}
	if err := s.Graphics().SetNodeTransform(in.ClientId, in.NodeId, xf, has); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setGraphicsNodeVisible toggles one node's visibility (wire.MethodGraphicsNodeSetVisible).
func setGraphicsNodeVisible(s *app.Session, in wire.SetNodeVisibleArgs) (wire.OKResult, error) {
	if err := s.Graphics().SetNodeVisible(in.ClientId, in.NodeId, in.Visible); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setGraphicsNodeSelectable toggles one node's pickability (wire.MethodGraphicsNodeSetSelectable).
func setGraphicsNodeSelectable(s *app.Session, in wire.SetNodeSelectableArgs) (wire.OKResult, error) {
	if err := s.Graphics().SetNodeSelectable(in.ClientId, in.NodeId, in.Selectable); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// registerColorMapper stores a named, reusable color mapper
// (wire.MethodClientGraphicsRegisterMapper).
func registerColorMapper(s *app.Session, in wire.RegisterColorMapperArgs) (wire.OKResult, error) {
	m, err := clientgraphics.MapperFromWire(in.Mapper)
	if err != nil {
		return wire.OKResult{}, err
	}
	if err := s.Graphics().RegisterMapper(in.Name, m); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// listColorMappers enumerates the registered named color mappers
// (wire.MethodClientGraphicsListMappers).
func listColorMappers(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	mappers := s.Graphics().Mappers()
	out := make([]wire.ColorMapperInfo, len(mappers))
	for i, m := range mappers {
		out[i] = wire.ColorMapperInfo{Name: m.Name, StopCount: m.StopCount}
	}
	return json.Marshal(wire.ColorMappersResult{Mappers: out})
}
