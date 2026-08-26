// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
)

// Conformal circular sampling (ADR-0054, Oblikovati#2167).
//
// Two operands of a mesh boolean that share a circular boundary — a cylinder's rim
// and a coaxial arc are literally the SAME circle — must discretize it into IDENTICAL
// points, or the boolean sees two mismatched faceted approximations and keeps a
// zero-volume sliver membrane between them (the #2167 seam). Adaptive midpoint
// bisection cannot guarantee this: it samples from each curve's ARBITRARY stored
// RefDir over its local [0,1] domain, so the same physical circle lands its vertices
// at different absolute angles per operand, and even the segment COUNT can differ.
//
// The fix is to sample at CANONICAL absolute angles derived only from the circle's
// axis LINE and radius — a property of the geometry, not of how the edge happened to
// be constructed. Then any two coincident circular curves conform BY CONSTRUCTION,
// independent of RefDir, normal sign, or edge identity. An arc's interior points are a
// strict subset of the coaxial full circle's points (identical formula), so an arc and
// the circle it lies on conform too; the arc's own endpoints (where a chord meets it)
// are imprinted onto the circle by the boolean's co-refinement.

// angEps is the angular slack (radians) below which a canonical angle is treated as
// coincident with an arc endpoint, so a canonical sample sitting exactly on the arc's
// start/end is not emitted twice.
const angEps = 1e-9

// CircleSegments returns the number of equal segments a full circle of the given radius is
// divided into at the (chordTol, angleTol) quality: the smallest POWER OF TWO that both
// satisfies the chordal sagitta bound r(1−cos(π/n)) ≤ chordTol and holds each facet's turn
// to angleTol (n ≥ π/angleTol). A power of two reproduces exactly what the adaptive
// midpoint bisection [adaptiveParams] converges to for a circle, so switching a circle/arc
// to canonical sampling leaves its facet density — and thus curved area/volume error —
// unchanged, while making n a pure function of radius+quality (never of an edge's
// parameterization). That is half of the conformance invariant; the shared canonical frame
// is the other half.
//
//	CircleSegments(3, 0.05, 10*math.Pi/180) // → 32, the display-quality facet count
func CircleSegments(radius, chordTol, angleTol float64) int {
	need := 3.0 // a circle needs at least a triangle's worth of segments
	if angleTol > 0 {
		need = stdmath.Max(need, stdmath.Pi/angleTol)
	}
	if chordTol > 0 && chordTol < radius {
		need = stdmath.Max(need, stdmath.Pi/stdmath.Acos(1-chordTol/radius))
	}
	n := 4
	for float64(n) < need {
		n *= 2
	}
	return n
}

// canonicalCircleAxis returns axis flipped into a fixed hemisphere, so a circle and a
// COAXIAL mate built with the opposite normal sign resolve to ONE shared frame. The sign
// is fixed by the first component (Z, then Y, then X) that is not ~0: two normals on the
// same line — however each was oriented — canonicalize to the identical axis.
func canonicalCircleAxis(n math.UnitVector3) math.UnitVector3 {
	dominant := n.Z()
	if stdmath.Abs(dominant) <= angEps {
		dominant = n.Y()
	}
	if stdmath.Abs(dominant) <= angEps {
		dominant = n.X()
	}
	if dominant < 0 {
		return n.Negate()
	}
	return n
}

// canonicalCircleFrame returns a right-handed in-plane basis (u0, w0) for a circle of the
// given axis, derived only from the axis LINE (via canonicalCircleAxis). Both vectors are
// a deterministic function of that sign-canonical axis, so two coincident circular curves
// share this frame and therefore sample to identical points at each canonical angle.
func canonicalCircleFrame(axis math.UnitVector3) (u0, w0 math.Vector3) {
	a := canonicalCircleAxis(axis)
	u := perpendicularUnit(a)
	return u.AsVector(), a.Cross(u)
}

// CircleConformalSamples returns the closed polyline of a full circle at canonical absolute
// angles 2πk/n (n = CircleSegments(radius, chordTol, angleTol)), ANCHORED at the seam point
// so the polyline starts and ends there — the seam is the circle edge's start/end vertex,
// where any seam rulings attach, so the boundary must pass through it and stay angularly
// monotonic (a non-monotonic ring folds the face mesh). The interior samples stay at the
// absolute canonical angles, so two circles with equal centre, axis (up to sign), and radius
// share every interior point regardless of RefDir or seam — the boolean conformance
// invariant; each contributes only its own seam point, which co-refinement imprints. The
// parallel params slice carries each point's value in the circle's OWN [0,1] domain.
func CircleConformalSamples(c Circle, seam math.Point3, chordTol, angleTol float64) (pts []math.Point3, params []float64) {
	u0, w0 := canonicalCircleFrame(c.Normal)
	ref, bin := c.RefDir.AsVector(), c.binormal()
	seamAbs := frameAngle(c.Center, u0, w0, seam)
	ordered := canonicalAnglesFromSeam(seamAbs, CircleSegments(c.Radius, chordTol, angleTol))
	pts = append(pts, seam)
	params = append(params, frameAngle(c.Center, ref, bin, seam)/twoPi)
	for _, ang := range ordered {
		p := pointOnCircle(c.Center, u0, w0, c.Radius, ang)
		pts = append(pts, p)
		params = append(params, frameAngle(c.Center, ref, bin, p)/twoPi)
	}
	return append(pts, seam), append(params, params[0]+1) // close on the seam, param at the domain end
}

