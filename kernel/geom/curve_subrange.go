// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// Restricting a curve to a sub-range of its own domain.
//
// A kernel edge's curve must span EXACTLY that edge — PointAt over its own domain running from the
// edge's start vertex to its end vertex — because every consumer that reads a curve's domain relies on
// it, the tessellator above all. A boolean's boundary walk hands back a run over [t0, t1] of some
// curve, and this is where that run becomes a curve of its own.
//
// The switch over curve kinds lives HERE, in geom, and not at the call site: adding a curve kind then
// means teaching one function, rather than finding every switch that must learn about it and leaving
// the ones nobody finds to take their default branch silently (#2188).

// SubCurve restricts a curve to [t0, t1] of its own domain, re-presented so the result's whole domain is
// that piece. A kind with a native restriction keeps its own kind — a circle's sub-range is an Arc3d, a
// ruled crossing's is a shorter ruled arc — and anything else is wrapped in a [TrimmedCurve3].
//
// t0 > t1 asks for the piece REVERSED, and the result runs that way. A curve kept whole is returned
// unchanged, so a full closed seam circle stays a Circle.
//
//	edge := geom.SubCurve(run.curve, run.t0, run.t1) // spans exactly this edge, in its traversal sense
func SubCurve(cv Curve3, t0, t1 float64) Curve3 {
	if storedWhole(cv, t0, t1) {
		return cv // kept whole: a closed loop stays the curve it is, whichever way the run walked it
	}
	switch c := cv.(type) {
	case Circle:
		return circleSubArc(c, t0, t1)
	case Arc3d:
		return arcSubArc(c, t0, t1)
	case LineSegment:
		return NewLineSegment(c.PointAt(t0), c.PointAt(t1))
	case Line:
		return NewLineSegment(c.PointAt(t0), c.PointAt(t1))
	}
	return conicSubRange(cv, t0, t1)
}

// conicSubRange restricts the analytic conic kinds and the ruled crossing, falling back to the generic
// restriction for a kind with none of its own.
func conicSubRange(cv Curve3, t0, t1 float64) Curve3 {
	// Both hyperbola forms restrict through ConicSubArc, which knows what each one's parameter means: a
	// Hyperbola's is θ, an already-bounded HyperbolicArc's is its own [0,1]. Storing a pre-clipped arc
	// unsliced would leave the edge's curve spanning more than its two vertices (#3459).
	if arc, ok := ConicSubArc(cv, t0, t1); ok {
		return arc
	}
	switch c := cv.(type) {
	case Parabola:
		return c.Arc(t0, t1) // a parabola loop edge's params are the cross coordinate t; store the bounded arc
	case EllipticalArc:
		return ellipticalSubArc(c, t0, t1) // restrict/re-anchor to the run's [t0,t1] (the reversed lobe walk)
	case EllipseFull:
		return ellipseSubArc(c, t0, t1) // a section sub-arc of a full ellipse (the (u,v) cone split)
	case SpiricArc:
		return spiricSubArc(c, t0, t1) // a torus-cut spiric branch, in its native tube-angle direction
	case RuledQuadricArc:
		return c.SubArc(t0, t1) // a ruled crossing clipped between triple points keeps its own kind
	}
	return TrimmedCurve3{Base: cv, Lo: t0, Hi: t1}
}

// storedWhole reports that a run may keep its curve unrestricted: it covers the whole domain, and it
// either walks it FORWARD or the curve is closed.
//
// A closed curve's edge carries one vertex and its direction rides on the use's reversed flag, so
// storing it forward is the convention — for EVERY closed kind. It was the rule for a circle and a
// ruled crossing only, and a closed spiric or a full ellipse walked backwards was re-presented running
// the other way; the stitch, which reads a closed run's direction against the curve's own parameter,
// then flagged the use as if the stored curve ran forward, and both faces on the loop walked it the
// same way. The torus figure-eight's second lobe came out inverted exactly so (ADR-0061 stage 4).
//
// An OPEN curve walked backwards is not the same case: the stitch anchors the start vertex to the
// run's first point, so a forward-stored curve then begins at the END vertex — and the tessellator,
// which pins a sampled polyline's ends to the vertices, folds the edge over itself.
func storedWhole(cv Curve3, t0, t1 float64) bool {
	return fullDomain(cv, t0, t1) && (t0 <= t1 || CurveIsClosed(cv))
}

