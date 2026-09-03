// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Exact line-in-face intervals for a planar face with curved boundary edges (ADR-0058). The mixed
// boolean needs where the line p0+t·dir lies INSIDE such a face — to clip a tool face's imprint to the
// shared region, and to prove non-interaction exactly. Every crossing of the line with the face's
// boundary is CLOSED FORM (lineSegCross for straight edges, the stable conic quadratic for circle
// edges), and the intervals are their even-odd pairing along the line — no sampled decision anywhere,
// the OCCT IntTools discipline. ok=false declines the query (an edge kind without a closed form, or a
// tangent grazing whose double root breaks the even-odd parity) so the caller stays conservative.

// curvedFaceLineIntervals returns the sorted parameter intervals of the line (p0 + t·dir, dir in the
// face plane) inside face f, exactly. The conic solver's segment is the face's OWN extent along the
// line (lineWindowOf), never a caller's clip range: even-odd parity is sound only when every boundary
// crossing is seen, and a narrow window silently reports a line lying wholly inside the face as
// outside it. A caller that wants a sub-range clips the returned intervals.
func curvedFaceLineIntervals(f curvedFace, p0 math.Point3, dir math.Vector3) ([][2]float64, bool) {
	pl, isPlane := f.surface.(geom.Plane)
	if !isPlane {
		return nil, false
	}
	window := lineWindowOf(f, p0, dir)
	o2, d2 := to2D(pl, p0), to2Dvec(pl, dir)
	var ts []float64
	for _, l := range f.loops {
		loopTs, ok := loopLineCrossings(l, pl, o2, d2, p0, dir, window)
		if !ok {
			return nil, false
		}
		ts = append(ts, loopTs...)
	}
	sort.Float64s(ts)
	ts = mergedLineParams(ts, geom.ResolutionForBox(faceLoopBox(f)).Weld()/stdmath.Sqrt(float64(dir.Dot(dir))))
	if len(ts)%2 != 0 {
		return nil, false // an unpaired crossing (grazing/degenerate): no sound even-odd pairing
	}
	out := make([][2]float64, 0, len(ts)/2)
	for i := 0; i+1 < len(ts); i += 2 {
		out = append(out, [2]float64{ts[i], ts[i+1]})
	}
	return out, true
}

// loopLineCrossings collects one loop's exact crossing parameters with the line: every edge's hits
// strictly inside it, plus each VERTEX on the line where the boundary passes from one side of it to
// the other. A vertex the boundary merely touches, or a run of edges collinear with the line, adds no
// crossing — the even-odd pairing stays sound exactly where the line follows the face's own boundary,
// as it does for a coplanar neighbour's edge (ADR-0060).
func loopLineCrossings(l curvedLoop, pl geom.Plane, o2 math.Point2, d2 math.Vector2, p0 math.Point3, dir math.Vector3, window [2]float64) ([]float64, bool) {
	var ts []float64
	n := len(l.edges)
	for i, e := range l.edges {
		hits, ok := edgeInteriorLineCrossings(e, pl, o2, d2, p0, dir, window)
		if !ok {
			return nil, false
		}
		ts = append(ts, hits...)
		if t, crosses := vertexLineCrossing(e, l.edges[(i+1)%n], pl, o2, d2); crosses {
			ts = append(ts, t)
		}
	}
	return ts, true
}

// edgeInteriorLineCrossings returns the line parameters where the edge crosses the line strictly
// inside its span; a straight edge parallel to the line contributes none.
func edgeInteriorLineCrossings(e loopEdge, pl geom.Plane, o2 math.Point2, d2 math.Vector2, p0 math.Point3, dir math.Vector3, window [2]float64) ([]float64, bool) {
	switch e.curve.(type) {
	case geom.LineSegment, geom.Line:
		a2, b2 := to2D(pl, e.start()), to2D(pl, e.end())
		t, hit := lineSegCrossStrict(o2, d2, a2, b2)
		if !hit {
			return nil, true
		}
		return []float64{t}, true
	default:
		return conicLineCrossings(e, pl, p0, dir, window)
	}
}

// vertexLineCrossing reports whether the boundary crosses the line AT the vertex where edge e ends and
// next begins: the vertex lies on the line and the boundary just before it and just after it sit on
// opposite sides. A collinear neighbour reads as "not above", so a collinear run counts once, at the
// end where the boundary leaves it upward.
func vertexLineCrossing(e, next loopEdge, pl geom.Plane, o2 math.Point2, d2 math.Vector2) (float64, bool) {
	v2 := to2D(pl, e.end())
	res := geom.ResolutionForPoints2D([]math.Point2{o2, v2})
	ov := o2.VectorTo(v2)
	if stdmath.Abs(d2.Cross(ov)) > res.Weld()*float64(d2.Length()) {
		return 0, false
	}
	before := to2D(pl, e.curve.PointAt(e.t1-(e.t1-e.t0)*vertexSideStep))
	after := to2D(pl, next.curve.PointAt(next.t0+(next.t1-next.t0)*vertexSideStep))
	above := func(q math.Point2) bool { return d2.Cross(o2.VectorTo(q)) > res.Weld()*float64(d2.Length()) }
	if above(before) == above(after) {
		return 0, false
	}
	return ov.Dot(d2) / d2.Dot(d2), true
}

