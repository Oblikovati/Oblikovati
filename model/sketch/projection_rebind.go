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

// RebindProjections re-attaches every restored projection's live source through the resolver, so
// projections become associative again after a reload; the next UpdateProjections re-projects from
// the live sources (#1268). Point projections (still entities) and curve projections (the Projection
// records that drive concrete reference entities, ADR-0055 phase 3) are both handled here, so the
// host needs no per-kind dispatch. A descriptor the resolver cannot resolve is left frozen.
func (s *Sketch) RebindProjections(res ProjectionSourceResolver) {
	for _, e := range s.ents {
		if p, ok := e.(*ProjectedPoint); ok {
			rebindPoint(p, res)
		}
	}
	for _, p := range s.projections {
		rebindCurve(p, res, s.plane)
	}
}

// rebindPoint re-attaches a projected point's live source; a legacy row without a descriptor stays
// frozen.
func rebindPoint(p *ProjectedPoint, res ProjectionSourceResolver) {
	kind, id := p.SourceDescriptor()
	if kind == "" {
		return
	}
	if src, ok := res.PointProjectionSource(kind, id); ok {
		p.Rebind(src)
	}
}

// rebindCurve re-attaches a curve projection's live source.
func rebindCurve(p *Projection, res ProjectionSourceResolver, plane Plane) {
	kind, id := p.SourceDescriptor()
	if kind == "" {
		return
	}
	if src, ok := res.CurveProjectionSource(kind, id, plane); ok {
		p.Rebind(src)
	}
}
