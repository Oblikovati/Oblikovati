// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/clientgraphics"
)

// registerGraphicsObjectModelHandlers wires the retained-mode node mutations and the named
// color-mapper registry of the client-graphics object model (M16-F05, #641).
func (r *Router) registerGraphicsObjectModelHandlers() {
	r.readOnly(wire.MethodGraphicsNodeSetTransform, setGraphicsNodeTransform)
	r.readOnly(wire.MethodGraphicsNodeSetVisible, setGraphicsNodeVisible)
	r.readOnly(wire.MethodGraphicsNodeSetSelectable, setGraphicsNodeSelectable)
	r.readOnly(wire.MethodClientGraphicsRegisterMapper, registerColorMapper)
	r.readOnly(wire.MethodClientGraphicsListMappers, listColorMappers)
}

// setGraphicsNodeTransform moves one node without resubmitting its mesh
// (wire.MethodGraphicsNodeSetTransform).
func setGraphicsNodeTransform(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetNodeTransformArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	xf, has, err := clientgraphics.TransformFromWire(a.Transform)
	if err != nil {
		return nil, err
	}
	if err := s.Graphics().SetNodeTransform(a.ClientId, a.NodeId, xf, has); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// setGraphicsNodeVisible toggles one node's visibility (wire.MethodGraphicsNodeSetVisible).
func setGraphicsNodeVisible(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetNodeVisibleArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.Graphics().SetNodeVisible(a.ClientId, a.NodeId, a.Visible); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// setGraphicsNodeSelectable toggles one node's pickability (wire.MethodGraphicsNodeSetSelectable).
func setGraphicsNodeSelectable(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetNodeSelectableArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.Graphics().SetNodeSelectable(a.ClientId, a.NodeId, a.Selectable); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// registerColorMapper stores a named, reusable color mapper
// (wire.MethodClientGraphicsRegisterMapper).
func registerColorMapper(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.RegisterColorMapperArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	m, err := clientgraphics.MapperFromWire(a.Mapper)
	if err != nil {
		return nil, err
	}
	if err := s.Graphics().RegisterMapper(a.Name, m); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
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