// vertexSideStep is how far along an edge, in its own parameter, the side probe steps back from a vertex.
const vertexSideStep = 1e-3 // tol:parametric — side probe offset along the edge (dimensionless)

// lineSegCrossStrict is lineSegCross with the crossing required strictly inside the segment: a hit at
// either end is the vertex's, decided by vertexLineCrossing.
func lineSegCrossStrict(o math.Point2, d math.Vector2, a, b math.Point2) (float64, bool) {
	e := a.VectorTo(b)
	den := d.Cross(e)
	if stdmath.Abs(den) < parallelDenomTol {
		return 0, false
	}
	ao := o.VectorTo(a)
	s := ao.Cross(d) / den
	if s <= vertexSideStep || s >= 1-vertexSideStep {
		return 0, false
	}
	return ao.Cross(e) / den, true
}

// conicLineCrossings solves one conic edge — a whole circle or ellipse, or an ARC of a conic — against
// the line over the caller's window, in closed form (the stable conic quadratic), keeping only the
// crossings that lie within the edge's own span. ok=false for a curve kind that is no conic, and on a
// tangent grazing (double root — parity unsound).
func conicLineCrossings(e loopEdge, pl geom.Plane, p0 math.Point3, dir math.Vector3, window [2]float64) ([]float64, bool) {
	pc, ok := toPlaneConic(e.curve, pl)
	if !ok {
		return nil, false
	}
	a3 := p0.TranslateBy(dir.Scale(math.Scalar(window[0])))
	b3 := p0.TranslateBy(dir.Scale(math.Scalar(window[1])))
	hits, tangent := conicEdgeHits(pc, to2D(pl, a3), to2D(pl, b3), geom.ResolutionForPoints([]math.Point3{a3, b3}))
	if tangent {
		return nil, false
	}
	ts := make([]float64, 0, len(hits))
	for _, h := range hits {
		if edgeParamStrictlyWithin(e, to3D(pl, h.p)) {
			ts = append(ts, window[0]+h.sEdge*(window[1]-window[0]))
		}
	}
	return ts, true
}

// edgeParamStrictlyWithin reports a point on the edge's conic lying strictly inside the edge's span,
// clear of both ends by the vertex probe step (a whole closed curve has no ends).
func edgeParamStrictlyWithin(e loopEdge, p math.Point3) bool {
	t, ok := edgeParamWithin(e, p)
	if !ok {
		return false
	}
	lo, hi := stdmath.Min(e.t0, e.t1), stdmath.Max(e.t0, e.t1)
	if dlo, dhi := e.curve.Domain(); geom.CurveIsClosed(e.curve) && hi-lo >= (dhi-dlo)-tjTol {
		return true
	}
	span := hi - lo
	return t > lo+vertexSideStep*span && t < hi-vertexSideStep*span
}

// exactFaceLineIntervals clips the line p0+t·dir to a face's TRUE trim: the polygon intervals for an
// all-straight face, the exact conic even-odd intervals for a face carrying curved loops.
func exactFaceLineIntervals(f curvedFace, p0 math.Point3, dir math.Vector3) ([][2]float64, bool) {
	if allStraightFace(f) {
		return faceLineIntervals(f, p0, dir), true
	}
	return curvedFaceLineIntervals(f, p0, dir)
}

// lineWindowOf is a parameter window covering the face's extent along the line: each edge's exact
// extent along dir (its ends and, for a conic, its interior stationary points — geom.AxialExtent),
// widened by the pair cull pad. An edge kind without a closed-form extent is projected at its ends.
func lineWindowOf(f curvedFace, p0 math.Point3, dir math.Vector3) [2]float64 {
	den := float64(dir.Dot(dir))
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, l := range f.loops {
		for _, e := range l.edges {
			elo, ehi, ok := geom.AxialExtent(e.curve, e.t0, e.t1, p0, dir)
			if !ok {
				elo = float64(p0.VectorTo(e.start()).Dot(dir))
				ehi = float64(p0.VectorTo(e.end()).Dot(dir))
			}
			lo, hi = stdmath.Min(lo, stdmath.Min(elo, ehi)/den), stdmath.Max(hi, stdmath.Max(elo, ehi)/den)
		}
	}
	pad := facePairCullPad / stdmath.Sqrt(den)
	return [2]float64{lo - pad, hi + pad}
}

// mergedLineParams collapses crossings that coincide within one weld along the line: a crossing AT the
// vertex two edges share is reported by both edges and is one crossing, not two (ADR-0060).
func mergedLineParams(ts []float64, slack float64) []float64 {
	if len(ts) < 2 {
		return ts
	}
	out := ts[:1]
	for _, t := range ts[1:] {
		if t-out[len(out)-1] > slack {
			out = append(out, t)
		}
	}
	return out
}
