// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Body point/ray/validity queries over the wire (M07-F07, #630).

// bodyLocateUsingPoint serves wire.MethodBodyLocateUsingPoint.
func bodyLocateUsingPoint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.LocateUsingPointArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	p, err := xyzPoint(in.Point, "point")
	if err != nil {
		return nil, err
	}
	kind, err := entityKindFilter(in.EntityKind)
	if err != nil {
		return nil, err
	}
	hit, found := ops.LocateUsingPoint(b, kind, p, in.ProximityTolerance, ops.DefaultQuality())
	out := wire.LocateUsingPointResult{Found: found}
	if found {
		out.Entity = locatedInfo(hit)
	}
	return json.Marshal(out)
}

// entityKindFilter maps the wire spelling (empty = any).
func entityKindFilter(spelling string) (topo.EntityKind, error) {
	switch spelling {
	case "":
		return 0, nil
	case "vertex":
		return topo.KindVertex, nil
	case "edge":
		return topo.KindEdge, nil
	case "face":
		return topo.KindFace, nil
	default:
		return 0, fmt.Errorf("unknown entity kind %q (want vertex, edge or face)", spelling)
	}
}

func locatedInfo(hit ops.LocatedEntity) wire.LocatedEntityInfo {
	key, tkey := locatedKeys(hit)
	return wire.LocatedEntityInfo{
		Kind: hit.Kind.String(), Key: key, TransientKey: tkey,
		Point:    []float64{float64(hit.Point.X), float64(hit.Point.Y), float64(hit.Point.Z)},
		Distance: hit.Distance,
	}
}

func locatedKeys(hit ops.LocatedEntity) (string, uint64) {
	switch hit.Kind {
	case topo.KindVertex:
		return string(hit.Vertex.ReferenceKey()), hit.Vertex.ID()
	case topo.KindEdge:
		return string(hit.Edge.ReferenceKey()), hit.Edge.ID()
	default:
		return string(hit.Face.ReferenceKey()), hit.Face.ID()
	}
}

// bodyFindUsingRay serves wire.MethodBodyFindUsingRay.
func bodyFindUsingRay(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.FindUsingRayArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	origin, err := xyzPoint(in.Origin, "origin")
	if err != nil {
		return nil, err
	}
	dir, err := xyz(in.Direction, "direction")
	if err != nil {
		return nil, err
	}
	var out wire.FindUsingRayResult
	for _, hit := range ops.FindUsingRay(b, origin, dir, in.Radius, ops.DefaultQuality(), in.FindFirstOnly) {
		out.Hits = append(out.Hits, locatedInfo(hit))
	}
	return json.Marshal(out)
}

// bodyIsPointInside serves wire.MethodBodyIsPointInside.
func bodyIsPointInside(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.IsPointInsideArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	p, err := xyzPoint(in.Point, "point")
	if err != nil {
		return nil, err
	}
	onTol := in.OnTolerance
	if onTol <= 0 {
		onTol = 1e-6
	}
	verdict, err := pointContainmentOf(b, in.ShellIndex, p, onTol)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.IsPointInsideResult{Containment: containmentSpelling(verdict)})
}

func pointContainmentOf(b *topo.Body, shellIndex *int, p math.Point3, onTol float64) (ops.PointContainment, error) {
	q := ops.DefaultQuality()
	if shellIndex == nil {
		return ops.BodyContainment(b, p, q, onTol), nil
	}
	shells := b.Shells()
	if *shellIndex < 0 || *shellIndex >= len(shells) {
		return 0, fmt.Errorf("shell index %d out of range (body has %d shells)", *shellIndex, len(shells))
	}
	return ops.ShellContainment(shells[*shellIndex], p, q, onTol), nil
}

func containmentSpelling(c ops.PointContainment) string {
	switch c {
	case ops.ContainInside:
		return types.InsideContainment.String()
	case ops.ContainOn:
		return types.OnContainment.String()
	default:
		return types.OutsideContainment.String()
	}
}

