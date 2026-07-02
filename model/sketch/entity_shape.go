// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// ShapedEntity is the capability consumers used to type-switch for (#1624,
// audit I1): what kind am I, and which points sketch my shape for display or
// enumeration. Every 2D entity implements it (assertions below), so a lost
// method fails the build, not an API reply.
type ShapedEntity interface {
	Entity
	Kind() EntityKind
	// ShapePoints returns the points that outline the entity for enumeration:
	// defining points for analytic curves, the anchor for annotations, nil for
	// entities whose geometry is expression- or reference-derived.
	ShapePoints() []math.Point2
}

// RadiusedEntity is the optional radius capability — only the circular kinds
// carry one, so it stays separate from ShapedEntity (interface segregation);
// consumers report 0 for everything else, as the retired switches did.
type RadiusedEntity interface{ ShapeRadius() float64 }

var (
	_ ShapedEntity = (*Point)(nil)
	_ ShapedEntity = (*Line)(nil)
	_ ShapedEntity = (*Circle)(nil)
	_ ShapedEntity = (*Arc)(nil)
	_ ShapedEntity = (*Ellipse)(nil)
	_ ShapedEntity = (*EllipticalArc)(nil)
	_ ShapedEntity = (*Spline)(nil)
	_ ShapedEntity = (*SplineHandle)(nil)
	_ ShapedEntity = (*SketchImage)(nil)
	_ ShapedEntity = (*FillRegion)(nil)
	_ ShapedEntity = (*TextBox)(nil)
	_ ShapedEntity = (*EquationCurve)(nil)
	_ ShapedEntity = (*FixedSpline)(nil)
	_ ShapedEntity = (*OffsetSpline)(nil)
	_ ShapedEntity = (*BlockInstance)(nil)
	_ ShapedEntity = (*ProjectedPoint)(nil)
	_ ShapedEntity = (*ProjectedCurve)(nil)

	_ RadiusedEntity = (*Circle)(nil)
	_ RadiusedEntity = (*Arc)(nil)
)

func (p *Point) ShapePoints() []math.Point2 { return []math.Point2{p.Position()} }
func (l *Line) ShapePoints() []math.Point2 {
	return []math.Point2{l.A.Position(), l.B.Position()}
}
func (c *Circle) ShapePoints() []math.Point2 { return []math.Point2{c.Center.Position()} }
func (a *Arc) ShapePoints() []math.Point2 {
	return []math.Point2{a.Center.Position(), a.Start.Position(), a.End.Position()}
}
func (e *Ellipse) ShapePoints() []math.Point2       { return []math.Point2{e.Center.Position()} }
func (e *EllipticalArc) ShapePoints() []math.Point2 { return []math.Point2{e.Center.Position()} }

func (s *Spline) ShapePoints() []math.Point2 {
	out := make([]math.Point2, len(s.Points))
	for i, p := range s.Points {
		out[i] = p.Position()
	}
	return out
}

func (h *SplineHandle) ShapePoints() []math.Point2 {
	return []math.Point2{h.Anchor.Position(), h.End.Position()}
}

func (i *SketchImage) ShapePoints() []math.Point2    { return []math.Point2{i.Anchor} }
func (f *FillRegion) ShapePoints() []math.Point2     { return []math.Point2{f.Seed} }
func (t *TextBox) ShapePoints() []math.Point2        { return []math.Point2{t.Anchor} }
func (e *EquationCurve) ShapePoints() []math.Point2  { return nil }
func (f *FixedSpline) ShapePoints() []math.Point2    { return append([]math.Point2(nil), f.Pts...) }
func (o *OffsetSpline) ShapePoints() []math.Point2   { return nil }
func (b *BlockInstance) ShapePoints() []math.Point2  { return nil }
func (p *ProjectedPoint) ShapePoints() []math.Point2 { return []math.Point2{p.Position()} }
func (c *ProjectedCurve) ShapePoints() []math.Point2 { return c.Points() }

func (c *Circle) ShapeRadius() float64 { return float64(c.Radius) }
func (a *Arc) ShapeRadius() float64    { return float64(a.Radius()) }
