// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
)

// The 3D curve adapters: each wraps a kernel curve and speaks the contract's
// value types. Evaluation delegates to the kernel; the adapters carry no state.

// curve3 supplies the umbrella members from the embedded kernel curve.
type curve3 struct {
	kind  types.CurveType
	form  types.CurveGeometryForm
	inner geom.Curve3
}

func (c curve3) CurveType() types.CurveType            { return c.kind }
func (c curve3) GeometryForm() types.CurveGeometryForm { return c.form }
func (c curve3) Evaluate(t float64) types.Point        { return toPoint(c.inner.PointAt(t)) }
func (c curve3) Tangent(t float64) types.Vector        { return toVector(c.inner.TangentAt(t)) }
func (c curve3) Domain() (lo, hi float64)              { return c.inner.Domain() }

// analytic3 / nurbs3 pick the geometry form for a kind.
func analytic3(kind types.CurveType, inner geom.Curve3) curve3 {
	return curve3{kind: kind, form: types.CurveFormNotNURBS, inner: inner}
}

// lineAdapter — contract.Line over geom.Line.
type lineAdapter struct {
	curve3
	g geom.Line
}

var _ contract.Line = lineAdapter{}

func newLine(g geom.Line) lineAdapter {
	return lineAdapter{curve3: analytic3(types.LineCurve, g), g: g}
}

func (a lineAdapter) RootPoint() types.Point      { return toPoint(a.g.Origin) }
func (a lineAdapter) Direction() types.UnitVector { return toUnit(a.g.Dir) }

// segmentAdapter — contract.LineSegment over geom.LineSegment.
type segmentAdapter struct {
	curve3
	g geom.LineSegment
}

var _ contract.LineSegment = segmentAdapter{}

func newSegment(g geom.LineSegment) segmentAdapter {
	return segmentAdapter{curve3: analytic3(types.LineSegmentCurve, g), g: g}
}

func (a segmentAdapter) StartPoint() types.Point { return toPoint(a.g.StartPoint) }
func (a segmentAdapter) EndPoint() types.Point   { return toPoint(a.g.EndPoint) }

// circleAdapter — contract.Circle over geom.Circle.
type circleAdapter struct {
	curve3
	g geom.Circle
}

var _ contract.Circle = circleAdapter{}

func newCircle(g geom.Circle) circleAdapter {
	return circleAdapter{curve3: analytic3(types.CircleCurve, g), g: g}
}

func (a circleAdapter) Center() types.Point      { return toPoint(a.g.Center) }
func (a circleAdapter) Normal() types.UnitVector { return toUnit(a.g.Normal) }
func (a circleAdapter) Radius() float64          { return a.g.Radius }

// arcAdapter — contract.Arc3d over geom.Arc3d.
type arcAdapter struct {
	curve3
	g geom.Arc3d
}

var _ contract.Arc3d = arcAdapter{}

func newArc(g geom.Arc3d) arcAdapter {
	return arcAdapter{curve3: analytic3(types.CircularArcCurve, g), g: g}
}

func (a arcAdapter) Center() types.Point               { return toPoint(a.g.Center) }
func (a arcAdapter) Normal() types.UnitVector          { return toUnit(a.g.Normal) }
func (a arcAdapter) ReferenceVector() types.UnitVector { return toUnit(a.g.RefDir) }
func (a arcAdapter) Radius() float64                   { return a.g.Radius }
func (a arcAdapter) StartAngle() float64               { return a.g.StartAngle }
func (a arcAdapter) SweepAngle() float64               { return a.g.SweepAngle }

// ellipseAdapter — contract.EllipseFull over geom.EllipseFull.
type ellipseAdapter struct {
	curve3
	g geom.EllipseFull
}

var _ contract.EllipseFull = ellipseAdapter{}

func newEllipse(g geom.EllipseFull) ellipseAdapter {
	return ellipseAdapter{curve3: analytic3(types.EllipseFullCurve, g), g: g}
}