// bodyConvexityEdges serves wire.MethodBodyConvexityEdges.
func bodyConvexityEdges(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ConvexityEdgesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	class, err := convexityClass(in.Collection)
	if err != nil {
		return nil, err
	}
	var out wire.ConvexityEdgesResult
	for _, e := range ops.BodyEdgeConvexity(b)[class] {
		out.Edges = append(out.Edges, wire.TopologyRef{
			Key:   string(e.ReferenceKey()),
			Point: topoRefPoint(e.RangeBox().Center()),
		})
	}
	return json.Marshal(out)
}

func convexityClass(spelling string) (ops.EdgeConvexity, error) {
	kind, ok := types.ParseEdgeCollectionKind(spelling)
	if !ok {
		return 0, fmt.Errorf("unknown edge collection %q (want allConvex, allConcave or tangentiallyConnected)", spelling)
	}
	switch kind {
	case types.AllConvexEdges:
		return ops.EdgeConvex, nil
	case types.AllConcaveEdges:
		return ops.EdgeConcave, nil
	case types.TangentiallyConnectedEdges:
		return ops.EdgeTangent, nil
	default:
		return ops.EdgeConvexityUnknown, nil
	}
}

// bodyValidate serves wire.MethodBodyValidate.
func bodyValidate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ValidateBodyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	level := ops.CheckTopology
	if in.CheckLevel >= int(ops.CheckGeometry) {
		level = ops.CheckGeometry
	}
	valid, problems := ops.ValidateBodyEntities(b, level, ops.DefaultQuality())
	out := wire.ValidateBodyResult{Valid: valid}
	for _, p := range problems {
		out.Problems = append(out.Problems, wire.ProblemEntityInfo{
			Kind: p.Kind.String(), Key: string(p.ReferenceKey), TransientKey: p.ID, Issue: p.Issue,
		})
	}
	return json.Marshal(out)
}

// bodyRangeBox serves wire.MethodBodyRangeBox.
func bodyRangeBox(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BodyRangeBoxArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	if in.Oriented {
		return orientedRangeBoxReply(b)
	}
	box := b.RangeBox()
	if in.Precise {
		box = ops.PreciseRangeBox(b, ops.DefaultQuality())
	}
	return json.Marshal(wire.BodyRangeBoxResult{
		Min: []float64{float64(box.Min.X), float64(box.Min.Y), float64(box.Min.Z)},
		Max: []float64{float64(box.Max.X), float64(box.Max.Y), float64(box.Max.Z)},
	})
}

func orientedRangeBoxReply(b *topo.Body) (json.RawMessage, error) {
	obb, err := ops.OrientedMinimumRangeBox(b)
	if err != nil {
		return nil, err
	}
	vec := func(v math.Vector3) []float64 { return []float64{float64(v.X), float64(v.Y), float64(v.Z)} }
	return json.Marshal(wire.BodyRangeBoxResult{
		Corner:       []float64{float64(obb.Corner.X), float64(obb.Corner.Y), float64(obb.Corner.Z)},
		DirectionOne: vec(obb.DirectionOne), DirectionTwo: vec(obb.DirectionTwo), DirectionThree: vec(obb.DirectionThree),
	})
}

// bodyBindTransientKey serves wire.MethodBodyBindTransientKey.
func bodyBindTransientKey(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.BindTransientKeyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	b, err := resolveBody(s, in.BodyIndex)
	if err != nil {
		return nil, err
	}
	ref, ok := b.BindTransientKey(in.TransientKey)
	if !ok {
		return json.Marshal(wire.BindTransientKeyResult{Found: false})
	}
	return json.Marshal(wire.BindTransientKeyResult{
		Found: true, Kind: ref.Kind.String(), Key: string(transientRefReferenceKey(ref)),
	})
}

func transientRefReferenceKey(ref topo.TransientRef) []byte {
	switch ref.Kind {
	case topo.KindVertex:
		return ref.Vertex.ReferenceKey()
	case topo.KindEdge:
		return ref.Edge.ReferenceKey()
	case topo.KindFace:
		return ref.Face.ReferenceKey()
	case topo.KindShell:
		return ref.Shell.ReferenceKey()
	default:
		return ref.Wire.ReferenceKey()
	}
}
