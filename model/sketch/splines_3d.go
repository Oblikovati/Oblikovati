// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/param"
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
}

// IsFitType reports whether the spline interpolates its points (fit) rather than
// approximating them (control); PointCount returns the number of defining points.
func (s *Spline3D) IsFitType() bool { return s.fit }
func (s *Spline3D) PointCount() int { return len(s.Points) }

// Sample returns the spline's representative polyline in model space.
func (s *Spline3D) Sample() []math.Point3 {
	pts := make([]math.Point3, len(s.Points))
	for i, p := range s.Points {
		pts[i] = p.Position()
	}
	return sampleChain3D(pts, s.Closed)
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
	return sampleChain3D(append([]math.Point3(nil), s.Pts...), s.Closed)
}

// EquationCurve3D is a parametric 3D curve x(t)/y(t)/z(t) over t ∈ [T0, T1].
type EquationCurve3D struct {
	entityBase
	XExpr, YExpr, ZExpr string
	T0, T1              float64
	xc, yc, zc          param.Expr
}

// At evaluates the curve at parameter t.
func (e *EquationCurve3D) At(t float64) math.Point3 {
	return math.P3(math.Scalar(evalT(e.xc, t)), math.Scalar(evalT(e.yc, t)), math.Scalar(evalT(e.zc, t)))
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
	xc, yc, zc, err := bindEquation3D(xExpr, yExpr, zExpr)
	if err != nil {
		return nil, err
	}
	e := &EquationCurve3D{
		entityBase: newEntity(), XExpr: xExpr, YExpr: yExpr, ZExpr: zExpr,
		T0: t0, T1: t1, xc: xc, yc: yc, zc: zc,
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
