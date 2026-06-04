// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// Include links part geometry (edges/vertices) into a 3D sketch as associative reference
// geometry — Inventor's "Include Geometry" for a 3D sketch. It reuses the model-side
// source seam ([PointSource]/[CurveSource]) so the sketch never depends on the B-rep
// kernel directly; the kernel's vertices/edges adapt to those interfaces (M22-F08
// PBI-241). Unlike the 2D projection there is no host plane: included geometry is the
// source's model-space position/polyline verbatim. Included entities are reference
// (construction) geometry and recompute-derived, so serialization skips them and they
// rebind from their source on recompute (via SourceID).

// IncludedPoint3D is a model vertex included into a 3D sketch, tracking its source. It
// owns a fixed reference anchor (a Point3D in the sketch's refPts) so other geometry can
// be constrained to it; the anchor's id is the entity's id (one identity for the include
// and the point you snap to).
type IncludedPoint3D struct {
	source PointSource
	anchor *Point3D
	linked bool
}

// EntityID is the anchor's id — constraining to this id binds to the fixed anchor.
func (p *IncludedPoint3D) EntityID() ID { return p.anchor.EntityID() }

// IsConstruction reports that an included point is reference/construction geometry.
func (p *IncludedPoint3D) IsConstruction() bool { return true }

// Anchor returns the constrainable fixed reference point.
func (p *IncludedPoint3D) Anchor() *Point3D { return p.anchor }

// Position returns the anchor's model-space position.
func (p *IncludedPoint3D) Position() math.Point3 { return p.anchor.Position() }

// SourceID returns the source identity, or "" once the link is broken.
func (p *IncludedPoint3D) SourceID() string {
	if !p.linked {
		return ""
	}
	return p.source.SourceID()
}

// Linked reports whether the include still tracks its source.
func (p *IncludedPoint3D) Linked() bool { return p.linked }

// Update re-reads the source position into the anchor; a no-op once the link is broken. A
// lost reference breaks the link, freezing the last position.
func (p *IncludedPoint3D) Update() {
	if !p.linked {
		return
	}
	pos, ok := p.source.Position()
	if !ok {
		p.linked = false
		return
	}
	p.anchor.SetPosition(pos)
}

// BreakLink detaches the include from its source, freezing the current position.
func (p *IncludedPoint3D) BreakLink() { p.linked = false }

// IncludedCurve3D is a model edge included into a 3D sketch as a reference polyline.
type IncludedCurve3D struct {
	entityBase
	source CurveSource
	points []math.Point3
	linked bool
}

// Points returns the cached model-space polyline.
func (c *IncludedCurve3D) Points() []math.Point3 {
	out := make([]math.Point3, len(c.points))
	copy(out, c.points)
	return out
}

// SourceID returns the source identity, or "" once the link is broken.
func (c *IncludedCurve3D) SourceID() string {
	if !c.linked {
		return ""
	}
	return c.source.SourceID()
}

// Linked reports whether the include still tracks its source.
func (c *IncludedCurve3D) Linked() bool { return c.linked }

// Update re-reads the source's sample points; a no-op once the link is broken. A lost
// reference breaks the link, freezing the last polyline.
func (c *IncludedCurve3D) Update() {
	if !c.linked {
		return
	}
	pts, ok := c.source.SamplePoints()
	if !ok {
		c.linked = false
		return
	}
	c.points = append(c.points[:0], pts...)
}

// BreakLink detaches the include from its source, freezing the current polyline.
func (c *IncludedCurve3D) BreakLink() { c.linked = false }

// IncludePoint3D includes a model vertex into the sketch as a fixed, constrainable
// reference point.
func (s *Sketch3D) IncludePoint3D(src PointSource) *IncludedPoint3D {
	pos, _ := src.Position() // resolved now; a later lost reference freezes via Update
	p := &IncludedPoint3D{source: src, anchor: s.newRefPoint3D(pos), linked: true}
	p.Update()
	s.addEntity3D(p)
	return p
}

// IncludeCurve3D includes a model edge into the sketch as a reference polyline.
func (s *Sketch3D) IncludeCurve3D(src CurveSource) *IncludedCurve3D {
	c := &IncludedCurve3D{entityBase: newEntity(), source: src, linked: true}
	c.SetConstruction(true)
	c.Update()
	s.addEntity3D(c)
	return c
}

// UpdateIncluded re-projects every linked included entity from its source — the hook a
// recompute calls so included geometry tracks the part as it changes.
func (s *Sketch3D) UpdateIncluded() {
	for _, e := range s.ents {
		switch v := e.(type) {
		case *IncludedPoint3D:
			v.Update()
		case *IncludedCurve3D:
			v.Update()
		}
	}
}