func (a ellipseAdapter) Center() types.Point         { return toPoint(a.g.Center) }
func (a ellipseAdapter) Normal() types.UnitVector    { return toUnit(a.g.Normal) }
func (a ellipseAdapter) MajorAxis() types.UnitVector { return toUnit(a.g.MajorAxis) }
func (a ellipseAdapter) MajorRadius() float64        { return a.g.MajorRadius }
func (a ellipseAdapter) MinorRadius() float64        { return a.g.MinorRadius }

// ellipticalArcAdapter — contract.EllipticalArc over geom.EllipticalArc.
type ellipticalArcAdapter struct {
	curve3
	g geom.EllipticalArc
}

var _ contract.EllipticalArc = ellipticalArcAdapter{}

func newEllipticalArc(g geom.EllipticalArc) ellipticalArcAdapter {
	return ellipticalArcAdapter{curve3: analytic3(types.EllipticalArcCurve, g), g: g}
}

func (a ellipticalArcAdapter) Center() types.Point         { return toPoint(a.g.Center) }
func (a ellipticalArcAdapter) Normal() types.UnitVector    { return toUnit(a.g.Normal) }
func (a ellipticalArcAdapter) MajorAxis() types.UnitVector { return toUnit(a.g.MajorAxis) }
func (a ellipticalArcAdapter) MajorRadius() float64        { return a.g.MajorRadius }
func (a ellipticalArcAdapter) MinorRadius() float64        { return a.g.MinorRadius }
func (a ellipticalArcAdapter) StartAngle() float64         { return a.g.StartAngle }
func (a ellipticalArcAdapter) SweepAngle() float64         { return a.g.SweepAngle }

// polylineAdapter — contract.Polyline3d over geom.Polyline.
type polylineAdapter struct {
	curve3
	g geom.Polyline
}

var _ contract.Polyline3d = polylineAdapter{}

func newPolyline(g geom.Polyline) polylineAdapter {
	return polylineAdapter{curve3: analytic3(types.PolylineCurve, g), g: g}
}

func (a polylineAdapter) Points() []types.Point { return toPoints(a.g.Vertices) }

// bsplineAdapter — contract.BSplineCurve over geom.BSplineCurve.
type bsplineAdapter struct {
	curve3
	g geom.BSplineCurve
}

var _ contract.BSplineCurve = bsplineAdapter{}

func newBSpline(g geom.BSplineCurve) bsplineAdapter {
	return bsplineAdapter{
		curve3: curve3{kind: types.BSplineCurveKind, form: types.CurveFormNURBS, inner: g},
		g:      g,
	}
}

func (a bsplineAdapter) Definition() types.BSplineCurveDef {
	return types.BSplineCurveDef{
		Degree:  a.g.Degree,
		Poles:   toPoints(a.g.Ctrl),
		Weights: append([]float64(nil), a.g.Weights...),
		Knots:   append([]float64(nil), a.g.Knots...),
	}
}

// helixAdapter — contract.Helix over geom.Helix3d (the Oblikovati extension).
type helixAdapter struct {
	curve3
	g geom.Helix3d
}

var _ contract.Helix = helixAdapter{}

func newHelix(g geom.Helix3d) helixAdapter {
	return helixAdapter{curve3: analytic3(types.HelixCurve, g), g: g}
}

func (a helixAdapter) BasePoint() types.Point { return toPoint(a.g.Origin) }
func (a helixAdapter) Axis() types.UnitVector { return toUnit(a.g.Axis) }
func (a helixAdapter) StartRadius() float64   { return a.g.StartRadius }
func (a helixAdapter) Pitch() float64         { return a.g.AxialPerTurn }
func (a helixAdapter) TaperPerTurn() float64  { return a.g.RadialPerTurn }
func (a helixAdapter) Turns() float64         { return a.g.Turns }