// fullDomain reports whether [t0, t1] spans the curve's whole domain, in either direction. It reads
// the curve's OWN domain: a hyperbola's parameter is an angle, not a fraction, and an unbounded line
// has no whole to cover.
func fullDomain(cv Curve3, t0, t1 float64) bool {
	dlo, dhi := cv.Domain()
	if stdmath.IsInf(dlo, 0) || stdmath.IsInf(dhi, 0) || !(dhi > dlo) {
		return false
	}
	slack := subRangeSlack * (dhi - dlo)
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	return lo < dlo+slack && hi > dhi-slack
}

// subRangeSlack is how far from an endpoint a run's parameter may sit and still count as covering the
// whole domain, as a fraction of the domain's length, so it carries no model scale.
const subRangeSlack = 1e-9 // tol:parametric — a run's parameter at its curve's own endpoint, relative

// circleSubArc builds the Arc3d covering a circle's parameter sub-range [t0, t1] (Circle.PointAt(t) is
// the point at angle 2πt), so the edge tessellates over that arc alone.
func circleSubArc(c Circle, t0, t1 float64) Curve3 {
	a, _ := NewArc3d(c.Center, c.Normal.AsVector(), c.RefDir.AsVector(), c.Radius, twoPi*t0, twoPi*(t1-t0))
	return a
}

// arcSubArc restricts an Arc3d to a parameter sub-range [t0, t1].
func arcSubArc(a Arc3d, t0, t1 float64) Curve3 {
	return Arc3d{
		Center: a.Center, Normal: a.Normal, RefDir: a.RefDir, Radius: a.Radius,
		StartAngle: a.StartAngle + t0*a.SweepAngle, SweepAngle: (t1 - t0) * a.SweepAngle,
	}
}

// ellipseSubArc builds the EllipticalArc covering a full ellipse's parameter sub-range [t0, t1]
// (EllipseFull.PointAt(t) is the point at angle 2πt), so the edge tessellates over that arc alone.
func ellipseSubArc(e EllipseFull, t0, t1 float64) Curve3 {
	a, _ := NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius,
		twoPi*t0, twoPi*(t1-t0))
	return a
}

// ellipticalSubArc restricts a partial EllipticalArc to its sub-range [t0, t1], re-anchored so the
// stored curve's PointAt(0) is the run's first point and PointAt(1) its last (EllipticalArc.PointAt(t)
// walks StartAngle+t·SweepAngle over t∈[0,1]). A lobe of the equal-radius Steinmetz bicylinder walks its
// shared arc in the arc's DECREASING-parameter direction (t0=1, t1=0); keeping the arc's original
// forward parameterisation left PointAt(0) at the FAR pinch, 2R from the edge's StartVertex, so the
// face's discretised boundary crossed the solid and the (u,v) trim loop self-intersected (#1403). For a
// run that already spans the whole arc forward this returns an identical arc.
func ellipticalSubArc(e EllipticalArc, t0, t1 float64) Curve3 {
	a, _ := NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius,
		e.StartAngle+t0*e.SweepAngle, (t1-t0)*e.SweepAngle)
	return a
}

// spiricSubArc restricts a SpiricArc to its sub-range [t0, t1], stored in its NATIVE tube-angle
// direction (V0 < V1) regardless of how the run walks it — orientation is carried by the edge's reversed
// flag, not by flipping V0/V1. A reversed-range edge (V0 > V1) would mesh as a DIFFERENT region in the
// direction-sensitive spiric loft (the two branches of a bigon must both be native so the cap patch
// comes out the right size, #1406); the stitch anchors the edge to this native arc's endpoints so the
// reversed flag stays correct.
func spiricSubArc(sa SpiricArc, t0, t1 float64) Curve3 {
	v0 := sa.V0 + t0*(sa.V1-sa.V0)
	v1 := sa.V0 + t1*(sa.V1-sa.V0)
	if v0 > v1 {
		v0, v1 = v1, v0
	}
	sa.V0, sa.V1 = v0, v1
	return sa
}
