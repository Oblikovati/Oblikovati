// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// ShapedEntity3D is the 3D twin of [ShapedEntity] (#1624). Free-form curves
// report a display sample rather than their defining points — the polyline the
// enumerate API has always shipped — and the surface-derived curves report
// nil, since their geometry is recompute-derived (only identity is shown).
type ShapedEntity3D interface {
	Entity
	Kind() EntityKind
	ShapePoints3D() []math.Point3
}

var (
	_ ShapedEntity3D = (*Point3D)(nil)
	_ ShapedEntity3D = (*Line3D)(nil)
	_ ShapedEntity3D = (*Circle3D)(nil)
	_ ShapedEntity3D = (*Arc3D)(nil)
	_ ShapedEntity3D = (*Ellipse3D)(nil)
	_ ShapedEntity3D = (*EllipticalArc3D)(nil)
	_ ShapedEntity3D = (*Spline3D)(nil)
	_ ShapedEntity3D = (*SplineHandle3D)(nil)
	_ ShapedEntity3D = (*FixedSpline3D)(nil)
	_ ShapedEntity3D = (*EquationCurve3D)(nil)
	_ ShapedEntity3D = (*HelicalCurve3D)(nil)
	_ ShapedEntity3D = (*IncludedPoint3D)(nil)
	_ ShapedEntity3D = (*IncludedCurve3D)(nil)
	_ ShapedEntity3D = (*IntersectionCurve3D)(nil)
	_ ShapedEntity3D = (*SilhouetteCurve3D)(nil)
	_ ShapedEntity3D = (*ProjectToSurfaceCurve3D)(nil)
	_ ShapedEntity3D = (*OnFaceCurve3D)(nil)
	_ ShapedEntity3D = (*OffsetCurve3)(nil)

	_ RadiusedEntity = (*Circle3D)(nil)
	_ RadiusedEntity = (*Arc3D)(nil)
	_ RadiusedEntity = (*HelicalCurve3D)(nil)
	_ RadiusedEntity = (*Ellipse3D)(nil)
	_ RadiusedEntity = (*EllipticalArc3D)(nil)
)

func (p *Point3D) ShapePoints3D() []math.Point3 { return []math.Point3{p.Position()} }
func (l *Line3D) ShapePoints3D() []math.Point3 {
	return []math.Point3{l.A.Position(), l.B.Position()}
}
func (c *Circle3D) ShapePoints3D() []math.Point3 { return []math.Point3{c.Center.Position()} }
func (a *Arc3D) ShapePoints3D() []math.Point3 {
	return []math.Point3{a.Center.Position(), a.Start.Position(), a.End.Position()}
}
func (e *Ellipse3D) ShapePoints3D() []math.Point3 { return []math.Point3{e.Center.Position()} }
func (e *EllipticalArc3D) ShapePoints3D() []math.Point3 {
	return []math.Point3{e.Center.Position()}
}
func (h *HelicalCurve3D) ShapePoints3D() []math.Point3 {
	return []math.Point3{h.Origin.Position()}
}
func (h *SplineHandle3D) ShapePoints3D() []math.Point3 {
	return []math.Point3{h.Anchor.Position(), h.End.Position()}
}

func (s *Spline3D) ShapePoints3D() []math.Point3        { return s.Sample() }
func (f *FixedSpline3D) ShapePoints3D() []math.Point3   { return f.Sample() }
func (e *EquationCurve3D) ShapePoints3D() []math.Point3 { return e.Sample(16) }

func (p *IncludedPoint3D) ShapePoints3D() []math.Point3 { return []math.Point3{p.Position()} }
func (c *IncludedCurve3D) ShapePoints3D() []math.Point3 { return c.Points() }

// The surface-derived curves rebind and re-evaluate on recompute; they carry
// no enumeration geometry of their own (M22-F11).
func (c *IntersectionCurve3D) ShapePoints3D() []math.Point3     { return nil }
func (c *SilhouetteCurve3D) ShapePoints3D() []math.Point3       { return nil }
func (c *ProjectToSurfaceCurve3D) ShapePoints3D() []math.Point3 { return nil }
func (c *OnFaceCurve3D) ShapePoints3D() []math.Point3           { return nil }
func (c *OffsetCurve3) ShapePoints3D() []math.Point3            { return nil }

func (c *Circle3D) ShapeRadius() float64        { return float64(c.Radius) }
func (a *Arc3D) ShapeRadius() float64           { return float64(a.Radius()) }
func (h *HelicalCurve3D) ShapeRadius() float64  { return float64(h.StartRadius) }
func (e *Ellipse3D) ShapeRadius() float64       { return float64(e.MajorRadius) }
func (e *EllipticalArc3D) ShapeRadius() float64 { return float64(e.MajorRadius) }
