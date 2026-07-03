// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The member-level evaluator adapters (M01-F06, #603): thin views over the
// kernel evaluator engine, speaking the contract's value types. Like the curve
// adapters they carry no state beyond the kernel geometry.

// curveEvaluator3 implements contract.CurveEvaluator over a kernel curve.
type curveEvaluator3 struct {
	c geom.Curve3
}

var _ contract.CurveEvaluator = curveEvaluator3{}

// Evaluator returns the member-level query surface of this curve.
func (c curve3) Evaluator() contract.CurveEvaluator { return curveEvaluator3{c: c.inner} }

func (e curveEvaluator3) RangeBox() types.Box {
	return toBox(geom.CurveRangeBox3(e.c))
}

func (e curveEvaluator3) Continuity() int { return geom.CurveContinuity3(e.c) }

func (e curveEvaluator3) EndPoints() (start, end types.Point, bounded bool) {
	s, en, ok := geom.CurveEndPoints3(e.c)
	return toPoint(s), toPoint(en), ok
}

func (e curveEvaluator3) ParamExtents() (lo, hi float64) { return e.c.Domain() }

func (e curveEvaluator3) Derivatives(t float64) (d1, d2, d3 types.Vector) {
	k1, k2, k3 := geom.CurveDerivatives3(e.c, t)
	return toVector(k1), toVector(k2), toVector(k3)
}

func (e curveEvaluator3) Curvature(t float64) (types.Vector, float64) {
	dir, k := geom.CurveCurvature3(e.c, t)
	return toVector(dir), k
}

func (e curveEvaluator3) Length(from, to float64) float64 {
	return geom.CurveLength3(e.c, from, to)
}

func (e curveEvaluator3) ParamAtLength(from, length float64) float64 {
	return geom.CurveParamAtLength3(e.c, from, length)
}

func (e curveEvaluator3) Strokes(from, to, tolerance float64) []types.Point {
	return toPoints(geom.CurveStrokes3(e.c, from, to, tolerance))
}

func (e curveEvaluator3) ParamAtPoint(p types.Point) (float64, types.SolutionNature) {
	return geom.CurveParamAtPoint3(e.c, fromPoint(p))
}

func (e curveEvaluator3) ParamAnomaly() types.ParamAnomaly { return geom.CurveAnomaly3(e.c) }

// curveEvaluator2 implements contract.Curve2dEvaluator over a kernel curve.
type curveEvaluator2 struct {
	c geom.Curve2
}

var _ contract.Curve2dEvaluator = curveEvaluator2{}

// Evaluator returns the member-level query surface of this curve.
func (c curve2) Evaluator() contract.Curve2dEvaluator { return curveEvaluator2{c: c.inner} }

func (e curveEvaluator2) RangeBox() types.Box2d {
	return toBox2d(geom.CurveRangeBox2(e.c))
}

func (e curveEvaluator2) Continuity() int { return geom.CurveContinuity2(e.c) }

func (e curveEvaluator2) EndPoints() (start, end types.Point2d, bounded bool) {
	s, en, ok := geom.CurveEndPoints2(e.c)
	return toPoint2d(s), toPoint2d(en), ok
}

func (e curveEvaluator2) ParamExtents() (lo, hi float64) { return e.c.Domain() }

func (e curveEvaluator2) Derivatives(t float64) (d1, d2, d3 types.Vector2d) {
	k1, k2, k3 := geom.CurveDerivatives2(e.c, t)
	return toVector2d(k1), toVector2d(k2), toVector2d(k3)
}

func (e curveEvaluator2) Curvature(t float64) float64 { return geom.CurveCurvature2(e.c, t) }

func (e curveEvaluator2) Length(from, to float64) float64 {
	return geom.CurveLength2(e.c, from, to)
}

func (e curveEvaluator2) ParamAtLength(from, length float64) float64 {
	return geom.CurveParamAtLength2(e.c, from, length)
}

func (e curveEvaluator2) Strokes(from, to, tolerance float64) []types.Point2d {
	return toPoints2d(geom.CurveStrokes2(e.c, from, to, tolerance))
}

func (e curveEvaluator2) ParamAtPoint(p types.Point2d) (float64, types.SolutionNature) {
	return geom.CurveParamAtPoint2(e.c, fromPoint2d(p))
}

func (e curveEvaluator2) ParamAnomaly() types.ParamAnomaly { return geom.CurveAnomaly2(e.c) }

// surfaceEvaluator implements contract.SurfaceEvaluator over a kernel surface.
type surfaceEvaluator struct {
	s geom.Surface
}

var _ contract.SurfaceEvaluator = surfaceEvaluator{}

