// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// Path is a connected chain of sketch entities used as a sweep/loft rail or guide.
// It may be open or closed; tangency continuity between consecutive entities is a
// property the feature engine can check at consumption (the chain order is provided
// here).
type Path struct {
	entities []ProfileEntity
	closed   bool
}

// Entities returns the path's ordered entities; Count returns their number.
func (p *Path) Entities() []ProfileEntity {
	out := make([]ProfileEntity, len(p.entities))
	copy(out, p.entities)
	return out
}
func (p *Path) Count() int { return len(p.entities) }

// IsClosed reports whether the path forms a closed loop.
func (p *Path) IsClosed() bool { return p.closed }

// Points returns the path's ordered vertices in sketch space — the endpoint chain of
// its segments, honoring each segment's traversal direction. A sweep maps these through
// the path's sketch plane to a 3D rail. (Curved segments contribute only their
// endpoints in phase A; sampling arcs into a polyline is a later refinement.)
func (p *Path) Points() []math.Point2 {
	var pts []math.Point2
	for i, pe := range p.entities {
		a, b, ok := segmentEnds(pe.Entity)
		if !ok {
			continue
		}
		if pe.reversed {
			a, b = b, a
		}
		if i == 0 {
			pts = append(pts, a.Position())
		}
		pts = append(pts, b.Position())
	}
	return pts
}

// Paths returns every maximal connected chain in the sketch — open and closed —
// from its non-construction geometry, for use as sweep/loft rails.
func (s *Sketch) Paths() []*Path {
	closed, open := detectLoops(s.normalGeometry())
	out := make([]*Path, 0, len(closed)+len(open))
	for _, l := range append(closed, open...) {
		out = append(out, &Path{entities: l.entities, closed: l.closed})
	}
	return out
}

// Path3D is a connected chain in 3D space (a sweep path from a 3D sketch). The 3D
// sketch entity model is built out incrementally; for now a Path3D is constructed
// directly from an ordered point chain, which is what the sweep feature consumes.
type Path3D struct {
	points []*Point3D
	closed bool
}

// NewPath3D builds a 3D path from an ordered chain of points.
func NewPath3D(points []*Point3D, closed bool) *Path3D {
	return &Path3D{points: append([]*Point3D(nil), points...), closed: closed}
}

// Points returns the path's ordered points as positions.
func (p *Path3D) Points() []math.Point3 {
	out := make([]math.Point3, len(p.points))
	for i, pt := range p.points {
		out[i] = pt.Position()
	}
	return out
}

// Count returns the number of points; IsClosed reports closure.
func (p *Path3D) Count() int     { return len(p.points) }
func (p *Path3D) IsClosed() bool { return p.closed }
