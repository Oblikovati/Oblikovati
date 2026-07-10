// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// A saved sketch restores its projected reference geometry (the projected origin centre point,
// projected edges/datums) frozen at its last position, carrying only a (kind, id) source
// descriptor — model/sketch cannot rebuild the part-owned source itself. rebindSketchProjections
// re-attaches a live, self-resolving source built from that descriptor, so projections become
// associative again after a reload. It runs after sketches/work geometry are restored and before
// Recompute, whose UpdateProjections then re-projects from the live sources (#1268).
func (d *PartComponentDefinition) rebindSketchProjections() {
	for i := 0; i < d.sketches.Count(); i++ {
		sk := d.sketches.Item(i)
		for _, e := range sk.Entities() {
			// Each projected entity re-attaches itself through the resolver
			// below — no per-kind dispatch here (#1624, audit I1).
			sketch.RebindProjection(e, d, sk.Plane())
		}
	}
}

// The part definition is the projection-source resolver: it owns the
// vertices/edges/work geometry the persisted descriptors point at.
var _ sketch.ProjectionSourceResolver = (*PartComponentDefinition)(nil)

// PointProjectionSource rebuilds a point projection source from its descriptor.
func (d *PartComponentDefinition) PointProjectionSource(kind, id string) (sketch.PointSource, bool) {
	switch kind {
	case "vertex":
		return NewVertexRefSource(d, id), true
	case "workPoint":
		return NewWorkPointRefSource(d, feature.WorkRef(id)), true
	default:
		return nil, false
	}
}

// CurveProjectionSource rebuilds a curve projection source from its descriptor; a work-plane
// projection also needs the target sketch plane it intersects.
func (d *PartComponentDefinition) CurveProjectionSource(kind, id string, plane sketch.Plane) (sketch.CurveSource, bool) {
	switch kind {
	case "edge":
		return NewEdgeRefSource(d, id), true
	case "workAxis":
		return NewWorkAxisRefSource(d, feature.WorkRef(id)), true
	case "workPlane":
		return NewWorkPlaneRefSource(d, feature.WorkRef(id), plane), true
	case "cutEdge":
		if bi, wi, ok := parseCutEdgeSourceID(id); ok {
			return NewCutEdgeRefSource(d, plane, bi, wi), true
		}
		return nil, false
	case "silhouette":
		if faceRef, prox, incl, ok := parseSilhouetteSourceID(id); ok {
			return NewSilhouetteRefSource(d, faceRef, plane, prox, incl), true
		}
		return nil, false
	default:
		return nil, false
	}
}
