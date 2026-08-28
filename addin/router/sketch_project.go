// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

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
		return sk.ProjectCurve(compdef.NewEdgeRefSource(part, ref)).Entity(), true
	}
	if part.VertexKeyResolves(ref) {
		return sk.ProjectPoint(compdef.NewVertexRefSource(part, ref)), true
	}
	if part.WorkPointKeyResolves(ref) {
		return sk.ProjectPoint(compdef.NewWorkPointRefSource(part, feature.WorkRef(ref))), true
	}
	if part.WorkAxisKeyResolves(ref) {
		return sk.ProjectCurve(compdef.NewWorkAxisRefSource(part, feature.WorkRef(ref))).Entity(), true
	}
	if part.WorkPlaneKeyResolves(ref) {
		if !part.WorkPlaneIntersectsSketch(feature.WorkRef(ref), sk.Plane()) {
			return nil, false // parallel to the sketch: no intersection line to project
		}
		return sk.ProjectCurve(compdef.NewWorkPlaneRefSource(part, feature.WorkRef(ref), sk.Plane())).Entity(), true
	}
	return nil, false
}

// projectCutEdges projects the section curves where the sketch plane cuts the part solid as
// associative reference geometry — one projected curve per section loop (#1873).
func projectCutEdges(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ProjectCutEdgesArgs) (wire.ProjectGeometryResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ProjectGeometryResult{}, err
	}
	sources := part.CutEdgeSources(sk.Plane())
	if len(sources) == 0 {
		return wire.ProjectGeometryResult{}, fmt.Errorf("sketch.projectCutEdges: the sketch plane cuts no solid body")
	}
	created := make([]uint64, 0, len(sources))
	for _, c := range sk.ProjectCutEdges(sources) {
		created = append(created, uint64(c.Entity().EntityID()))
	}
	return wire.ProjectGeometryResult{Created: created, Healthy: true}, nil
}

// projectSilhouette projects the referenced face's silhouette onto the sketch plane as an
// associative reference curve, choosing the loop nearest the proximity point (#1873).
func projectSilhouette(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ProjectSilhouetteArgs) (wire.ProjectGeometryResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ProjectGeometryResult{}, err
	}
	prox, err := xyzPoint(in.ProximityPoint, "sketch.projectSilhouette: proximityPoint")
	if err != nil {
		return wire.ProjectGeometryResult{}, err
	}
	src, ok := part.SilhouetteSource(in.FaceRef, sk.Plane(), prox, in.IncludeBoundary)
	if !ok {
		return wire.ProjectGeometryResult{}, fmt.Errorf("sketch.projectSilhouette: face %q has no silhouette on the sketch plane", in.FaceRef)
	}
	c := sk.ProjectCurve(src)
	return wire.ProjectGeometryResult{Created: []uint64{uint64(c.Entity().EntityID())}, Healthy: true}, nil
}
