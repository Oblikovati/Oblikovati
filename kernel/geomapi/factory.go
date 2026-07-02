// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Factory is the host's contract.TransientGeometry: every constructor delegates
// to the kernel's validated geom constructors and wraps the result in its
// contract adapter. It is stateless — one shared instance serves every caller.
type Factory struct{}

var _ contract.TransientGeometry = Factory{}

// The umbrella curve/surface contracts are satisfied by the shared adapter
// bases every concrete adapter embeds (#1619, ADR-0018: implicit satisfaction
// at usage sites is not enforcement — the assertion is the tripwire).
var (
	_ contract.Curve   = curve3{}
	_ contract.Curve2d = curve2{}
	_ contract.Surface = surface{}
)

// New returns the transient-geometry factory.
func New() Factory { return Factory{} }

// positive guards the radii/extents the kernel constructors do not all validate
// themselves, naming the offending value.
func positive(name string, v float64) error {
	if v <= 0 {
		return fmt.Errorf("geomapi: %s %g is not positive", name, v)
	}
	return nil
}

func (Factory) CreateLine(root types.Point, direction types.UnitVector) (contract.Line, error) {
	g, err := geom.NewLine(fromPoint(root), fromUnit(direction))
	if err != nil {
		return nil, err
	}
	return newLine(g), nil
}

func (Factory) CreateLineSegment(start, end types.Point) (contract.LineSegment, error) {
	if start == end {
		return nil, fmt.Errorf("geomapi: degenerate segment: start and end are both %+v", start)
	}
	return newSegment(geom.NewLineSegment(fromPoint(start), fromPoint(end))), nil
}

func (Factory) CreateCircle(center types.Point, normal types.UnitVector, radius float64) (contract.Circle, error) {
	if err := positive("circle radius", radius); err != nil {
		return nil, err
	}
	g, err := geom.NewCircle(fromPoint(center), fromUnit(normal), radius)
	if err != nil {
		return nil, err
	}
	return newCircle(g), nil
}

func (Factory) CreateCircleByThreePoints(a, b, c types.Point) (contract.Circle, error) {
	g, err := geom.CircleByThreePoints(fromPoint(a), fromPoint(b), fromPoint(c))
	if err != nil {
		return nil, err
	}
	return newCircle(g), nil
}

func (Factory) CreateArc(center types.Point, normal, reference types.UnitVector, radius, startAngle, sweepAngle float64) (contract.Arc3d, error) {
	if err := positive("arc radius", radius); err != nil {
		return nil, err
	}
	g, err := geom.NewArc3d(fromPoint(center), fromUnit(normal), fromUnit(reference), radius, startAngle, sweepAngle)
	if err != nil {
		return nil, err
	}
	return newArc(g), nil
}

func (Factory) CreateArcByThreePoints(start, on, end types.Point) (contract.Arc3d, error) {
	g, err := geom.Arc3dByThreePoints(fromPoint(start), fromPoint(on), fromPoint(end))
	if err != nil {
		return nil, err
	}
	return newArc(g), nil
}

func (Factory) CreateEllipseFull(center types.Point, normal, majorAxis types.UnitVector, majorRadius, minorRadius float64) (contract.EllipseFull, error) {
	if err := positive("ellipse major radius", majorRadius); err != nil {
		return nil, err
	}
	if err := positive("ellipse minor radius", minorRadius); err != nil {
		return nil, err
	}
	g, err := geom.NewEllipseFull(fromPoint(center), fromUnit(normal), fromUnit(majorAxis), majorRadius, minorRadius)
	if err != nil {
		return nil, err
	}
	return newEllipse(g), nil
}

func (Factory) CreateEllipticalArc(center types.Point, normal, majorAxis types.UnitVector, majorRadius, minorRadius, startAngle, sweepAngle float64) (contract.EllipticalArc, error) {
	g, err := geom.NewEllipticalArc(fromPoint(center), fromUnit(normal), fromUnit(majorAxis), majorRadius, minorRadius, startAngle, sweepAngle)
	if err != nil {
		return nil, err
	}
	return newEllipticalArc(g), nil
}

func (Factory) CreatePolyline(points []types.Point) (contract.Polyline3d, error) {
	g, err := geom.NewPolyline(fromPoints(points))
	if err != nil {
		return nil, err
	}
	return newPolyline(g), nil
}

func (Factory) CreateBSplineCurve(def types.BSplineCurveDef) (contract.BSplineCurve, error) {
	g, err := bsplineFromDef(def)
	if err != nil {
		return nil, err
	}
	return newBSpline(g), nil
}

