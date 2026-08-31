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
	if len(ts)%2 != 0 {
		return nil, false // an unpaired crossing (grazing/degenerate): no sound even-odd pairing
	}
	out := make([][2]float64, 0, len(ts)/2)
	for i := 0; i+1 < len(ts); i += 2 {
		out = append(out, [2]float64{ts[i], ts[i+1]})
	}
	return out, true
}

// loopLineCrossings collects one loop's exact crossing parameters with the line.
func loopLineCrossings(l curvedLoop, pl geom.Plane, o2 math.Point2, d2 math.Vector2, p0 math.Point3, dir math.Vector3, window [2]float64) ([]float64, bool) {
	var ts []float64
	for _, e := range l.edges {
		switch e.curve.(type) {
		case geom.LineSegment, geom.Line:
			a2, b2 := to2D(pl, e.start()), to2D(pl, e.end())
			if t, hit := lineSegCross(o2, d2, a2, b2); hit {
				ts = append(ts, t)
			}
		case geom.Circle:
			ct, ok := circleLineCrossings(e, pl, p0, dir, window)
			if !ok {
				return nil, false
			}
			ts = append(ts, ct...)
		default:
			return nil, false // no closed form for this edge kind yet: decline the query
		}
	}
	return ts, true
}

// circleLineCrossings solves one full-circle edge against the line over the caller's window, in closed
// form (the stable conic quadratic). ok=false on a tangent grazing (double root — parity unsound).
func circleLineCrossings(e loopEdge, pl geom.Plane, p0 math.Point3, dir math.Vector3, window [2]float64) ([]float64, bool) {
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
		ts = append(ts, window[0]+h.sEdge*(window[1]-window[0]))
	}
	return ts, true
}

// exactFaceLineIntervals clips the line p0+t·dir to a face's TRUE trim: the polygon intervals for an
// all-straight face, the exact conic even-odd intervals for a face carrying curved loops.
func exactFaceLineIntervals(f curvedFace, p0 math.Point3, dir math.Vector3) ([][2]float64, bool) {
	if allStraightFace(f) {
		return faceLineIntervals(f, p0, dir), true
	}
	return curvedFaceLineIntervals(f, p0, dir)
}

// lineWindowOf is a conservative parameter window covering the face's extent along the line: every
// loop-edge endpoint's projection, widened by each circle edge's radius (a superset by construction).
func lineWindowOf(f curvedFace, p0 math.Point3, dir math.Vector3) [2]float64 {
	den := float64(dir.Dot(dir))
	lo, hi, slack := stdmath.Inf(1), stdmath.Inf(-1), 0.0
	for _, l := range f.loops {
		for _, e := range l.edges {
			t := float64(p0.VectorTo(e.start()).Dot(dir)) / den
			lo, hi = stdmath.Min(lo, t), stdmath.Max(hi, t)
			if c, isCircle := e.curve.(geom.Circle); isCircle {
				tc := float64(p0.VectorTo(c.Center).Dot(dir)) / den
				r := c.Radius / stdmath.Sqrt(den)
				lo, hi = stdmath.Min(lo, tc-r), stdmath.Max(hi, tc+r)
				slack = stdmath.Max(slack, r)
			}
		}
	}
	return [2]float64{lo - slack - facePairCullPad, hi + slack + facePairCullPad}
}
