// SPDX-License-Identifier: GPL-2.0-only

package sketch

// ProjectionSourceResolver rebuilds a live projection source from a persisted
// (kind, id) descriptor. The part definition implements it — it owns the
// referenced vertices/edges/work geometry that model/sketch cannot see (#1268).
type ProjectionSourceResolver interface {
	PointProjectionSource(kind, id string) (PointSource, bool)
	// CurveProjectionSource also receives the target sketch plane: a work-plane
	// projection is the line where the two planes intersect.
	CurveProjectionSource(kind, id string, plane Plane) (CurveSource, bool)
}

// sourceRebindable is the sealed rebind capability: each projected entity
// re-attaches its own source through the resolver, so no consumer needs to
// know which projected kinds exist (#1624, audit I1).
type sourceRebindable interface {
	rebindFrom(res ProjectionSourceResolver, plane Plane)
}

func (p *ProjectedPoint) rebindFrom(res ProjectionSourceResolver, _ Plane) {
	kind, id := p.SourceDescriptor()
	if kind == "" {
		return // legacy row without a descriptor stays frozen
	}
	if src, ok := res.PointProjectionSource(kind, id); ok {
		p.Rebind(src)
	}
}

func (c *ProjectedCurve) rebindFrom(res ProjectionSourceResolver, plane Plane) {
	kind, id := c.SourceDescriptor()
	if kind == "" {
		return
	}
	if src, ok := res.CurveProjectionSource(kind, id, plane); ok {
		c.Rebind(src)
	}
}

// RebindProjection re-attaches e's live projection source after a reload,
// making the projection associative again; non-projected entities and
// unresolvable descriptors are left frozen.
//
//	sketch.RebindProjection(entity, partDef, sk.Plane())
func RebindProjection(e Entity, res ProjectionSourceResolver, plane Plane) {
	if r, ok := e.(sourceRebindable); ok {
		r.rebindFrom(res, plane)
	}
}