// bsplineFromDef builds the kernel spline, defaulting nil weights to uniform.
func bsplineFromDef(def types.BSplineCurveDef) (geom.BSplineCurve, error) {
	if def.Weights == nil {
		return geom.NewBSplineCurveUniformWeights(def.Degree, fromPoints(def.Poles), def.Knots)
	}
	return geom.NewBSplineCurve(def.Degree, fromPoints(def.Poles), def.Weights, def.Knots)
}

func (Factory) CreateFittedBSplineCurve(through []types.Point) (contract.BSplineCurve, error) {
	g, err := geom.NewFittedBSplineCurve(fromPoints(through))
	if err != nil {
		return nil, err
	}
	return newBSpline(g), nil
}

func (Factory) CreateHelix(base types.Point, axis, reference types.UnitVector, startRadius, pitch, taperPerTurn, turns float64, clockwise bool) (contract.Helix, error) {
	g, err := geom.NewHelix3d(fromPoint(base), fromUnit(axis), fromUnit(reference), startRadius, pitch, taperPerTurn, turns, clockwise)
	if err != nil {
		return nil, err
	}
	return newHelix(g), nil
}

func (Factory) CreateLine2d(root types.Point2d, direction types.UnitVector2d) (contract.Line2d, error) {
	g, err := geom.NewLine2d(fromPoint2d(root), fromUnit2d(direction))
	if err != nil {
		return nil, err
	}
	return newLine2d(g), nil
}

func (Factory) CreateLineSegment2d(start, end types.Point2d) (contract.LineSegment2d, error) {
	if start == end {
		return nil, fmt.Errorf("geomapi: degenerate 2D segment: start and end are both %+v", start)
	}
	return newSegment2d(geom.NewLineSegment2d(fromPoint2d(start), fromPoint2d(end))), nil
}

func (Factory) CreateCircle2d(center types.Point2d, radius float64) (contract.Circle2d, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("geomapi: circle radius %g is not positive", radius)
	}
	return newCircle2d(geom.NewCircle2d(fromPoint2d(center), radius)), nil
}

func (Factory) CreateArc2d(center types.Point2d, radius, startAngle, sweepAngle float64) (contract.Arc2d, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("geomapi: arc radius %g is not positive", radius)
	}
	return newArc2d(geom.NewArc2d(fromPoint2d(center), radius, startAngle, sweepAngle)), nil
}

func (Factory) CreateEllipseFull2d(center types.Point2d, majorAxis types.UnitVector2d, majorRadius, minorRadius float64) (contract.EllipseFull2d, error) {
	g, err := geom.NewEllipseFull2d(fromPoint2d(center), fromUnit2d(majorAxis), majorRadius, minorRadius)
	if err != nil {
		return nil, err
	}
	return newEllipse2d(g), nil
}

func (Factory) CreateEllipticalArc2d(center types.Point2d, majorAxis types.UnitVector2d, majorRadius, minorRadius, startAngle, sweepAngle float64) (contract.EllipticalArc2d, error) {
	g, err := geom.NewEllipticalArc2d(fromPoint2d(center), fromUnit2d(majorAxis), majorRadius, minorRadius, startAngle, sweepAngle)
	if err != nil {
		return nil, err
	}
	return newEllipticalArc2d(g), nil
}

func (Factory) CreatePolyline2d(points []types.Point2d) (contract.Polyline2d, error) {
	g, err := geom.NewPolyline2d(fromPoints2d(points))
	if err != nil {
		return nil, err
	}
	return newPolyline2d(g), nil
}

func (Factory) CreateBSplineCurve2d(def types.BSplineCurve2dDef) (contract.BSplineCurve2d, error) {
	g, err := bspline2dFromDef(def)
	if err != nil {
		return nil, err
	}
	return newBSpline2d(g), nil
}

// bspline2dFromDef builds the kernel spline, defaulting nil weights to uniform.
func bspline2dFromDef(def types.BSplineCurve2dDef) (geom.BSplineCurve2d, error) {
	if def.Weights == nil {
		return geom.NewBSplineCurve2dUniformWeights(def.Degree, fromPoints2d(def.Poles), def.Knots)
	}
	return geom.NewBSplineCurve2d(def.Degree, fromPoints2d(def.Poles), def.Weights, def.Knots)
}

func (Factory) CreateFittedBSplineCurve2d(through []types.Point2d) (contract.BSplineCurve2d, error) {
	g, err := geom.NewFittedBSplineCurve2d(fromPoints2d(through))
	if err != nil {
		return nil, err
	}
	return newBSpline2d(g), nil
}

