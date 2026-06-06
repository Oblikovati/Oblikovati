// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati/math"
	"oblikovati/model/param"
)

// tParamID is the synthetic parameter id the equation-curve parameter "t" binds to.
const tParamID param.ID = 1

// EquationCurve is a parametric curve x(t), y(t) driven by the expression engine over
// t ∈ [T0, T1] (t is unitless; the evaluated x/y are sketch-plane cm). Immutable — it is a
// reference curve, not a constrainable entity.
type EquationCurve struct {
	entityBase
	XExpr, YExpr string
	T0, T1       float64
	xc, yc       param.Expr
}

// EquationCurves creates and tracks equation curves.
type EquationCurves struct {
	s     *Sketch
	items []*EquationCurve
}

// Add parses x(t)/y(t) and registers a parametric curve over [t0, t1]. It errors on a
// bad expression or an empty t-range.
func (c *EquationCurves) Add(xExpr, yExpr string, t0, t1 float64) (*EquationCurve, error) {
	if t0 == t1 {
		return nil, fmt.Errorf("equation curve: empty t-range [%g, %g]", t0, t1)
	}
	xc, err := bindT(xExpr)
	if err != nil {
		return nil, fmt.Errorf("equation curve x(t) %q: %w", xExpr, err)
	}
	yc, err := bindT(yExpr)
	if err != nil {
		return nil, fmt.Errorf("equation curve y(t) %q: %w", yExpr, err)
	}
	e := &EquationCurve{entityBase: newEntity(), XExpr: xExpr, YExpr: yExpr, T0: t0, T1: t1, xc: xc, yc: yc}
	c.s.add(e)
	c.items = append(c.items, e)
	return e, nil
}

// Count returns the number of equation curves; Item returns the i-th.
func (c *EquationCurves) Count() int                { return len(c.items) }
func (c *EquationCurves) Item(i int) *EquationCurve { return c.items[i] }

// At evaluates the curve at parameter t.
func (e *EquationCurve) At(t float64) math.Point2 {
	return math.P2(math.Scalar(evalT(e.xc, t)), math.Scalar(evalT(e.yc, t)))
}

// Sample returns n+1 evenly-spaced points along the curve.
func (e *EquationCurve) Sample(n int) []math.Point2 {
	pts := make([]math.Point2, n+1)
	for i := range pts {
		t := e.T0 + (e.T1-e.T0)*float64(i)/float64(n)
		pts[i] = e.At(t)
	}
	return pts
}

// bindT parses an expression and binds its sole variable "t" to tParamID.
func bindT(src string) (param.Expr, error) {
	expr, err := param.Parse(src)
	if err != nil {
		return param.Expr{}, err
	}
	unresolved := expr.Bind(func(name string) (param.ID, bool) {
		if name == "t" {
			return tParamID, true
		}
		return 0, false
	})
	if len(unresolved) > 0 {
		return param.Expr{}, fmt.Errorf("unknown variable(s) %v (only t is allowed)", unresolved)
	}
	return expr, nil
}

// evalT evaluates a t-bound expression at the given t (0 on error — equation curves are
// best-effort reference geometry).
func evalT(expr param.Expr, t float64) float64 {
	q, err := expr.Eval(tScope{t: t})
	if err != nil {
		return 0
	}
	return q.Value
}

// tScope feeds the current t value to the expression evaluator.
type tScope struct{ t float64 }

func (s tScope) ValueOf(id param.ID) (param.Quantity, bool) {
	if id == tParamID {
		return param.Quantity{Value: s.t, Unit: param.Unitless}, true
	}
	return param.Quantity{}, false
}

// FixedSpline is an immutable spline through a fixed set of sample points (e.g. derived
// from projected/included geometry); its points are not solver variables.
type FixedSpline struct {
	entityBase
	Pts []math.Point2
}

// FixedSplines creates and tracks fixed splines.
type FixedSplines struct {
	s     *Sketch
	items []*FixedSpline
}

// Add registers a fixed spline through the given points.
func (c *FixedSplines) Add(pts []math.Point2) *FixedSpline {
	f := &FixedSpline{entityBase: newEntity(), Pts: append([]math.Point2(nil), pts...)}
	c.s.add(f)
	c.items = append(c.items, f)
	return f
}

// Count returns the number of fixed splines; Item returns the i-th.
func (c *FixedSplines) Count() int              { return len(c.items) }
func (c *FixedSplines) Item(i int) *FixedSpline { return c.items[i] }

// OffsetSpline is the offset of a parent spline by a signed distance (left of the parent's
// direction for a positive distance). Immutable; tracks the parent by reference.
type OffsetSpline struct {
	entityBase
	Parent *Spline
	Dist   float64
}

// OffsetSplines creates and tracks offset splines.
type OffsetSplines struct {
	s     *Sketch
	items []*OffsetSpline
}

// Add registers an offset of parent at signed distance dist.
func (c *OffsetSplines) Add(parent *Spline, dist float64) *OffsetSpline {
	o := &OffsetSpline{entityBase: newEntity(), Parent: parent, Dist: dist}
	c.s.add(o)
	c.items = append(c.items, o)
	return o
}

// Count returns the number of offset splines; Item returns the i-th.
func (c *OffsetSplines) Count() int               { return len(c.items) }
func (c *OffsetSplines) Item(i int) *OffsetSpline { return c.items[i] }

// Sample returns the parent's sampled polyline offset by Dist along its left normal.
func (o *OffsetSpline) Sample() []math.Point2 {
	return offsetPolyline(sampleSplineEntity(o.Parent), o.Dist)
}
