// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The 2D (sketch-space) twin of evaluator_curve3.go (M01-F06, #603).

// CurveEndPoints2 returns the curve's end points, bounded=false when the
// domain is infinite.
func CurveEndPoints2(c Curve2) (start, end math.Point2, bounded bool) {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return math.Point2{}, math.Point2{}, false
	}
	return c.PointAt(lo), c.PointAt(hi), true
}

// CurveLength2 returns the arc length between the two parameters.
func CurveLength2(c Curve2, from, to float64) float64 {
	lo, hi := orderedInterval(from, to)
	switch g := c.(type) {
	case Line2d:
		return hi - lo
	case LineSegment2d:
		return (hi - lo) * g.Length()
	case Circle2d:
		return (hi - lo) * g.Circumference()
	case Arc2d:
		return (hi - lo) * g.Length()
	case Polyline2d:
		return polylineLengthBetween2(g, lo, hi)
	default:
		return integrateSpans(curveSpeed2(c), lo, hi, curveLengthBreaks2(c))
	}
}

// curveSpeed2 returns the arc-length integrand |dP/dt|.
func curveSpeed2(c Curve2) func(float64) float64 {
	return func(t float64) float64 {
		d1, _, _ := CurveDerivatives2(c, t)
		return d1.Length()
	}
}

// curveLengthBreaks2 returns the NURBS knot breakpoints, nil otherwise.
func curveLengthBreaks2(c Curve2) []float64 {
	if b, ok := c.(BSplineCurve2d); ok {
		return interiorKnots(b.Knots, b.Degree)
	}
	return nil
}

// polylineLengthBetween2 sums the segment lengths overlapped by [lo, hi].
func polylineLengthBetween2(p Polyline2d, lo, hi float64) float64 {
	segs := len(p.Vertices) - 1
	total := 0.0
	for i := range segs {
		s0, s1 := float64(i)/float64(segs), float64(i+1)/float64(segs)
		overlap := stdmath.Min(hi, s1) - stdmath.Max(lo, s0)
		if overlap > 0 {
			total += overlap * float64(segs) * p.Vertices[i].DistanceTo(p.Vertices[i+1])
		}
	}
	return total
}

// CurveParamAtLength2 returns the parameter at the signed arc length from the
// given parameter, clamped to the domain.
func CurveParamAtLength2(c Curve2, from, length float64) float64 {
	if length == 0 {
		return from
	}
	if t, ok := constantSpeedParam2(c, from, length); ok {
		return t
	}
	lo, hi := paramSearchRange(c.Domain, from, length)
	signed := func(t float64) float64 { return signedLength2(c, from, t) }
	return invertLength(signed, curveSpeed2(c), length, lo, hi)
}

// signedLength2 is CurveLength2 with the sign of the parameter direction.
func signedLength2(c Curve2, from, t float64) float64 {
	if t < from {
		return -CurveLength2(c, t, from)
	}
	return CurveLength2(c, from, t)
}

// constantSpeedParam2 inverts length in closed form for constant-speed curves.
func constantSpeedParam2(c Curve2, from, length float64) (float64, bool) {
	switch g := c.(type) {
	case Line2d:
		return from + length, true
	case LineSegment2d:
		return math.Clamp01(from + length/g.Length()), true
	case Circle2d:
		return from + length/g.Circumference(), true
	case Arc2d:
		return math.Clamp01(from + length/g.Length()), true
	default:
		return 0, false
	}
}

// CurveStrokes2 tessellates [from, to] within the chordal tolerance.
func CurveStrokes2(c Curve2, from, to, tolerance float64) []math.Point2 {
	lo, hi := orderedInterval(from, to)
	if p, ok := c.(Polyline2d); ok {
		return polylineStrokeVertices2(p, lo, hi)
	}
	pts := []math.Point2{c.PointAt(lo)}
	if lo == hi {
		return pts
	}
	quarter := (hi - lo) / 4 // see CurveStrokes3: break closed-curve symmetry
	for i := range 4 {
		a, b := lo+float64(i)*quarter, lo+float64(i+1)*quarter
		strokeRecurse2(c, a, b, c.PointAt(a), c.PointAt(b), tolerance, strokeMaxDepth, &pts)
	}
	return pts
}

// polylineStrokeVertices2 returns the exact vertices covered by [lo, hi].
func polylineStrokeVertices2(p Polyline2d, lo, hi float64) []math.Point2 {
	segs := len(p.Vertices) - 1
	pts := []math.Point2{p.PointAt(lo)}
	for i := 1; i <= segs; i++ {
		t := float64(i) / float64(segs)
		if t > lo && t < hi {
			pts = append(pts, p.Vertices[i])
		}
	}
	return append(pts, p.PointAt(hi))
}

// strokeRecurse2 subdivides until the curve midpoint is within tolerance of
// the chord.
func strokeRecurse2(c Curve2, a, b float64, pa, pb math.Point2, tol float64, depth int, pts *[]math.Point2) {
	m := (a + b) / 2
	pm := c.PointAt(m)
	if depth <= 0 || chordDeviation2(pa, pb, pm) <= tol {
		*pts = append(*pts, pb)
		return
	}
	strokeRecurse2(c, a, m, pa, pm, tol, depth-1, pts)
	strokeRecurse2(c, m, b, pm, pb, tol, depth-1, pts)
}

// chordDeviation2 returns the distance from p to the segment (a, b).
func chordDeviation2(a, b, p math.Point2) float64 {
	chord := a.VectorTo(b)
	den := float64(chord.LengthSquared())
	if den == 0 {
		return a.DistanceTo(p)
	}
	t := math.Clamp01(float64(a.VectorTo(p).Dot(chord)) / den)
	return a.TranslateBy(chord.Scale(t)).DistanceTo(p)
}

// CurveContinuity2 returns the largest maintained continuity order.
func CurveContinuity2(c Curve2) int {
	switch g := c.(type) {
	case Polyline2d:
		return 0
	case BSplineCurve2d:
		return bsplineContinuity(g.Knots, g.Degree)
	default:
		return ContinuityInfinite
	}
}

// CurveAnomaly2 reports the curve's parameter irregularities.
func CurveAnomaly2(c Curve2) ParamAnomaly {
	switch g := c.(type) {
	case Line2d:
		return ParamAnomaly{Unbounded: true}
	case Circle2d, EllipseFull2d:
		return ParamAnomaly{Periodic: true, Period: 1}
	case Polyline2d:
		return ParamAnomaly{Singular: len(g.Vertices) > 2}
	case BSplineCurve2d:
		return bsplineCurveAnomaly2(g)
	default:
		return ParamAnomaly{}
	}
}

// bsplineCurveAnomaly2 reports a geometrically closed NURBS curve as periodic.
func bsplineCurveAnomaly2(g BSplineCurve2d) ParamAnomaly {
	lo, hi := g.Domain()
	if g.PointAt(lo).DistanceTo(g.PointAt(hi)) <= math.DefaultTolerance {
		return ParamAnomaly{Periodic: true, Period: hi - lo}
	}
	return ParamAnomaly{}
}