// The capability families SurfaceEvaluator embeds (I9, api v0.104.1). surfaceEvaluator
// satisfies each as part of the union; the explicit assertions let a caller depend on
// the narrowest slice (extents-only, differential-only, …) and satisfy the #1619 guard.
var (
	_ contract.SurfaceExtents      = surfaceEvaluator{}
	_ contract.SurfaceDifferential = surfaceEvaluator{}
	_ contract.SurfaceProjection   = surfaceEvaluator{}
	_ contract.SurfaceIso          = surfaceEvaluator{}
)

// Evaluator returns the member-level query surface of this surface.
func (s surface) Evaluator() contract.SurfaceEvaluator { return surfaceEvaluator{s: s.inner} }

func (e surfaceEvaluator) RangeBox() types.Box {
	return toBox(geom.SurfaceRangeBox(e.s))
}

func (e surfaceEvaluator) ParamRangeRect() types.Box2d {
	uLo, uHi := e.s.UDomain()
	vLo, vHi := e.s.VDomain()
	return types.Box2d{Min: types.NewPoint2d(uLo, vLo), Max: types.NewPoint2d(uHi, vHi)}
}

func (e surfaceEvaluator) Area() float64 { return geom.SurfaceArea(e.s) }

func (e surfaceEvaluator) Continuity() int { return geom.SurfaceContinuity(e.s) }

func (e surfaceEvaluator) Tangents(u, v float64) (uTangent, vTangent types.Vector) {
	ut, vt := geom.SurfaceTangents(e.s, u, v)
	return toVector(ut), toVector(vt)
}

func (e surfaceEvaluator) FirstPartials(u, v float64) (pu, pv types.Vector) {
	du, dv := e.s.DerivativesAt(u, v)
	return toVector(du), toVector(dv)
}

func (e surfaceEvaluator) SecondPartials(u, v float64) (puu, puv, pvv types.Vector) {
	kuu, kuv, kvv := geom.SurfaceSecondPartials(e.s, u, v)
	return toVector(kuu), toVector(kuv), toVector(kvv)
}

func (e surfaceEvaluator) ThirdPartials(u, v float64) (puuu, pvvv types.Vector) {
	kuuu, kvvv := geom.SurfaceThirdPartials(e.s, u, v)
	return toVector(kuuu), toVector(kvvv)
}

func (e surfaceEvaluator) Curvatures(u, v float64) (maxDirection types.Vector, maxCurvature, minCurvature float64) {
	dir, kMax, kMin := geom.SurfaceCurvatures(e.s, u, v)
	return toVector(dir), kMax, kMin
}

func (e surfaceEvaluator) ParamAtPoint(p types.Point) (u, v float64, nature types.SolutionNature) {
	return geom.SurfaceParamAtPoint(e.s, fromPoint(p))
}

func (e surfaceEvaluator) NormalAtPoint(p types.Point) types.Vector {
	return toVector(geom.SurfaceNormalAtPoint(e.s, fromPoint(p)))
}

func (e surfaceEvaluator) IsoCurve(uDirection bool, param float64) (contract.Curve, error) {
	iso, err := geom.SurfaceIsoCurve(e.s, uDirection, param)
	if err != nil {
		return nil, err
	}
	return wrapCurve3(iso)
}

func (e surfaceEvaluator) ParamAnomaly() (u, v types.ParamAnomaly) {
	return geom.SurfaceAnomaly(e.s)
}

// wrapCurve3 adapts a kernel curve back into its contract adapter — the
// inverse of the factory constructors, used by iso-curve extraction.
func wrapCurve3(c geom.Curve3) (contract.Curve, error) {
	switch g := c.(type) {
	case geom.Line:
		return newLine(g), nil
	case geom.LineSegment:
		return newSegment(g), nil
	case geom.Polyline:
		return newPolyline(g), nil
	case geom.BSplineCurve:
		return newBSpline(g), nil
	case geom.Helix3d:
		return newHelix(g), nil
	default:
		return wrapConic3(c)
	}
}

// wrapConic3 adapts the circular/elliptical kernel curves.
func wrapConic3(c geom.Curve3) (contract.Curve, error) {
	switch g := c.(type) {
	case geom.Circle:
		return newCircle(g), nil
	case geom.Arc3d:
		return newArc(g), nil
	case geom.EllipseFull:
		return newEllipse(g), nil
	case geom.EllipticalArc:
		return newEllipticalArc(g), nil
	default:
		return nil, fmt.Errorf("geomapi: no contract adapter for kernel curve type %T", c)
	}
}

// toBox converts a kernel box to the contract box.
func toBox(b math.Box) types.Box {
	return types.Box{Min: toPoint(b.Min), Max: toPoint(b.Max)}
}

// toBox2d converts a kernel 2D box to the contract box.
func toBox2d(b math.Box2d) types.Box2d {
	return types.Box2d{Min: toPoint2d(b.Min), Max: toPoint2d(b.Max)}
}
