// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Conic-against-conic crossings on a planar face's own boundary (Oblikovati/Oblikovati#3503).
//
// A section conic entering a planar face used to be tested against that face's boundary as a
// POLYLINE, which is exact only while every boundary edge is straight — allStraightFace is the gate
// that admitted it. A face whose boundary carries arcs (an exact section edge cut from a neighbouring
// wall) has to be tested against those arcs themselves, and that is a conic meeting a conic.
//
// The pairing follows geom's bucketing rather than a table of kinds: the SECTION is taken implicitly,
// as the quadratic form it satisfies, and each boundary EDGE parametrically, as a point moving along
// it. So one routine covers arc-against-ellipse, arc-against-hyperbola and the straight case alike.

// conicEdgeCrossings counts where the section conic meets ONE boundary edge, and whether it grazes
// it. ok=false when the edge is a curve this cannot put in parametric form, which the caller must
// treat as "cannot decide" rather than "does not cross".
func conicEdgeCrossings(pc planeConic, e loopEdge, pl geom.Plane, res geom.Resolution) (hits int, tangent, ok bool) {
	if isStraightEdge(e) {
		hs, tan := conicEdgeHits(pc, to2D(pl, e.start()), to2D(pl, e.end()), res)
		return len(hs), tan, true
	}
	params, span, got := edgeConicParams(e, pl)
	if !got {
		return 0, false, false
	}
	form, formOK := pc.implicit()
	if !formOK {
		return 0, false, false
	}
	ts, infinite := geom.IntersectConic2d(params, form)
	if infinite {
		return 0, true, true // the edge lies ON the section: a graze along its whole length
	}
	return countInSpan(ts, span, params.Hyperbolic), false, true
}

// edgeSpan is the parameter interval of one boundary edge in its OWN curve's parameter — the angle
// for an arc, the hyperbolic angle for a hyperbola branch — against which a root is admitted.
type edgeSpan struct{ lo, hi float64 }

// countInSpan counts the roots lying within the edge's own parameter interval. An angular parameter
// is compared MODULO a turn, because a root reported on [0, 2π) and an arc spanning the seam name the
// same place by different numbers; a hyperbolic one is not periodic and is compared directly.
func countInSpan(ts []float64, span edgeSpan, hyperbolic bool) int {
	n := 0
	for _, t := range ts {
		if hyperbolic {
			if t >= span.lo-conicSpanSlack && t <= span.hi+conicSpanSlack {
				n++
			}
			continue
		}
		if angleWithin(t, span) {
			n++
		}
	}
	return n
}

// angleWithin reports an angle inside the span, both folded onto one turn from the span's start so a
// span crossing the seam stays one interval.
func angleWithin(t float64, span edgeSpan) bool {
	turn := 2 * stdmath.Pi
	width := span.hi - span.lo
	if width >= turn-conicSpanSlack {
		return true // the edge is a whole closed conic: every root is on it
	}
	d := stdmath.Mod(t-span.lo, turn)
	if d < 0 {
		d += turn
	}
	return d <= width+conicSpanSlack
}

// conicSpanSlack is how far outside its own parameter interval a root may fall and still count as on
// the edge. It is a PARAMETER slack on curves whose parameters are angles, so it carries no model
// scale; it exists because a root landing exactly on an edge's endpoint is a genuine crossing that
// float noise can push either side of.
const conicSpanSlack = 1e-9 // tol:parametric — root admitted at an edge's own endpoint

// implicit returns the section conic as the quadratic form it satisfies, the currency a boundary edge
// is substituted into. ok=false for a degenerate conic with no extent to write.
func (pc planeConic) implicit() (geom.Conic2dImplicit, bool) {
	return geom.ImplicitConic2dOf(pc.center, pc.maj, perp2(pc.maj), pc.A, pc.B, pc.hyper)
}

// perp2 is the in-plane quarter turn, the conjugate axis direction a conic's frame is completed by.
func perp2(v math.Vector2) math.Vector2 { return math.V2(-v.Y, v.X) }

// edgeConicParams puts one boundary edge in parametric conic form IN THE FACE'S PLANE, with the
// parameter interval it actually covers. The edge already lies in that plane, so its centre and axes
// project directly and the parameter carries over unchanged — no refitting, and no chance of the
// projection disagreeing with the curve the edge stores.
//
// ok=false for a curve with no conic form here (a b-spline edge), which the caller declines on rather
// than approximating.
func edgeConicParams(e loopEdge, pl geom.Plane) (geom.EllipticalParams2d, edgeSpan, bool) {
	switch c := e.curve.(type) {
	case geom.Circle:
		return circleParams(c.Center, c.RefDir.AsVector(), c.Normal.Cross(c.RefDir), c.Radius, c.Radius, pl,
			edgeSpan{0, 2 * stdmath.Pi})
	case geom.Arc3d:
		return circleParams(c.Center, c.RefDir.AsVector(), c.Normal.Cross(c.RefDir), c.Radius, c.Radius, pl,
			sweepSpan(c.StartAngle, c.SweepAngle))
	case geom.EllipseFull:
		return circleParams(c.Center, c.MajorAxis.AsVector(), c.Normal.Cross(c.MajorAxis), c.MajorRadius, c.MinorRadius, pl,
			edgeSpan{0, 2 * stdmath.Pi})
	case geom.EllipticalArc:
		return circleParams(c.Center, c.MajorAxis.AsVector(), c.Normal.Cross(c.MajorAxis), c.MajorRadius, c.MinorRadius, pl,
			sweepSpan(c.StartAngle, c.SweepAngle))
	case geom.HyperbolicArc:
		p, span, ok := circleParams(c.Center, c.TransverseAxis.AsVector(), c.ConjugateAxis.AsVector(), c.A, c.B, pl,
			orderedSpan(c.Theta0, c.Theta1))
		p.Hyperbolic = true
		return p, span, ok
	}
	return geom.EllipticalParams2d{}, edgeSpan{}, false
}

// circleParams projects a conic's centre and its two axis directions into the plane. The axes stay
// unit there because the curve lies IN the plane, so the projection is a rotation on them.
func circleParams(center math.Point3, u, v math.Vector3, a, b float64, pl geom.Plane, span edgeSpan) (geom.EllipticalParams2d, edgeSpan, bool) {
	if a == 0 || b == 0 {
		return geom.EllipticalParams2d{}, edgeSpan{}, false
	}
	return geom.EllipticalParams2d{
		Center: to2D(pl, center), U: to2Dvec(pl, u), V: to2Dvec(pl, v), A: a, B: b,
	}, span, true
}

// sweepSpan turns a start angle and a signed sweep into an ascending interval, so a clockwise arc and
// its counter-clockwise twin admit the same roots.
func sweepSpan(start, sweep float64) edgeSpan {
	if sweep < 0 {
		return edgeSpan{start + sweep, start}
	}
	return edgeSpan{start, start + sweep}
}

// orderedSpan is the ascending form of an interval given either way round.
func orderedSpan(a, b float64) edgeSpan {
	if b < a {
		return edgeSpan{b, a}
	}
	return edgeSpan{a, b}
}

// isStraightEdge reports an edge whose curve is a line, the case conicEdgeHits already solves in
// closed form against a segment.
func isStraightEdge(e loopEdge) bool {
	switch e.curve.(type) {
	case geom.LineSegment, geom.Line:
		return true
	}
	return false
}
