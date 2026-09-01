// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
)

// Body point/ray/validity queries over the wire (M07-F07, #630).

// bodyLocateUsingPoint serves wire.MethodBodyLocateUsingPoint.
func bodyLocateUsingPoint(_ *app.Session, part *compdef.PartComponentDefinition, in wire.LocateUsingPointArgs) (wire.LocateUsingPointResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.LocateUsingPointResult{}, err
	}
	p, err := xyzPoint(in.Point, "point")
	if err != nil {
		return wire.LocateUsingPointResult{}, err
	}
	kind, err := entityKindFilter(in.EntityKind)
	if err != nil {
		return wire.LocateUsingPointResult{}, err
	}
	hit, found := query.LocateUsingPoint(b, kind, p, in.ProximityTolerance, ops.DefaultQuality())
	out := wire.LocateUsingPointResult{Found: found}
	if found {
		out.Entity = locatedInfo(hit)
	}
	return out, nil
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

func locatedInfo(hit query.LocatedEntity) wire.LocatedEntityInfo {
	key, tkey := locatedKeys(hit)
	return wire.LocatedEntityInfo{
		Kind: hit.Kind.String(), Key: key, TransientKey: tkey,
		Point:    []float64{float64(hit.Point.X), float64(hit.Point.Y), float64(hit.Point.Z)},
		Distance: hit.Distance,
	}
}

func locatedKeys(hit query.LocatedEntity) (string, uint64) {
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
func bodyFindUsingRay(_ *app.Session, part *compdef.PartComponentDefinition, in wire.FindUsingRayArgs) (wire.FindUsingRayResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.FindUsingRayResult{}, err
	}
	origin, err := xyzPoint(in.Origin, "origin")
	if err != nil {
		return wire.FindUsingRayResult{}, err
	}
	dir, err := xyz(in.Direction, "direction")
	if err != nil {
		return wire.FindUsingRayResult{}, err
	}
	var out wire.FindUsingRayResult
	for _, hit := range query.FindUsingRay(b, origin, dir, in.Radius, ops.DefaultQuality(), in.FindFirstOnly) {
		out.Hits = append(out.Hits, locatedInfo(hit))
	}
	return out, nil
}

// bodyIsPointInside serves wire.MethodBodyIsPointInside.
func bodyIsPointInside(_ *app.Session, part *compdef.PartComponentDefinition, in wire.IsPointInsideArgs) (wire.IsPointInsideResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.IsPointInsideResult{}, err
	}
	p, err := xyzPoint(in.Point, "point")
	if err != nil {
		return wire.IsPointInsideResult{}, err
	}
	onTol := in.OnTolerance
	if onTol <= 0 {
		onTol = 1e-6
	}
	verdict, err := pointContainmentOf(b, in.ShellIndex, p, onTol)
	if err != nil {
		return wire.IsPointInsideResult{}, err
	}
	return wire.IsPointInsideResult{Containment: containmentSpelling(verdict)}, nil
}

func pointContainmentOf(b *topo.Body, shellIndex *int, p math.Point3, onTol float64) (query.PointContainment, error) {
	q := ops.DefaultQuality()
	if shellIndex == nil {
		return query.BodyContainment(b, p, q, onTol), nil
	}
	shells := b.Shells()
	if *shellIndex < 0 || *shellIndex >= len(shells) {
		return 0, fmt.Errorf("shell index %d out of range (body has %d shells)", *shellIndex, len(shells))
	}
	return query.ShellContainment(shells[*shellIndex], p, q, onTol), nil
}

func containmentSpelling(c query.PointContainment) string {
	switch c {
	case query.ContainInside:
		return types.InsideContainment.String()
	case query.ContainOn:
		return types.OnContainment.String()
	default:
		return types.OutsideContainment.String()
	}
}

// bodyMinimumDistance serves wire.MethodBodyMinimumDistance: the closest approach between the body
// and a transient probe polyline (a CAM travel path), optionally widened by the tool radius. The
// out-of-process projection of MeasureTools.GetMinimumDistance for a transient operand (#630).
func bodyMinimumDistance(_ *app.Session, part *compdef.PartComponentDefinition, in wire.MinimumDistanceArgs) (wire.MinimumDistanceResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.MinimumDistanceResult{}, err
	}
	probe, err := probePolyline(in.Points)
	if err != nil {
		return wire.MinimumDistanceResult{}, err
	}
	dist := analysis.MinDistanceProbeToBody(probe, b, ops.DefaultQuality()) - in.Radius
	if dist < 0 {
		dist = 0
	}
	return wire.MinimumDistanceResult{Distance: dist}, nil
}

