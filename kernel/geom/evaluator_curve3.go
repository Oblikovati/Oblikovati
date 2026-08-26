// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The member-level 3D curve evaluator (M01-F06, #603): arc length and its
// inverse, chordal stroking, end points, continuity and parameter anomaly.
// Closed forms cover the analytic curves; NURBS and the helix integrate
// adaptively. The 2D twins live in evaluator_curve2.go.

// CurveEndPoints3 returns the curve's end points, bounded=false when the
// domain is infinite (a Line has no end points).
func CurveEndPoints3(c Curve3) (start, end math.Point3, bounded bool) {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return math.Point3{}, math.Point3{}, false
	}
	return c.PointAt(lo), c.PointAt(hi), true
}

// CurveLength3 returns the arc length between the two parameters (order
// agnostic): exact closed forms for constant-speed curves, piecewise sums for
// polylines, adaptive quadrature of |P′| otherwise.
func CurveLength3(c Curve3, from, to float64) float64 {
	lo, hi := orderedInterval(from, to)
	switch g := c.(type) {
	case Line:
		return hi - lo
	case LineSegment:
		return (hi - lo) * g.Length()
	case Circle:
		return (hi - lo) * g.Circumference()
	case Arc3d:
		return (hi - lo) * g.Length()
	case Polyline:
		return polylineLengthBetween3(g, lo, hi)
	default:
		return integrateSpans(curveSpeed3(c), lo, hi, curveLengthBreaks3(c))
	}
}

// curveSpeed3 returns the arc-length integrand |dP/dt|.
func curveSpeed3(c Curve3) func(float64) float64 {
	return func(t float64) float64 {
		d1, _, _ := CurveDerivatives3(c, t)
		return d1.Length()
	}
}

// curveLengthBreaks3 returns the quadrature breakpoints: NURBS speed has kinks
// at interior knots; everything else is smooth.
func curveLengthBreaks3(c Curve3) []float64 {
	if b, ok := c.(BSplineCurve); ok {
		return interiorKnots(b.Knots, b.Degree)
	}
	return nil
}

