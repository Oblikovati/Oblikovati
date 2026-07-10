// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Occurrence-qualified datum listing (#1857): when a workPlanes/Axes/Points.list request carries an
// Occurrence path, it lists that sub-component's datums instead of the active host's own — each
// returned as an occurrence-qualified ref ("occ/<path>/plane|axis|point/N") with its geometry
// transformed into the assembly's space. The returned rows are read-only proxies (no editable
// scalars/slots): a proxy is a projection of the native datum, redefined only in its own component.

// occurrenceContext resolves the active assembly host + occurrence path to the composed transform
// and the target component's work geometry, erroring when the host is not an assembly or the path
// names no part.
func occurrenceContext(host workHost, path []string, method string) (math.Matrix4, *feature.WorkGeometry, error) {
	asm, ok := host.(*compdef.AssemblyComponentDefinition)
	if !ok {
		return math.Identity4(), nil, fmt.Errorf("%s: an occurrence path is only valid in an assembly context", method)
	}
	transform, work, ok := asm.ResolveOccurrenceContext(path)
	if !ok {
		return math.Identity4(), nil, fmt.Errorf("%s: occurrence path %v resolves to no part component", method, path)
	}
	return transform, work, nil
}

// listWorkPlanesInOccurrence lists a sub-component's datum planes as occurrence-qualified refs (#1857).
func listWorkPlanesInOccurrence(host workHost, path []string, includeConstruction bool) (wire.ListWorkPlanesResult, error) {
	xf, work, err := occurrenceContext(host, path, wire.MethodWorkPlanesList)
	if err != nil {
		return wire.ListWorkPlanesResult{}, err
	}
	planes := projectListedDatums(work.WorkPlanes(), includeConstruction, func(i int, wp *feature.WorkPlane) wire.WorkPlaneInfo {
		pl := wp.Plane()
		return wire.WorkPlaneInfo{
			Index:        i,
			Name:         wp.Name(),
			Ref:          string(feature.EncodeOccurrenceRef(path, wp.Key())),
			Origin:       point3Slice(xf.TransformPoint(pl.Origin())),
			Normal:       vector3Slice(xf.TransformVector(pl.Normal().AsVector())),
			IsOrigin:     wp.IsCoordinateSystemElement(),
			Construction: wp.Construction(),
			Healthy:      wp.Health().OK(),
			Reason:       wp.Health().Reason,
			Kind:         wp.Kind(),
		}
	})
	return wire.ListWorkPlanesResult{Planes: planes}, nil
}

// listWorkAxesInOccurrence lists a sub-component's datum axes as occurrence-qualified refs (#1857).
func listWorkAxesInOccurrence(host workHost, path []string, includeConstruction bool) (wire.ListWorkAxesResult, error) {
	xf, work, err := occurrenceContext(host, path, wire.MethodWorkAxesList)
	if err != nil {
		return wire.ListWorkAxesResult{}, err
	}
	axes := projectListedDatums(work.WorkAxes(), includeConstruction, func(i int, wa *feature.WorkAxis) wire.WorkAxisInfo {
		dir, derr := xf.TransformUnitVector(wa.Direction())
		if derr != nil {
			dir = wa.Direction()
		}
		return wire.WorkAxisInfo{
			Index:        i,
			Name:         wa.Name(),
			Ref:          string(feature.EncodeOccurrenceRef(path, wa.Key())),
			Kind:         wa.Kind(),
			Origin:       point3Slice(xf.TransformPoint(wa.Origin())),
			Direction:    vector3Slice(dir.AsVector()),
			IsOrigin:     wa.IsCoordinateSystemElement(),
			Construction: wa.Construction(),
			Healthy:      wa.Health().OK(),
			Reason:       wa.Health().Reason,
		}
	})
	return wire.ListWorkAxesResult{Axes: axes}, nil
}

// listWorkPointsInOccurrence lists a sub-component's datum points as occurrence-qualified refs (#1857).
func listWorkPointsInOccurrence(host workHost, path []string, includeConstruction bool) (wire.ListWorkPointsResult, error) {
	xf, work, err := occurrenceContext(host, path, wire.MethodWorkPointsList)
	if err != nil {
		return wire.ListWorkPointsResult{}, err
	}
	points := projectListedDatums(work.WorkPoints(), includeConstruction, func(i int, wp *feature.WorkPoint) wire.WorkPointInfo {
		return wire.WorkPointInfo{
			Index:        i,
			Name:         wp.Name(),
			Ref:          string(feature.EncodeOccurrenceRef(path, wp.Key())),
			Position:     point3Slice(xf.TransformPoint(wp.Point())),
			IsOrigin:     wp.IsCoordinateSystemElement(),
			Visible:      wp.Visible(),
			Construction: wp.Construction(),
			Healthy:      wp.Health().OK(),
			Reason:       wp.Health().Reason,
			Kind:         wp.Kind(),
		}
	})
	return wire.ListWorkPointsResult{Points: points}, nil
}