func (Factory) CreatePlane(root types.Point, normal types.UnitVector) (contract.Plane, error) {
	g, err := geom.NewPlane(fromPoint(root), fromUnit(normal))
	if err != nil {
		return nil, err
	}
	return newPlane(g), nil
}

func (Factory) CreatePlaneByThreePoints(a, b, c types.Point) (contract.Plane, error) {
	g, err := geom.PlaneByThreePoints(fromPoint(a), fromPoint(b), fromPoint(c))
	if err != nil {
		return nil, err
	}
	return newPlane(g), nil
}

func (Factory) CreateCylinder(base types.Point, axis types.UnitVector, radius float64) (contract.Cylinder, error) {
	if err := positive("cylinder radius", radius); err != nil {
		return nil, err
	}
	g, err := geom.NewCylinder(fromPoint(base), fromUnit(axis), radius)
	if err != nil {
		return nil, err
	}
	return newCylinder(g), nil
}

func (Factory) CreateCone(apex types.Point, axis types.UnitVector, halfAngle float64) (contract.Cone, error) {
	g, err := geom.NewCone(fromPoint(apex), fromUnit(axis), halfAngle)
	if err != nil {
		return nil, err
	}
	return newCone(g), nil
}

func (Factory) CreateSphere(center types.Point, radius float64) (contract.Sphere, error) {
	g, err := geom.NewSphere(fromPoint(center), radius)
	if err != nil {
		return nil, err
	}
	return newSphere(g), nil
}

func (Factory) CreateTorus(center types.Point, axis types.UnitVector, majorRadius, minorRadius float64) (contract.Torus, error) {
	if err := positive("torus major radius", majorRadius); err != nil {
		return nil, err
	}
	if err := positive("torus minor radius", minorRadius); err != nil {
		return nil, err
	}
	g, err := geom.NewTorus(fromPoint(center), fromUnit(axis), majorRadius, minorRadius)
	if err != nil {
		return nil, err
	}
	return newTorus(g), nil
}

func (Factory) CreateEllipticalCylinder(base types.Point, axis, majorAxis types.UnitVector, majorRadius, minorRadius float64) (contract.EllipticalCylinder, error) {
	g, err := geom.NewEllipticalCylinder(fromPoint(base), fromUnit(axis), fromUnit(majorAxis), majorRadius, minorRadius)
	if err != nil {
		return nil, err
	}
	return newEllipticalCylinder(g), nil
}

func (Factory) CreateEllipticalCone(apex types.Point, axis, majorAxis types.UnitVector, majorHalfAngle, minorHalfAngle float64) (contract.EllipticalCone, error) {
	g, err := geom.NewEllipticalCone(fromPoint(apex), fromUnit(axis), fromUnit(majorAxis), majorHalfAngle, minorHalfAngle)
	if err != nil {
		return nil, err
	}
	return newEllipticalCone(g), nil
}

func (Factory) CreateBSplineSurface(def types.BSplineSurfaceDef) (contract.BSplineSurface, error) {
	if def.PolesU <= 0 || def.PolesV <= 0 || len(def.Poles) != def.PolesU*def.PolesV {
		return nil, fmt.Errorf("geomapi: surface poles %d do not fill the %d×%d net", len(def.Poles), def.PolesU, def.PolesV)
	}
	ctrl := make([][]math.Point3, def.PolesU)
	for i := 0; i < def.PolesU; i++ {
		ctrl[i] = fromPoints(def.Poles[i*def.PolesV : (i+1)*def.PolesV])
	}
	weights, err := surfaceWeightNet(def)
	if err != nil {
		return nil, err
	}
	g, err := geom.NewBSplineSurface(def.DegreeU, def.DegreeV, ctrl, weights, def.KnotsU, def.KnotsV)
	if err != nil {
		return nil, err
	}
	return newBSplineSurface(g), nil
}

// surfaceWeightNet shapes the flat weight list into the kernel's rectangular
// net, defaulting an omitted list to uniform 1s.
func surfaceWeightNet(def types.BSplineSurfaceDef) ([][]float64, error) {
	weights := make([][]float64, def.PolesU)
	if def.Weights == nil {
		for i := range weights {
			row := make([]float64, def.PolesV)
			for j := range row {
				row[j] = 1
			}
			weights[i] = row
		}
		return weights, nil
	}
	if len(def.Weights) != def.PolesU*def.PolesV {
		return nil, fmt.Errorf("geomapi: surface weights %d do not fill the %d×%d net", len(def.Weights), def.PolesU, def.PolesV)
	}
	for i := 0; i < def.PolesU; i++ {
		weights[i] = append([]float64(nil), def.Weights[i*def.PolesV:(i+1)*def.PolesV]...)
	}
	return weights, nil
}