// polylineLengthBetween3 sums the segment lengths overlapped by [lo, hi] in
// the polyline's uniform parameterization.
func polylineLengthBetween3(p Polyline, lo, hi float64) float64 {
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

// CurveParamAtLength3 returns the parameter at the signed arc length from the
// given parameter (the inverse of [CurveLength3]), clamped to the domain.
func CurveParamAtLength3(c Curve3, from, length float64) float64 {
	if length == 0 {
		return from
	}
	if t, ok := constantSpeedParam3(c, from, length); ok {
		return t
	}
	lo, hi := paramSearchRange(c.Domain, from, length)
	signed := func(t float64) float64 { return signedLength3(c, from, t) }
	return invertLength(signed, curveSpeed3(c), length, lo, hi)
}

// signedLength3 is CurveLength3 with the sign of the parameter direction.
func signedLength3(c Curve3, from, t float64) float64 {
	if t < from {
		return -CurveLength3(c, t, from)
	}
	return CurveLength3(c, from, t)
}

// constantSpeedParam3 inverts length in closed form for constant-speed curves.
func constantSpeedParam3(c Curve3, from, length float64) (float64, bool) {
	switch g := c.(type) {
	case Line:
		return from + length, true
	case LineSegment:
		return math.Clamp01(from + length/g.Length()), true
	case Circle:
		return from + length/g.Circumference(), true
	case Arc3d:
		return math.Clamp01(from + length/g.Length()), true
	default:
		return 0, false
	}
}

// paramSearchRange bounds the inverse-length search: the domain when finite,
// otherwise a window from the start parameter wide enough to hold the target
// (unbounded curves reach any length within |length| of parameter, since the
// only unbounded kernel curve is unit-speed; the 2× margin absorbs others).
func paramSearchRange(domain func() (float64, float64), from, length float64) (lo, hi float64) {
	lo, hi = domain()
	if length > 0 {
		lo = from
		if stdmath.IsInf(hi, 0) {
			hi = from + 2*length
		}
		return lo, hi
	}
	hi = from
	if stdmath.IsInf(lo, 0) {
		lo = from + 2*length
	}
	return lo, hi
}

// CurveStrokes3 tessellates [from, to] into a polyline whose chordal deviation
// from the curve stays within tolerance. A polyline returns its own vertices.
func CurveStrokes3(c Curve3, from, to, tolerance float64) []math.Point3 {
	lo, hi := orderedInterval(from, to)
	if p, ok := c.(Polyline); ok {
		return polylineStrokeVertices3(p, lo, hi)
	}
	pts := []math.Point3{c.PointAt(lo)}
	if lo == hi {
		return pts
	}
	// Four initial slices break the symmetry of closed curves, whose full-range
	// chord midpoint can coincide with the curve (a zero-length chord on a circle).
	quarter := (hi - lo) / 4
	for i := range 4 {
		a, b := lo+float64(i)*quarter, lo+float64(i+1)*quarter
		strokeRecurse3(c, a, b, c.PointAt(a), c.PointAt(b), tolerance, strokeMaxDepth, &pts)
	}
	return pts
}

// polylineStrokeVertices3 returns the exact vertices covered by [lo, hi],
// including the clamped range ends.
func polylineStrokeVertices3(p Polyline, lo, hi float64) []math.Point3 {
	segs := len(p.Vertices) - 1
	pts := []math.Point3{p.PointAt(lo)}
	for i := 1; i <= segs; i++ {
		t := float64(i) / float64(segs)
		if t > lo && t < hi {
			pts = append(pts, p.Vertices[i])
		}
	}
	return append(pts, p.PointAt(hi))
}

// strokeRecurse3 subdivides [a, b] until the curve midpoint sits within
// tolerance of the chord, then emits the right end.
func strokeRecurse3(c Curve3, a, b float64, pa, pb math.Point3, tol float64, depth int, pts *[]math.Point3) {
	m := (a + b) / 2
	pm := c.PointAt(m)
	if depth <= 0 || chordDeviation3(pa, pb, pm) <= tol {
		*pts = append(*pts, pb)
		return
	}
	strokeRecurse3(c, a, m, pa, pm, tol, depth-1, pts)
	strokeRecurse3(c, m, b, pm, pb, tol, depth-1, pts)
}

// chordDeviation3 returns the distance from p to the segment (a, b).
func chordDeviation3(a, b, p math.Point3) float64 {
	chord := a.VectorTo(b)
	den := float64(chord.LengthSquared())
	if den == 0 {
		return a.DistanceTo(p)
	}
	t := math.Clamp01(float64(a.VectorTo(p).Dot(chord)) / den)
	return a.TranslateBy(chord.Scale(t)).DistanceTo(p)
}

// CurveContinuity3 returns the largest maintained continuity order: 0 for a
// polyline, degree − max interior knot multiplicity for NURBS (C∞ when there
// are no interior knots), [ContinuityInfinite] for analytic curves.
func CurveContinuity3(c Curve3) int {
	switch g := c.(type) {
	case Polyline:
		return 0
	case BSplineCurve:
		return bsplineContinuity(g.Knots, g.Degree)
	default:
		return ContinuityInfinite
	}
}

// bsplineContinuity returns degree − max interior multiplicity, or C∞ for a
// knot vector with no interior knots (a single polynomial span).
func bsplineContinuity(knots []float64, degree int) int {
	if len(interiorKnots(knots, degree)) == 0 {
		return ContinuityInfinite
	}
	return degree - maxInteriorMultiplicity(knots, degree)
}

// CurveAnomaly3 reports the curve's parameter irregularities.
func CurveAnomaly3(c Curve3) ParamAnomaly {
	switch g := c.(type) {
	case Line:
		return ParamAnomaly{Unbounded: true}
	case Circle, EllipseFull:
		return ParamAnomaly{Periodic: true, Period: 1}
	case Polyline:
		return ParamAnomaly{Singular: len(g.Vertices) > 2}
	case BSplineCurve:
		return bsplineCurveAnomaly3(g)
	default:
		return ParamAnomaly{}
	}
}

// bsplineCurveAnomaly3 reports a closed NURBS curve as periodic over its
// domain span (the geometric closure test, not knot-vector structure).
func bsplineCurveAnomaly3(g BSplineCurve) ParamAnomaly {
	lo, hi := g.Domain()
	if g.PointAt(lo).DistanceTo(g.PointAt(hi)) <= math.DefaultTolerance {
		return ParamAnomaly{Periodic: true, Period: hi - lo}
	}
	return ParamAnomaly{}
}
