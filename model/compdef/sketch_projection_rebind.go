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
			d.rebindProjection(e, sk.Plane())
		}
	}
}

// rebindProjection re-attaches one projected entity's live source from its persisted descriptor;
// an unknown/empty kind leaves the entity frozen.
func (d *PartComponentDefinition) rebindProjection(e sketch.Entity, plane sketch.Plane) {
	switch v := e.(type) {
	case *sketch.ProjectedPoint:
		if kind, id := v.SourceDescriptor(); kind != "" {
			if src, ok := d.pointRefSource(kind, id); ok {
				v.Rebind(src)
			}
		}
	case *sketch.ProjectedCurve:
		if kind, id := v.SourceDescriptor(); kind != "" {
			if src, ok := d.curveRefSource(kind, id, plane); ok {
				v.Rebind(src)
			}
		}
	}
}

// pointRefSource rebuilds a point projection source from its descriptor.
func (d *PartComponentDefinition) pointRefSource(kind, id string) (sketch.PointSource, bool) {
	switch kind {
	case "vertex":
		return NewVertexRefSource(d, id), true
	case "workPoint":
		return NewWorkPointRefSource(d, feature.WorkRef(id)), true
	default:
		return nil, false
	}
}

// curveRefSource rebuilds a curve projection source from its descriptor; a work-plane projection
// also needs the target sketch plane it intersects.
func (d *PartComponentDefinition) curveRefSource(kind, id string, plane sketch.Plane) (sketch.CurveSource, bool) {
	switch kind {
	case "edge":
		return NewEdgeRefSource(d, id), true
	case "workAxis":
		return NewWorkAxisRefSource(d, feature.WorkRef(id)), true
	case "workPlane":
		return NewWorkPlaneRefSource(d, feature.WorkRef(id), plane), true
	default:
		return nil, false
	}
}
