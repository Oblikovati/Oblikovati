// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
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

// entity3DInfo renders one 3D entity as its wire summary (kind, defining points in
// model cm, radius for circular kinds, construction flag).
func entity3DInfo(index int, e sketch.Entity) wire.Sketch3DEntityInfo {
	info := wire.Sketch3DEntityInfo{Index: index, ID: uint64(e.EntityID()), Kind: string(types.Sketch3DEntityUnknown)}
	if p, ok := e.(*sketch.Point3D); ok {
		info.Kind = string(types.Sketch3DEntityPoint)
		info.Points = [][]float64{p3coords(p.Position())}
		return info
	}
	if fillSegmentCurve3DInfo(&info, e) {
		return info
	}
	fillRoundCurve3DInfo(&info, e)
	return info
}

// fillSegmentCurve3DInfo renders the straight/poly curve families (line/arc) and the
// plain circle into info, reporting whether it matched.
func fillSegmentCurve3DInfo(info *wire.Sketch3DEntityInfo, e sketch.Entity) bool {
	switch v := e.(type) {
	case *sketch.Line3D:
		info.Kind = string(types.Sketch3DEntityLine)
		info.Points = [][]float64{p3coords(v.A.Position()), p3coords(v.B.Position())}
		info.Construction = v.IsConstruction()
	case *sketch.Circle3D:
		info.Kind = string(types.Sketch3DEntityCircle)
		info.Points = [][]float64{p3coords(v.Center.Position())}
		info.Radius = float64(v.Radius)
		info.Construction = v.IsConstruction()
	case *sketch.Arc3D:
		info.Kind = string(types.Sketch3DEntityArc)
		info.Points = [][]float64{p3coords(v.Center.Position()), p3coords(v.Start.Position()), p3coords(v.End.Position())}
		info.Radius = float64(v.Radius())
		info.Construction = v.IsConstruction()
	default:
		return false
	}
	return true
}

// fillRoundCurve3DInfo renders the helix and conic families (centered, radius-bearing).
func fillRoundCurve3DInfo(info *wire.Sketch3DEntityInfo, e sketch.Entity) {
	switch v := e.(type) {
	case *sketch.HelicalCurve3D:
		info.Kind = string(types.Sketch3DEntityHelical)
		info.Points = [][]float64{p3coords(v.Origin.Position())}
		info.Radius = float64(v.StartRadius)
		info.Construction = v.IsConstruction()
	case *sketch.Ellipse3D:
		info.Kind = string(types.Sketch3DEntityEllipse)
		info.Points = [][]float64{p3coords(v.Center.Position())}
		info.Radius = float64(v.MajorRadius)
		info.Construction = v.IsConstruction()
	case *sketch.EllipticalArc3D:
		info.Kind = string(types.Sketch3DEntityEllipticalArc)
		info.Points = [][]float64{p3coords(v.Center.Position())}
		info.Radius = float64(v.MajorRadius)
		info.Construction = v.IsConstruction()
	}
}

// p3coords flattens a model point to [x,y,z].
func p3coords(p math.Point3) []float64 {
	return []float64{float64(p.X), float64(p.Y), float64(p.Z)}
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