// canonicalAnglesFromSeam returns the n canonical angles 2πk/n that are not essentially AT
// the seam (within angEps), ordered by their angular distance from seamAbs — i.e. the ring
// starting just after the seam and sweeping a full turn back to it.
func canonicalAnglesFromSeam(seamAbs float64, n int) []float64 {
	type ordered struct{ frac, ang float64 }
	var s []ordered
	for k := 0; k < n; k++ {
		ang := twoPi * float64(k) / float64(n)
		if frac := wrap2pi(ang - seamAbs); frac > angEps && frac < twoPi-angEps {
			s = append(s, ordered{frac, ang})
		}
	}
	sort.Slice(s, func(i, j int) bool { return s[i].frac < s[j].frac })
	out := make([]float64, len(s))
	for i := range s {
		out[i] = s[i].ang
	}
	return out
}

// ArcConformalSamples returns points along the arc from its start point to its end point:
// the two exact endpoints, with every canonical full-circle angle 2πk/n the arc covers
// placed between them in arc order. Because the interior points use the SAME frame and
// formula as [CircleConformalSamples], an arc and the coaxial full circle it lies on
// share every interior vertex; the two endpoints are the arc's own. The parallel params
// slice carries each point's value in the arc's [0,1] domain — t is the along-arc
// fraction, exact because the arc is uniform in angle (endpoints 0 and 1).
func ArcConformalSamples(a Arc3d, chordTol, angleTol float64) (pts []math.Point3, params []float64) {
	u0, w0 := canonicalCircleFrame(a.Normal)
	start, end := a.PointAt(0), a.PointAt(1)
	n := CircleSegments(a.Radius, chordTol, angleTol)
	sweepMag := stdmath.Abs(a.SweepAngle)
	startAbs := frameAngle(a.Center, u0, w0, start)
	interior := arcInteriorAngles(startAbs, sweepMag, arcWindsCCW(a), n)
	pts = append(pts, start)
	params = append(params, 0)
	for _, s := range interior {
		pts = append(pts, pointOnCircle(a.Center, u0, w0, a.Radius, s.ang))
		params = append(params, s.frac/sweepMag)
	}
	return append(pts, end), append(params, 1)
}

// frameAngle returns the angle (in [0, 2π)) of point p about center in the plane basis
// (u0, w0) — atan2 of p's (w0, u0) components.
func frameAngle(center math.Point3, u0, w0 math.Vector3, p math.Point3) float64 {
	d := center.VectorTo(p)
	return wrap2pi(stdmath.Atan2(d.Dot(w0), d.Dot(u0)))
}

// arcWindsCCW reports whether the arc advances in the +angle direction of the canonical
// frame. SweepAngle is signed about the arc's OWN Normal; the canonical axis is ±Normal,
// so the canonical winding flips when the canonical axis opposes Normal.
func arcWindsCCW(a Arc3d) bool {
	sameDir := a.Normal.Dot(canonicalCircleAxis(a.Normal)) > 0
	return (a.SweepAngle >= 0) == sameDir
}

// canonAngle is one interior canonical sample of an arc: its absolute angle ang in the
// canonical frame (where the point is placed) and its along-arc distance frac from the
// arc start (which gives the curve parameter t = frac/sweepMag).
type canonAngle struct{ frac, ang float64 }

// arcInteriorAngles returns the canonical angles 2πk/n strictly inside an arc that starts
// at startAbs and covers sweepMag radians, winding CCW (or CW) in the canonical frame,
// ordered start→end by their along-arc distance. A canonical station within HALF a step
// (π/n) of either endpoint is dropped: the exact endpoint already covers that corner, and
// emitting a station a sliver's width from it makes a near-degenerate facet that folds a
// downstream mesh. Conformance survives — the kept stations are still a subset of the full
// circle's, so a coaxial circle and arc still share every station the arc keeps.
func arcInteriorAngles(startAbs, sweepMag float64, ccw bool, n int) []canonAngle {
	minGap := stdmath.Pi / float64(n) // half a canonical step
	var s []canonAngle
	for k := 0; k < n; k++ {
		ang := twoPi * float64(k) / float64(n)
		frac := wrap2pi(ang - startAbs)
		if !ccw {
			frac = wrap2pi(startAbs - ang)
		}
		if frac > minGap && frac < sweepMag-minGap {
			s = append(s, canonAngle{frac, ang})
		}
	}
	sort.Slice(s, func(i, j int) bool { return s[i].frac < s[j].frac })
	return s
}
