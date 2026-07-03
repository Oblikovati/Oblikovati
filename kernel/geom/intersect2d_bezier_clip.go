// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Bézier-clipping intersection of a 2D B-spline curve with a straight support line (Oblikovati#1608,
// audit A12). The retired 256-sample sign-change bracketing (intersect2d_curve.go) saw only
// TRANSVERSAL crossings: an even-multiplicity contact — a curve tangent to the line — touches the
// signed field's zero without changing sign, so no bracket ever formed and the contact was invisible
// in trim/offset paths. Bézier clipping is certified. On each Bézier segment the signed line distance
// is a scalar Bézier N(t) = Σ wᵢ·f(Pᵢ)·Bᵢ(t): because f is affine and the rational weights sum to a
// strictly positive denominator, sign(f(C(t))) = sign(N(t)) with the SAME root multiplicities. The
// convex-hull property then isolates every root — a sub-interval whose control coefficients are all
// one sign holds no root (discard); one straddling zero is subdivided until it isolates the single or
// tangential root. Sederberg & Nishita, "Curve intersection using Bézier clipping" (CAD 1990).

// bezierRootIters bounds the convex-hull subdivision depth: 60 halvings shrink a sub-interval below
// 1e-18 of a Bézier segment — past float64 resolution — mirroring bisectCurveZero's iteration floor.
const bezierRootIters = 60

// bezierSeg2d is one Bézier piece of a decomposed B-spline: its degree+1 control points and weights,
// and the [t0, t1] sub-domain of the parent curve it spans (so a local root maps back to the curve).
type bezierSeg2d struct {
	ctrl   []math.Point2
	w      []float64
	t0, t1 float64
}

// lineBSpline2dIntersection returns every point where the affine field f (a line/segment side) meets
// the B-spline c, including even-multiplicity tangential contacts, via per-segment Bézier clipping.
// Acceptance is model-relative (ADR-0042): a candidate is a contact only if it lands within the
// on-line tolerance, and duplicates within the weld tolerance (a root on a segment seam) merge.
func lineBSpline2dIntersection(f func(math.Point2) float64, c BSplineCurve2d) []math.Point2 {
	res := ResolutionForPoints2D(c.Ctrl)
	var pts []math.Point2
	for _, seg := range bezierSegments2d(c) {
		for _, s := range bezierSegmentRoots(seg, f) {
			p := c.PointAt(seg.t0 + s*(seg.t1-seg.t0))
			if stdmath.Abs(f(p)) <= res.Plane() && !containsPoint2(pts, p, res.Weld()) {
				pts = append(pts, p)
			}
		}
	}
	return pts
}

// bezierSegmentRoots isolates the local [0,1] roots of the affine field over one Bézier segment: the
// scalar field-Bézier coefficients are dᵢ = wᵢ·f(Pᵢ) (see file header), clipped by the convex hull.
func bezierSegmentRoots(seg bezierSeg2d, f func(math.Point2) float64) []float64 {
	d := make([]float64, len(seg.ctrl))
	for i, p := range seg.ctrl {
		d[i] = seg.w[i] * f(p)
	}
	var roots []float64
	clipBezierRoots(d, 0, 1, bezierRootIters, &roots)
	return roots
}

// clipBezierRoots appends every root of the scalar Bézier (control coefficients d) in [lo,hi]. A hull
// that does not straddle zero holds no root; otherwise the segment is subdivided (de Casteljau) and
// each half recursed, so an even-multiplicity touch is isolated exactly like a transversal crossing.
func clipBezierRoots(d []float64, lo, hi float64, iter int, out *[]float64) {
	if !hullStraddlesZero(d) {
		return
	}
	if iter <= 0 {
		*out = append(*out, (lo+hi)/2)
		return
	}
	left, right := splitBezierHalf(d)
	mid := (lo + hi) / 2
	clipBezierRoots(left, lo, mid, iter-1, out)
	clipBezierRoots(right, mid, hi, iter-1, out)
}

// hullStraddlesZero reports whether the Bézier control coefficients bracket zero — the convex-hull
// necessary condition for a root in the interval (variation-diminishing property).
func hullStraddlesZero(d []float64) bool {
	lo, hi := d[0], d[0]
	for _, v := range d[1:] {
		lo, hi = stdmath.Min(lo, v), stdmath.Max(hi, v)
	}
	return lo <= 0 && hi >= 0
}

// splitBezierHalf subdivides a scalar Bézier at t=0.5 (de Casteljau), returning the control
// coefficients of the left and right halves.
func splitBezierHalf(d []float64) (left, right []float64) {
	n := len(d)
	tmp := append([]float64(nil), d...)
	left, right = make([]float64, n), make([]float64, n)
	left[0], right[n-1] = tmp[0], tmp[n-1]
	for i := 1; i < n; i++ {
		for j := 0; j < n-i; j++ {
			tmp[j] = (tmp[j] + tmp[j+1]) / 2
		}
		left[i], right[n-1-i] = tmp[0], tmp[n-1-i]
	}
	return left, right
}

// bezierSegments2d decomposes a 2D B-spline into its Bézier segments by inserting every interior knot
// to full multiplicity (Piegl & Tiller §5.6): the refined control net then splits into degree+1-point
// Bézier pieces sharing seam points, one per knot span. A knot insertion that fails to validate leaves
// the original span un-split — clipping still runs on the whole span, only less tightly bracketed.
func bezierSegments2d(c BSplineCurve2d) []bezierSeg2d {
	breaks := domainBreakpoints(c.Knots, c.Degree)
	refined := c
	for _, u := range breaks[1 : len(breaks)-1] {
		if add := c.Degree - knotMultiplicity(c.Knots, u); add > 0 {
			if r, err := refined.InsertKnot(u, add); err == nil {
				refined = r
			}
		}
	}
	return sliceBezierSegments(refined, breaks)
}

// sliceBezierSegments carves the fully knot-refined curve into per-span Bézier control groups; segment
// k owns control points [k·degree, k·degree+degree] and the parent domain [breaks[k], breaks[k+1]].
func sliceBezierSegments(c BSplineCurve2d, breaks []float64) []bezierSeg2d {
	d := c.Degree
	segs := make([]bezierSeg2d, 0, len(breaks)-1)
	for k := 0; k+1 < len(breaks) && k*d+d < len(c.Ctrl); k++ {
		lo := k * d
		segs = append(segs, bezierSeg2d{ctrl: c.Ctrl[lo : lo+d+1], w: c.Weights[lo : lo+d+1], t0: breaks[k], t1: breaks[k+1]})
	}
	return segs
}

// containsPoint2 reports whether p already appears in pts within tol.
func containsPoint2(pts []math.Point2, p math.Point2, tol float64) bool {
	for _, q := range pts {
		if float64(q.DistanceTo(p)) <= tol {
			return true
		}
	}
	return false
}
