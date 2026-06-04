// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// enumerateEntities3D lists a 3D sketch's geometry (kind, construction, points, radius).
func enumerateEntities3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	ents := sk.Entities()
	out := make([]wire.Sketch3DEntityInfo, 0, len(ents))
	for i, e := range ents {
		out = append(out, entity3DInfo(i, e))
	}
	return json.Marshal(wire.EnumerateEntities3DResult{Entities: out})
}

// enumerateConstraints3D lists a 3D sketch's geometric constraints.
func enumerateConstraints3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	cons := sk.GeometricConstraints3D().All()
	out := make([]wire.Constraint3DInfo, 0, len(cons))
	for i, c := range cons {
		kind, ids := constraint3DKind(c)
		out = append(out, wire.Constraint3DInfo{Index: i, Kind: string(kind), Entities: ids})
	}
	return json.Marshal(wire.ListConstraints3DResult{Constraints: out})
}

// enumerateDimensions3D lists a 3D sketch's dimensional constraints.
func enumerateDimensions3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	dims := sk.DimensionConstraints3D().All()
	out := make([]wire.Dimension3DInfo, 0, len(dims))
	for i, d := range dims {
		out = append(out, wire.Dimension3DInfo{
			Index:      i,
			Kind:       d.KindName(),
			Name:       d.Parameter().Name(),
			Expression: d.Parameter().Expression(),
			Value:      d.Measured(),
			Driven:     d.Driven(),
		})
	}
	return json.Marshal(wire.ListDimensions3DResult{Dimensions: out})
}

// entity3DInfo renders one 3D entity as its wire summary. Curve kinds are added by F02+;
// the spine renders standalone points (the only entity kind in F01).
func entity3DInfo(index int, e sketch.Entity) wire.Sketch3DEntityInfo {
	info := wire.Sketch3DEntityInfo{Index: index, ID: uint64(e.EntityID()), Kind: string(types.Sketch3DEntityUnknown)}
	if p, ok := e.(*sketch.Point3D); ok {
		pos := p.Position()
		info.Kind = string(types.Sketch3DEntityPoint)
		info.Points = [][]float64{{float64(pos.X), float64(pos.Y), float64(pos.Z)}}
	}
	return info
}

// constraint3DKind maps a 3D constraint to its wire kind and the session ids it relates.
func constraint3DKind(c sketch.Constraint) (types.Geometric3DConstraintKind, []uint64) {
	switch v := c.(type) {
	case *sketch.Coincident3D:
		return types.Geo3DCoincident, []uint64{uint64(v.A.EntityID()), uint64(v.B.EntityID())}
	case *sketch.Collinear3D:
		return types.Geo3DCollinear, []uint64{uint64(v.A.EntityID()), uint64(v.B.EntityID()), uint64(v.C.EntityID())}
	case *sketch.Concentric3D:
		return types.Geo3DConcentric, []uint64{uint64(v.Center1.EntityID()), uint64(v.Center2.EntityID())}
	case *sketch.Equal3D:
		return types.Geo3DEqual, nil
	default:
		return types.Geo3DUnknown, nil
	}
}
