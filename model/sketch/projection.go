// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// Projection links model geometry into a sketch as reference geometry that updates
// when the source changes. The model side is reached through a seam — [PointSource]
// / [CurveSource] — so the sketch never depends on the B-rep kernel directly; the
// kernel's vertices/edges implement these in M07 (the same seam discipline used for
// reference keys). Projected geometry is construction/reference by default.

// PointSource is a model entity that yields a 3D position to project (e.g. a topo
// vertex). SourceID is a stable identity used to recognize the source across
// recompute for associative re-projection.
type PointSource interface {
	SourceID() string
	Position() math.Point3
}

// CurveSource is a model entity that yields a sampled 3D polyline to project (e.g.
// a topo edge's tessellation, or a plane-cut edge).
type CurveSource interface {
	SourceID() string
	SamplePoints() []math.Point3
}

// ProjectedPoint is a sketch point projected from a model vertex. It caches the
// sketch-space position and re-projects on [ProjectedPoint.Update] until the link
// is broken.
type ProjectedPoint struct {
	id        ID
	source    PointSource
	plane     Plane
	sketchPos math.Point2
	linked    bool
}

// EntityID implements [Entity].
func (p *ProjectedPoint) EntityID() ID { return p.id }

// Position returns the cached sketch-space position.
func (p *ProjectedPoint) Position() math.Point2 { return p.sketchPos }

// SourceID returns the projected source's identity, or "" if the link was broken.
func (p *ProjectedPoint) SourceID() string {
	if !p.linked {
		return ""
	}
	return p.source.SourceID()
}

// IsReference reports that projected geometry is reference/construction geometry.
func (p *ProjectedPoint) IsReference() bool { return true }

// Linked reports whether the projection still tracks its source.
func (p *ProjectedPoint) Linked() bool { return p.linked }

// Update re-projects from the current source position. It is a no-op once the link
// is broken, so the last projected position is frozen.
func (p *ProjectedPoint) Update() {
	if p.linked {
		p.sketchPos = p.plane.ToSketch(p.source.Position())
	}
}

// BreakLink detaches the projection from its source, freezing its current geometry
// (the "break link" / include-without-associativity option).
func (p *ProjectedPoint) BreakLink() { p.linked = false }

// ProjectedCurve is a sketch polyline projected from a model edge (or cut edge).
type ProjectedCurve struct {
	id     ID
	source CurveSource
	plane  Plane
	points []math.Point2
	linked bool
}

// EntityID implements [Entity].
func (c *ProjectedCurve) EntityID() ID { return c.id }

// Points returns the cached sketch-space polyline.
func (c *ProjectedCurve) Points() []math.Point2 {
	out := make([]math.Point2, len(c.points))
	copy(out, c.points)
	return out
}

// SourceID returns the source identity, or "" once the link is broken.
func (c *ProjectedCurve) SourceID() string {
	if !c.linked {
		return ""
	}
	return c.source.SourceID()
}

// IsReference reports that projected geometry is reference/construction geometry.
func (c *ProjectedCurve) IsReference() bool { return true }

// Linked reports whether the projection still tracks its source.
func (c *ProjectedCurve) Linked() bool { return c.linked }

// Update re-projects the source's sample points onto the sketch plane.
func (c *ProjectedCurve) Update() {
	if !c.linked {
		return
	}
	src := c.source.SamplePoints()
	c.points = c.points[:0]
	for _, q := range src {
		c.points = append(c.points, c.plane.ToSketch(q))
	}
}

// BreakLink detaches the projection from its source, freezing the current polyline.
func (c *ProjectedCurve) BreakLink() { c.linked = false }

// ProjectPoint projects a model vertex into the sketch as reference geometry and
// returns the associative projected point.
func (s *Sketch) ProjectPoint(src PointSource) *ProjectedPoint {
	p := &ProjectedPoint{id: nextID(), source: src, plane: s.plane, linked: true}
	p.Update()
	s.add(p)
	return p
}

// ProjectCurve projects a model edge into the sketch as reference geometry.
func (s *Sketch) ProjectCurve(src CurveSource) *ProjectedCurve {
	c := &ProjectedCurve{id: nextID(), source: src, plane: s.plane, linked: true}
	c.Update()
	s.add(c)
	return c
}

// ProjectCutEdges projects the edges where the sketch plane cuts the model. The cut
// edges are computed by the kernel (M07) and supplied as sources; the sketch maps
// them to its plane and adds them as reference geometry.
func (s *Sketch) ProjectCutEdges(sources []CurveSource) []*ProjectedCurve {
	out := make([]*ProjectedCurve, 0, len(sources))
	for _, src := range sources {
		out = append(out, s.ProjectCurve(src))
	}
	return out
}
