// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
)

// The 2D curve adapters, mirroring curves3.go in sketch space.

// curve2 supplies the umbrella members from the embedded kernel curve.
type curve2 struct {
	kind  types.Curve2dType
	form  types.CurveGeometryForm
	inner geom.Curve2
}

func (c curve2) CurveType() types.Curve2dType          { return c.kind }
func (c curve2) GeometryForm() types.CurveGeometryForm { return c.form }
func (c curve2) Evaluate(t float64) types.Point2d      { return toPoint2d(c.inner.PointAt(t)) }
func (c curve2) Tangent(t float64) types.Vector2d      { return toVector2d(c.inner.TangentAt(t)) }
func (c curve2) Domain() (lo, hi float64)              { return c.inner.Domain() }

func analytic2(kind types.Curve2dType, inner geom.Curve2) curve2 {
	return curve2{kind: kind, form: types.CurveFormNotNURBS, inner: inner}
}

// line2dAdapter — contract.Line2d over geom.Line2d.
type line2dAdapter struct {
	curve2
	g geom.Line2d
}

var _ contract.Line2d = line2dAdapter{}

func newLine2d(g geom.Line2d) line2dAdapter {
	return line2dAdapter{curve2: analytic2(types.LineCurve2d, g), g: g}
}

func (a line2dAdapter) RootPoint() types.Point2d      { return toPoint2d(a.g.Origin) }
func (a line2dAdapter) Direction() types.UnitVector2d { return toUnit2d(a.g.Dir) }

// segment2dAdapter — contract.LineSegment2d over geom.LineSegment2d.
type segment2dAdapter struct {
	curve2
	g geom.LineSegment2d
}

var _ contract.LineSegment2d = segment2dAdapter{}

func newSegment2d(g geom.LineSegment2d) segment2dAdapter {
	return segment2dAdapter{curve2: analytic2(types.LineSegmentCurve2d, g), g: g}
}

func (a segment2dAdapter) StartPoint() types.Point2d { return toPoint2d(a.g.StartPoint) }
func (a segment2dAdapter) EndPoint() types.Point2d   { return toPoint2d(a.g.EndPoint) }

// circle2dAdapter — contract.Circle2d over geom.Circle2d.
type circle2dAdapter struct {
	curve2
	g geom.Circle2d
}

var _ contract.Circle2d = circle2dAdapter{}

func newCircle2d(g geom.Circle2d) circle2dAdapter {
	return circle2dAdapter{curve2: analytic2(types.CircleCurve2d, g), g: g}
}

func (a circle2dAdapter) Center() types.Point2d { return toPoint2d(a.g.Center) }
func (a circle2dAdapter) Radius() float64       { return a.g.Radius }

// arc2dAdapter — contract.Arc2d over geom.Arc2d.
type arc2dAdapter struct {
	curve2
	g geom.Arc2d
}

var _ contract.Arc2d = arc2dAdapter{}

func newArc2d(g geom.Arc2d) arc2dAdapter {
	return arc2dAdapter{curve2: analytic2(types.CircularArcCurve2d, g), g: g}
}

func (a arc2dAdapter) Center() types.Point2d { return toPoint2d(a.g.Center) }
func (a arc2dAdapter) Radius() float64       { return a.g.Radius }
func (a arc2dAdapter) StartAngle() float64   { return a.g.StartAngle }
func (a arc2dAdapter) SweepAngle() float64   { return a.g.SweepAngle }

// ellipse2dAdapter — contract.EllipseFull2d over geom.EllipseFull2d.
type ellipse2dAdapter struct {
	curve2
	g geom.EllipseFull2d
}

var _ contract.EllipseFull2d = ellipse2dAdapter{}

func newEllipse2d(g geom.EllipseFull2d) ellipse2dAdapter {
	return ellipse2dAdapter{curve2: analytic2(types.EllipseFullCurve2d, g), g: g}
}

func (a ellipse2dAdapter) Center() types.Point2d         { return toPoint2d(a.g.Center) }
func (a ellipse2dAdapter) MajorAxis() types.UnitVector2d { return toUnit2d(a.g.MajorAxis) }
func (a ellipse2dAdapter) MajorRadius() float64          { return a.g.MajorRadius }
func (a ellipse2dAdapter) MinorRadius() float64          { return a.g.MinorRadius }

// ellipticalArc2dAdapter — contract.EllipticalArc2d over geom.EllipticalArc2d.
type ellipticalArc2dAdapter struct {
	curve2
	g geom.EllipticalArc2d
}

var _ contract.EllipticalArc2d = ellipticalArc2dAdapter{}

func newEllipticalArc2d(g geom.EllipticalArc2d) ellipticalArc2dAdapter {
	return ellipticalArc2dAdapter{curve2: analytic2(types.EllipticalArcCurve2d, g), g: g}
}

func (a ellipticalArc2dAdapter) Center() types.Point2d         { return toPoint2d(a.g.Center) }
func (a ellipticalArc2dAdapter) MajorAxis() types.UnitVector2d { return toUnit2d(a.g.MajorAxis) }
func (a ellipticalArc2dAdapter) MajorRadius() float64          { return a.g.MajorRadius }
func (a ellipticalArc2dAdapter) MinorRadius() float64          { return a.g.MinorRadius }
func (a ellipticalArc2dAdapter) StartAngle() float64           { return a.g.StartAngle }
func (a ellipticalArc2dAdapter) SweepAngle() float64           { return a.g.SweepAngle }

// polyline2dAdapter — contract.Polyline2d over geom.Polyline2d.
type polyline2dAdapter struct {
	curve2
	g geom.Polyline2d
}

var _ contract.Polyline2d = polyline2dAdapter{}

func newPolyline2d(g geom.Polyline2d) polyline2dAdapter {
	return polyline2dAdapter{curve2: analytic2(types.PolylineCurve2d, g), g: g}
}

func (a polyline2dAdapter) Points() []types.Point2d { return toPoints2d(a.g.Vertices) }

// bspline2dAdapter — contract.BSplineCurve2d over geom.BSplineCurve2d.
type bspline2dAdapter struct {
	curve2
	g geom.BSplineCurve2d
}

var _ contract.BSplineCurve2d = bspline2dAdapter{}

func newBSpline2d(g geom.BSplineCurve2d) bspline2dAdapter {
	return bspline2dAdapter{
		curve2: curve2{kind: types.BSplineCurve2dKind, form: types.CurveFormNURBS, inner: g},
		g:      g,
	}
}

func (a bspline2dAdapter) Definition() types.BSplineCurve2dDef {
	return types.BSplineCurve2dDef{
		Degree:  a.g.Degree,
		Poles:   toPoints2d(a.g.Ctrl),
		Weights: append([]float64(nil), a.g.Weights...),
		Knots:   append([]float64(nil), a.g.Knots...),
	}
}