// probePolyline decodes a flat [x,y,z, ...] list (database units) into points; it requires a
// non-empty multiple of three so the offending shape is reported rather than silently truncated.
func probePolyline(flat []float64) ([]math.Point3, error) {
	if len(flat) == 0 || len(flat)%3 != 0 {
		return nil, fmt.Errorf("probe points = %d floats, want a non-empty multiple of 3 (x,y,z per point)", len(flat))
	}
	pts := make([]math.Point3, 0, len(flat)/3)
	for i := 0; i < len(flat); i += 3 {
		pts = append(pts, math.V3(math.Scalar(flat[i]), math.Scalar(flat[i+1]), math.Scalar(flat[i+2])).AsPoint())
	}
	return pts, nil
}

// bodyConvexityEdges serves wire.MethodBodyConvexityEdges.
func bodyConvexityEdges(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ConvexityEdgesArgs) (wire.ConvexityEdgesResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.ConvexityEdgesResult{}, err
	}
	class, err := convexityClass(in.Collection)
	if err != nil {
		return wire.ConvexityEdgesResult{}, err
	}
	var out wire.ConvexityEdgesResult
	for _, e := range blend.BodyEdgeConvexity(b)[class] {
		out.Edges = append(out.Edges, wire.TopologyRef{
			Key:   string(e.ReferenceKey()),
			Point: topoRefPoint(e.RangeBox().Center()),
		})
	}
	return out, nil
}

func convexityClass(spelling string) (blend.EdgeConvexity, error) {
	kind, ok := types.ParseEdgeCollectionKind(spelling)
	if !ok {
		return 0, fmt.Errorf("unknown edge collection %q (want allConvex, allConcave or tangentiallyConnected)", spelling)
	}
	switch kind {
	case types.AllConvexEdges:
		return blend.EdgeConvex, nil
	case types.AllConcaveEdges:
		return blend.EdgeConcave, nil
	case types.TangentiallyConnectedEdges:
		return blend.EdgeTangent, nil
	default:
		return blend.EdgeConvexityUnknown, nil
	}
}

// bodyValidate serves wire.MethodBodyValidate.
func bodyValidate(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ValidateBodyArgs) (wire.ValidateBodyResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.ValidateBodyResult{}, err
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
	return out, nil
}

// bodyRangeBox serves wire.MethodBodyRangeBox.
func bodyRangeBox(_ *app.Session, part *compdef.PartComponentDefinition, in wire.BodyRangeBoxArgs) (wire.BodyRangeBoxResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BodyRangeBoxResult{}, err
	}
	if in.Oriented {
		return orientedRangeBoxReply(b)
	}
	box := b.RangeBox()
	if in.Precise {
		box = query.PreciseRangeBox(b, ops.DefaultQuality())
	}
	return wire.BodyRangeBoxResult{
		Min: []float64{float64(box.Min.X), float64(box.Min.Y), float64(box.Min.Z)},
		Max: []float64{float64(box.Max.X), float64(box.Max.Y), float64(box.Max.Z)},
	}, nil
}

func orientedRangeBoxReply(b *topo.Body) (wire.BodyRangeBoxResult, error) {
	obb, err := query.OrientedMinimumRangeBox(b)
	if err != nil {
		return wire.BodyRangeBoxResult{}, err
	}
	vec := func(v math.Vector3) []float64 { return []float64{float64(v.X), float64(v.Y), float64(v.Z)} }
	corner, edges := obb.MinCorner(), obb.EdgeVectors()
	return wire.BodyRangeBoxResult{
		Corner:       []float64{float64(corner.X), float64(corner.Y), float64(corner.Z)},
		DirectionOne: vec(edges[0]), DirectionTwo: vec(edges[1]), DirectionThree: vec(edges[2]),
	}, nil
}

// bodyBindTransientKey serves wire.MethodBodyBindTransientKey.
func bodyBindTransientKey(_ *app.Session, part *compdef.PartComponentDefinition, in wire.BindTransientKeyArgs) (wire.BindTransientKeyResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.BindTransientKeyResult{}, err
	}
	ref, ok := b.BindTransientKey(in.TransientKey)
	if !ok {
		return wire.BindTransientKeyResult{Found: false}, nil
	}
	return wire.BindTransientKeyResult{
		Found: true, Kind: ref.Kind.String(), Key: string(transientRefReferenceKey(ref)),
	}, nil
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
