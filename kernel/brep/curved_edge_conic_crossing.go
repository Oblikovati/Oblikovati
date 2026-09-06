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
	ps, tangent, ok := conicEdgeCrossingPoints(pc, e, pl, res)
	return len(ps), tangent, ok
}

// conicEdgeCrossingPoints is conicEdgeCrossings with the crossings themselves: WHERE the section meets
// the edge, in the face's own plane chart. Counting and locating are the same solve, so they are one
// routine — a clip that needs the parameters must not re-derive them by a second, polyline route that
// can disagree with the count (ADR-0061 stage 4).
func conicEdgeCrossingPoints(pc planeConic, e loopEdge, pl geom.Plane, res geom.Resolution) ([]math.Point2, bool, bool) {
	if isStraightEdge(e) {
		hs, tan := conicEdgeHits(pc, to2D(pl, e.start()), to2D(pl, e.end()), res)
		out := make([]math.Point2, 0, len(hs))
		for _, h := range hs {
			out = append(out, h.p)
		}
		return out, tan, true
	}
	params, span, got := edgeConicParams(e, pl)
	if !got {
		return nil, false, false
	}
	form, formOK := pc.implicit()
	if !formOK {
		return nil, false, false
	}
	ts, infinite := geom.IntersectConic2d(params, form)
	if infinite {
		return nil, true, true // the edge lies ON the section: a graze along its whole length
	}
	return pointsInSpan(params, ts, span), false, true
}

// pointsInSpan evaluates the roots that lie within the edge's own parameter interval.
func pointsInSpan(params geom.EllipticalParams2d, ts []float64, span edgeSpan) []math.Point2 {
	var out []math.Point2
	for _, t := range ts {
		if inEdgeSpan(t, span, params.Hyperbolic) {
			out = append(out, params.PointAt(t))
		}
	}
	return out
}

// edgeSpan is the parameter interval of one boundary edge in its OWN curve's parameter — the angle
// for an arc, the hyperbolic angle for a hyperbola branch — against which a root is admitted.
type edgeSpan struct{ lo, hi float64 }

// inEdgeSpan reports a root lying within the edge's own parameter interval. An angular parameter
// is compared MODULO a turn, because a root reported on [0, 2π) and an arc spanning the seam name the
// same place by different numbers; a hyperbolic one is not periodic and is compared directly.
func inEdgeSpan(t float64, span edgeSpan, hyperbolic bool) bool {
	if hyperbolic {
		return t >= span.lo-conicSpanSlack && t <= span.hi+conicSpanSlack
	}
	return angleWithin(t, span)
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
// parameter interval it covers. The switch over curve kinds lives in geom, where the kernel rules put
// it, so what remains here is only the edgeSpan adaptation.
func edgeConicParams(e loopEdge, pl geom.Plane) (geom.EllipticalParams2d, edgeSpan, bool) {
	params, lo, hi, ok := geom.PlaneConicParams(e.curve, pl)
	return params, edgeSpan{lo, hi}, ok
}

// isStraightEdge reports an edge whose curve is a line, the case conicEdgeHits already solves in
// closed form against a segment.
func isStraightEdge(e loopEdge) bool { return geom.IsStraightCurve(e.curve) }
