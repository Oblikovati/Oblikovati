// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

// Evaluators are the numeric services snapping, measurement, and downstream
// features rely on: points, tangents, normals, curvature, length/area, and
// closest-point/parameter on model topology (PBI-081). Analytic queries delegate to
// the underlying geom evaluators (exact); curvature/length/closest are numeric.

// CurveEvaluator evaluates a 3D curve.
type CurveEvaluator struct {
	curve geom.Curve3
}

// NewCurveEvaluator wraps a curve.
func NewCurveEvaluator(c geom.Curve3) CurveEvaluator { return CurveEvaluator{curve: c} }

// PointAt returns the position at parameter t.
func (e CurveEvaluator) PointAt(t float64) math.Point3 { return e.curve.PointAt(t) }

// TangentAt returns the unit tangent at t (zero vector at a degenerate point).
func (e CurveEvaluator) TangentAt(t float64) math.Vector3 { return unit(e.curve.TangentAt(t)) }

// CurvatureAt returns κ = |r′×r″| / |r′|³ at t, with r″ by central difference.
func (e CurveEvaluator) CurvatureAt(t float64) float64 {
	const h = 1e-5
	r1 := e.curve.TangentAt(t)
	r2 := e.curve.TangentAt(t + h).Sub(e.curve.TangentAt(t - h)).Scale(1 / (2 * h))
	speed := r1.Length()
	if speed < math.DefaultTolerance {
		return 0
	}
	return r1.Cross(r2).Length() / (speed * speed * speed)
}

// Length returns the arc length over [lo, hi] by composite Simpson integration of
// the tangent speed. It requires a bounded interval.
func (e CurveEvaluator) Length(lo, hi float64) float64 {
	const n = 200 // even
	h := (hi - lo) / n
	speed := func(t float64) float64 { return e.curve.TangentAt(t).Length() }
	sum := speed(lo) + speed(hi)
	for i := 1; i < n; i++ {
		w := 2.0
		if i%2 == 1 {
			w = 4.0
		}
		sum += w * speed(lo+float64(i)*h)
	}
	return sum * h / 3
}

// ClosestParam returns the parameter in [lo, hi] whose point is nearest p, by a
// coarse sample then golden-section refinement of the best bracket.
func (e CurveEvaluator) ClosestParam(p math.Point3, lo, hi float64) float64 {
	const samples = 64
	best, bestD := lo, stdmath.Inf(1)
	step := (hi - lo) / samples
	for i := 0; i <= samples; i++ {
		t := lo + float64(i)*step
		if d := e.curve.PointAt(t).DistanceSquaredTo(p); d < bestD {
			best, bestD = t, d
		}
	}
	return refineMin(func(t float64) float64 { return e.curve.PointAt(t).DistanceSquaredTo(p) },
		stdmath.Max(lo, best-step), stdmath.Min(hi, best+step))
}

// EdgeEvaluator evaluates an edge over its bounded parameter span.
type EdgeEvaluator struct {
	CurveEvaluator
	lo, hi float64
}

// NewEdgeEvaluator wraps an edge with its curve domain.
func NewEdgeEvaluator(e *Edge) EdgeEvaluator {
	lo, hi := e.curve.Domain()
	return EdgeEvaluator{CurveEvaluator: NewCurveEvaluator(e.curve), lo: lo, hi: hi}
}

// Length returns the edge's arc length over its span.
func (ev EdgeEvaluator) Length() float64 { return ev.CurveEvaluator.Length(ev.lo, ev.hi) }

// ClosestParam returns the parameter nearest p within the edge's span.
func (ev EdgeEvaluator) ClosestParam(p math.Point3) float64 {
	return ev.CurveEvaluator.ClosestParam(p, ev.lo, ev.hi)
}

// ClosestPoint returns the point on the edge nearest p.
func (ev EdgeEvaluator) ClosestPoint(p math.Point3) math.Point3 {
	return ev.PointAt(ev.ClosestParam(p))
}

// unit normalizes a vector, returning the zero vector if it is degenerate.
func unit(v math.Vector3) math.Vector3 {
	if l := v.Length(); l > math.DefaultTolerance {
		return v.Scale(1 / l)
	}
	return math.V3(0, 0, 0)
}

// refineMin returns the argument minimizing f on [a, b] by golden-section search.
func refineMin(f func(float64) float64, a, b float64) float64 {
	const gr = 0.6180339887498949
	c := b - gr*(b-a)
	d := a + gr*(b-a)
	for i := 0; i < 60; i++ {
		if f(c) < f(d) {
			b, d = d, c
			c = b - gr*(b-a)
		} else {
			a, c = c, d
			d = a + gr*(b-a)
		}
	}
	return (a + b) / 2
}
