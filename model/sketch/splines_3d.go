// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// This file holds the 3D-sketch spline family (M22-F03): interpolation/control-point
// splines through constrainable points, an immutable fixed spline, and a parametric
// equation curve. Splines are sampled with the shared 3D Catmull–Rom
// (curvesample_spline_3d.go); their defining points are solver DOFs.

// Spline3D is a 3D spline through (fit) or over (control) its defining points. fit reports
// which; both are sampled by the representative Catmull–Rom polyline. Its points are
// solver variables, so a spline can be dimensioned/constrained like any 3D geometry.
type Spline3D struct {
	entityBase
	Points []*Point3D
	Closed bool
	fit    bool
	// handles are the active tangency handles keyed by fit-point index
	// (M06-F11; see spline_handles_3d.go).
	handles map[int]*SplineHandle3D
}

// IsFitType reports whether the spline interpolates its points (fit) rather than
// approximating them (control); PointCount returns the number of defining points.
func (s *Spline3D) IsFitType() bool { return s.fit }
func (s *Spline3D) PointCount() int { return len(s.Points) }

// Sample returns the spline's representative polyline in model space.
func (s *Spline3D) Sample() []math.Point3 {
	if len(s.handles) > 0 && s.fit {
		return sampleHandledSpline3D(s, splineSamplesPerSpan)
	}
	pts := make([]math.Point3, len(s.Points))
	for i, p := range s.Points {
		pts[i] = p.Position()
	}
	return sampleChain3D(pts, s.Closed, s.fit)
}

// FixedSpline3D is an immutable 3D spline through a stored coordinate list (a derived /
// imported curve). Its points are not solver variables.
type FixedSpline3D struct {
	entityBase
	Pts    []math.Point3
	Closed bool
}

// Sample returns the fixed spline's representative polyline.
func (s *FixedSpline3D) Sample() []math.Point3 {
	return sampleChain3D(append([]math.Point3(nil), s.Pts...), s.Closed, true)
}

// EquationCurve3D is a parametric 3D curve over t ∈ [T0, T1]. The three expressions are
// interpreted by Coord (#1846): Cartesian x/y/z, cylindrical radius/theta/z, or spherical
// radius/theta/phi.
type EquationCurve3D struct {
	entityBase
	XExpr, YExpr, ZExpr string
	T0, T1              float64
	coord               types.CoordinateSystemType
	xc, yc, zc          param.Expr
}

// CoordinateSystem reports how the three expressions are interpreted (#1846).
func (e *EquationCurve3D) CoordinateSystem() types.CoordinateSystemType { return e.coord }

// At evaluates the curve at parameter t, mapping the three expression values to a Cartesian
// point through the curve's coordinate system.
func (e *EquationCurve3D) At(t float64) math.Point3 {
	return equationCurvePoint(e.coord, evalT(e.xc, t), evalT(e.yc, t), evalT(e.zc, t))
}

// equationCurvePoint converts the three evaluated expressions to Cartesian per coordinate system
// (#1846): cylindrical (r,θ,z)→(r·cosθ, r·sinθ, z); spherical (r,θ,φ)→(r·sinφ·cosθ, r·sinφ·sinθ,
// r·cosφ). θ/φ are radians, r/z centimetres.
func equationCurvePoint(coord types.CoordinateSystemType, a, b, c float64) math.Point3 {
	switch coord {
	case types.CoordinateSystemCylindrical:
		return math.P3(math.Scalar(a*stdmath.Cos(b)), math.Scalar(a*stdmath.Sin(b)), math.Scalar(c))
	case types.CoordinateSystemSpherical:
		return math.P3(
			math.Scalar(a*stdmath.Sin(c)*stdmath.Cos(b)),
			math.Scalar(a*stdmath.Sin(c)*stdmath.Sin(b)),
			math.Scalar(a*stdmath.Cos(c)),
		)
	default:
		return math.P3(math.Scalar(a), math.Scalar(b), math.Scalar(c))
	}
}

// Sample returns n+1 evenly-spaced points along the curve.
func (e *EquationCurve3D) Sample(n int) []math.Point3 {
	pts := make([]math.Point3, n+1)
	for i := range pts {
		t := e.T0 + (e.T1-e.T0)*float64(i)/float64(n)
		pts[i] = e.At(t)
	}
	return pts
}

// AddSpline3D adds an interpolation or control-point spline through new points at the
// given positions (fit=true interpolates, fit=false treats them as a control polygon).
func (s *Sketch3D) AddSpline3D(positions []math.Point3, closed, fit bool) *Spline3D {
	pts := make([]*Point3D, len(positions))
	for i, p := range positions {
		pts[i] = s.newPoint3D(p)
	}
	return s.addSpline3DPts(pts, closed, fit)
}

// addSpline3DPts builds a spline over existing points (the restore seam).
func (s *Sketch3D) addSpline3DPts(pts []*Point3D, closed, fit bool) *Spline3D {
	sp := &Spline3D{entityBase: newEntity(), Points: pts, Closed: closed, fit: fit}
	s.addEntity3D(sp)
	return sp
}

// AddFixedSpline3D adds an immutable spline through the given coordinates.
func (s *Sketch3D) AddFixedSpline3D(coords []math.Point3, closed bool) *FixedSpline3D {
	sp := &FixedSpline3D{entityBase: newEntity(), Pts: append([]math.Point3(nil), coords...), Closed: closed}
	s.addEntity3D(sp)
	return sp
}

// AddEquationCurve3D adds a parametric curve from x(t)/y(t)/z(t) expressions over [t0,t1],
// erroring if any expression fails to parse/bind to t.
func (s *Sketch3D) AddEquationCurve3D(xExpr, yExpr, zExpr string, t0, t1 float64) (*EquationCurve3D, error) {
	return s.AddEquationCurve3DIn(types.CoordinateSystemCartesian, xExpr, yExpr, zExpr, t0, t1)
}

// AddEquationCurve3DIn adds a parametric curve whose three expressions are interpreted in the given
// coordinate system (#1846); the Cartesian case is [Sketch3D.AddEquationCurve3D].
func (s *Sketch3D) AddEquationCurve3DIn(coord types.CoordinateSystemType, xExpr, yExpr, zExpr string, t0, t1 float64) (*EquationCurve3D, error) {
	xc, yc, zc, err := bindEquation3D(xExpr, yExpr, zExpr)
	if err != nil {
		return nil, err
	}
	e := &EquationCurve3D{
		entityBase: newEntity(), XExpr: xExpr, YExpr: yExpr, ZExpr: zExpr,
		T0: t0, T1: t1, coord: coord, xc: xc, yc: yc, zc: zc,
	}
	s.addEntity3D(e)
	return e, nil
}

// bindEquation3D parses and binds the three coordinate expressions to the parameter t.
func bindEquation3D(xExpr, yExpr, zExpr string) (xc, yc, zc param.Expr, err error) {
	if xc, err = bindT(xExpr); err != nil {
		return
	}
	if yc, err = bindT(yExpr); err != nil {
		return
	}
	zc, err = bindT(zExpr)
	return
}
