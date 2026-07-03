// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// projectGeometry projects the referenced part edges/vertices onto the sketch plane as
// associative reference entities (re-derived through recompute via their source keys).
func projectGeometry(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ProjectGeometryArgs) (wire.ProjectGeometryResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ProjectGeometryResult{}, err
	}
	created, healthy := projectRefs(part, sk, in.Refs)
	return wire.ProjectGeometryResult{Created: created, Healthy: healthy}, nil
}

// projectRefs resolves each reference to a part edge/vertex and projects it; an
// unresolved reference is skipped and the result reported unhealthy (not an error).
func projectRefs(part *compdef.PartComponentDefinition, sk *sketch.Sketch, refs []string) ([]uint64, bool) {
	var created []uint64
	healthy := true
	for _, ref := range refs {
		e, ok := projectRef(part, sk, ref)
		if !ok {
			healthy = false
			continue
		}
		created = append(created, uint64(e.EntityID()))
	}
	return created, healthy
}

// projectRef resolves one reference to a B-rep edge/vertex or a datum (work point/axis/plane)
// among the part's geometry and projects it, returning the created sketch entity. Datum
// references use the public [types.WorkRef] vocabulary (e.g. "origin/point/center"), so an
// automation client projects the origin centre point, axes and planes the same way the
// interactive tool does (#1262).
func projectRef(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ref string) (sketch.Entity, bool) {
	if part.EdgeKeyResolves(ref) {
		return sk.ProjectCurve(compdef.NewEdgeRefSource(part, ref)), true
	}
	if part.VertexKeyResolves(ref) {
		return sk.ProjectPoint(compdef.NewVertexRefSource(part, ref)), true
	}
	if part.WorkPointKeyResolves(ref) {
		return sk.ProjectPoint(compdef.NewWorkPointRefSource(part, feature.WorkRef(ref))), true
	}
	if part.WorkAxisKeyResolves(ref) {
		return sk.ProjectCurve(compdef.NewWorkAxisRefSource(part, feature.WorkRef(ref))), true
	}
	if part.WorkPlaneKeyResolves(ref) {
		if !part.WorkPlaneIntersectsSketch(feature.WorkRef(ref), sk.Plane()) {
			return nil, false // parallel to the sketch: no intersection line to project
		}
		return sk.ProjectCurve(compdef.NewWorkPlaneRefSource(part, feature.WorkRef(ref), sk.Plane())), true
	}
	return nil, false
}
