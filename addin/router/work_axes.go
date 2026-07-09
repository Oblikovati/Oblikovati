// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// The work-axis wire surface (workAxes.create / workAxes.list), mirroring work_planes.go: a
// datum axis is a grounded "line" (origin + direction), an axis through two points, or the axis
// where two planes meet. An unsatisfiable-but-well-formed definition still creates the axis
// (reported healthy=false); the call only fails on bad arguments.

// listWorkAxes enumerates the active model's datum axes (origin frame first, then user axes)
// with their current geometry and health.
func listWorkAxes(_ *app.Session, host workHost) (wire.ListWorkAxesResult, error) {
	axes := projectLiveDatums(host.WorkAxes(), workAxisInfo)
	return wire.ListWorkAxesResult{Axes: axes}, nil
}

// workAxisInfo renders one axis as the wire DTO: its identity, current origin + unit direction,
// whether it is one of the origin coordinate axes, health, and constructor kind.
func workAxisInfo(index int, wa *feature.WorkAxis) wire.WorkAxisInfo {
	return wire.WorkAxisInfo{
		Index:        index,
		Name:         wa.Name(),
		Ref:          string(wa.Key()),
		Kind:         wa.Kind(),
		Origin:       point3Slice(wa.Origin()),
		Direction:    vector3Slice(wa.Direction().AsVector()),
		IsOrigin:     wa.IsCoordinateSystemElement(),
		Construction: wa.Construction(),
		Healthy:      wa.Health().OK(),
		Reason:       wa.Health().Reason, // empty when healthy
	}
}

// listWorkPoints enumerates the active model's datum points (origin centre first, then user
// points) with their position, kind, origin flag, visibility, and health (#1842).
func listWorkPoints(_ *app.Session, host workHost) (wire.ListWorkPointsResult, error) {
	points := projectLiveDatums(host.WorkPoints(), workPointInfo)
	return wire.ListWorkPointsResult{Points: points}, nil
}

// workPointInfo renders one datum point as the wire DTO.
func workPointInfo(index int, wp *feature.WorkPoint) wire.WorkPointInfo {
	return wire.WorkPointInfo{
		Index:        index,
		Name:         wp.Name(),
		Ref:          string(wp.Key()),
		Position:     point3Slice(wp.Point()),
		IsOrigin:     wp.IsCoordinateSystemElement(),
		Visible:      wp.Visible(),
		Construction: wp.Construction(),
		Healthy:      wp.Health().OK(),
		Reason:       wp.Health().Reason, // empty when healthy
		Kind:         wp.Kind(),
	}
}

// axisRefCtor builds a reference-model axis from exactly its references; axisRefCtors is the table
// of the kinds dispatched purely by reference count, keeping buildWorkAxis's body small (the
// grounded "line" kind, which reads scalar origin/direction, stays a special case). Mirrors
// refPlaneCtors.
type axisRefCtor struct {
	arity int
	build func(*feature.WorkAxes, []feature.WorkRef) *feature.WorkAxis
}

var axisRefCtors = map[types.WorkAxisKind]axisRefCtor{
	types.WorkAxisTwoPoints: {2, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis { return a.AddByTwoPoints(r[0], r[1]) }},
	types.WorkAxisPlaneIntersection: {2, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis {
		return a.AddByPlaneIntersection(r[0], r[1])
	}},
	types.WorkAxisPointAndPlane: {2, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis {
		return a.AddByPointAndPlane(r[0], r[1])
	}},
	types.WorkAxisLineAndPoint: {2, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis {
		return a.AddByLineAndPoint(r[0], r[1])
	}},
	types.WorkAxisLineAndPlane: {2, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis {
		return a.AddByLineAndPlane(r[0], r[1])
	}},
	types.WorkAxisRevolvedFace: {1, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis { return a.AddByRevolvedFace(r[0]) }},
	types.WorkAxisAnalyticEdge: {1, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis { return a.AddByAnalyticEdge(r[0]) }},
	types.WorkAxisLineByEntity: {1, func(a *feature.WorkAxes, r []feature.WorkRef) *feature.WorkAxis { return a.AddByLineByEntity(r[0]) }},
}

// buildWorkAxis dispatches a create request to the matching model constructor.
func buildWorkAxis(host workHost, in wire.CreateWorkAxisArgs) (*feature.WorkAxis, error) {
	axes := host.WorkAxes()
	kind := types.WorkAxisKind(in.Kind)
	if kind == types.WorkAxisLine {
		return addLineAxis(axes, in)
	}
	c, ok := axisRefCtors[kind]
	if !ok {
		return nil, fmt.Errorf("workAxes.create: unknown kind %q (see api/types WorkAxis*)", in.Kind)
	}
	refs := toWorkRefs(in.Refs)
	if len(refs) != c.arity {
		return nil, fmt.Errorf("workAxes.create: %s needs %d references, got %d", in.Kind, c.arity, len(refs))
	}
	return c.build(axes, refs), nil
}

// addLineAxis builds the grounded "line" axis from its origin point and direction vector.
func addLineAxis(axes *feature.WorkAxes, in wire.CreateWorkAxisArgs) (*feature.WorkAxis, error) {
	origin, err := parseCoords(in.Origin, "workAxes.create: origin")
	if err != nil {
		return nil, err
	}
	dir, err := parseAxisVector(in.Direction, "workAxes.create: direction")
	if err != nil {
		return nil, err
	}
	return axes.AddByLine(math.P3(origin[0], origin[1], origin[2]), dir), nil
}
