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
	axes := projectAll(host.WorkAxes(), workAxisInfo)
	return wire.ListWorkAxesResult{Axes: axes}, nil
}

// workAxisInfo renders one axis as the wire DTO: its identity, current origin + unit direction,
// whether it is one of the origin coordinate axes, health, and constructor kind.
func workAxisInfo(index int, wa *feature.WorkAxis) wire.WorkAxisInfo {
	return wire.WorkAxisInfo{
		Index:     index,
		Name:      wa.Name(),
		Ref:       string(wa.Key()),
		Kind:      wa.Kind(),
		Origin:    point3Slice(wa.Origin()),
		Direction: vector3Slice(wa.Direction().AsVector()),
		IsOrigin:  wa.IsCoordinateSystemElement(),
		Healthy:   wa.Health().OK(),
		Reason:    wa.Health().Reason, // empty when healthy
	}
}

// buildWorkAxis dispatches a create request to the matching model constructor.
func buildWorkAxis(host workHost, in wire.CreateWorkAxisArgs) (*feature.WorkAxis, error) {
	axes := host.WorkAxes()
	switch types.WorkAxisKind(in.Kind) {
	case types.WorkAxisLine:
		return addLineAxis(axes, in)
	case types.WorkAxisTwoPoints:
		return addRefAxis(in, "two-points", axes.AddByTwoPoints)
	case types.WorkAxisPlaneIntersection:
		return addRefAxis(in, "plane-intersection", axes.AddByPlaneIntersection)
	default:
		return nil, fmt.Errorf("workAxes.create: unknown kind %q (see api/types WorkAxis*)", in.Kind)
	}
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

// addRefAxis builds a two-reference axis kind (two-points / plane-intersection) from exactly two
// work references, erroring on the wrong count.
func addRefAxis(in wire.CreateWorkAxisArgs, kind string, build func(a, b feature.WorkRef) *feature.WorkAxis) (*feature.WorkAxis, error) {
	refs := toWorkRefs(in.Refs)
	if len(refs) != 2 {
		return nil, fmt.Errorf("workAxes.create: %s needs 2 references, got %d", kind, len(refs))
	}
	return build(refs[0], refs[1]), nil
}
