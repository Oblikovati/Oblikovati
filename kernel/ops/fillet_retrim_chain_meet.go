// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Where a retrim CHAIN meets a host RING. This is the geometry half of the chain-capable loop rebuild
// (fillet_retrim_chain.go): given the host face's original boundary as an ordered ring of endSegs and
// one segment of the fillet's contact chain, it answers "at what parameter along this chain segment
// does the chain first reach the boundary, and where". That question is what turns a chain running
// PAST the face — the far-end trim's off-face landing, the setback band's overrun of its own host —
// into one whose extremes lie ON the boundary, which is the precondition every existing splice needs.
//
// It is deliberately a DISTANCE minimisation rather than a curve/curve intersector: a retrim chain
// carries whatever curve its producer built (a section arc, a band∩wall b-spline, a spiric), and the
// ring carries lines, arcs and conic rims, so the pairwise-intersector matrix would be quadratic in the
// curve families and would still have to decline the pairs it has no closed form for. The minimisation
// asks only for PointAt, converges linearly on a transversal crossing (the distance is V-shaped there,
// not quadratic), and is gated by tol at the end — so a meeting it reports is on the ring within the
// model weld, and one it cannot resolve is honestly declined rather than guessed.

// ringMeetScan is the number of equal parameter steps the meet search walks along one chain segment
// looking for the ring. It only has to SEPARATE the distance function's local minima — the refinement
// below finds each one exactly — so it is sized for the coarsest realistic chain (a terminal section
// spanning a quadrant against a ring of a handful of edges), not for accuracy.
const ringMeetScan = 512

// ringMeetRefine is the number of golden-section contractions applied to a bracketed minimum. Each
// contracts the bracket by 0.618, so 60 takes one scan step (1/512 of the chain segment) to ~1e-14 of
// it — below the parameter resolution of any curve the retrim carries.
const ringMeetRefine = 60

// ringSampleChords is the chord count used to measure the distance to a boundary segment whose curve is
// neither straight nor a circular arc (an elliptic or spline rim). Sampling only ever OVER-states the
// distance, so it can make the search miss a meeting (→ honest decline) but never invent one.
const ringSampleChords = 256

// goldenInset is the golden-section interior fraction (1 − 1/φ), the point pair that contracts a
// bracketed minimum by the largest constant factor per evaluation.
const goldenInset = 0.3819660112501051

// segPointAt evaluates a boundary segment at fractional parameter t ∈ [0,1]: along its carried curve
// when it has one — a segment's curve runs from→to by the loop's orientation invariant
// (alignCarriedArcsToSegments, planar-retrim-selfcross-report.md §0) — and along the straight chord
// otherwise.
func segPointAt(s endSeg, t float64) math.Point3 {
	if s.curve == nil {
		return s.from.Lerp(s.to, math.Scalar(t))
	}
	lo, hi := s.curve.Domain()
	return s.curve.PointAt(lo + t*(hi-lo))
}

// distToRing is the distance from p to the nearest point of the host ring.
func distToRing(ring []endSeg, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, r := range ring {
		if d := distToRingSeg(r, p); d < best {
			best = d
		}
	}
	return best
}

// distToRingSeg is the distance from p to one boundary segment: exact for a straight edge and for a
// circular arc, a chord sample for any other carried curve.
func distToRingSeg(s endSeg, p math.Point3) float64 {
	if s.curve == nil {
		return distToLineSeg(s.from, s.to, p)
	}
	if arc, ok := s.curve.(geom.Arc3d); ok {
		return distToArcSeg(arc, s, p)
	}
	return distToSampledSeg(s, p)
}

// distToLineSeg is the distance from p to the straight segment a→b (clamped to the segment, so a point
// beyond an end measures to that end, not to the supporting line).
func distToLineSeg(a, b, p math.Point3) float64 {
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return float64(p.DistanceTo(a))
	}
	t := stdmath.Min(1, stdmath.Max(0, float64(a.VectorTo(p).Dot(d))/l2))
	return float64(a.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p))
}

// distToArcSeg is the distance from p to an arc edge: to p's projection on the arc's OWN circle when
// that projection falls inside the arc's sweep, else to the nearer endpoint — the arc analogue of
// clamping a line projection to its segment.
func distToArcSeg(arc geom.Arc3d, s endSeg, p math.Point3) float64 {
	q := projectOntoArcCircle(arc, p)
	if t, ok := arcFrac(arc, q); ok && t >= 0 && t <= 1 {
		return float64(p.DistanceTo(q))
	}
	return stdmath.Min(float64(p.DistanceTo(s.from)), float64(p.DistanceTo(s.to)))
}

// distToSampledSeg is the distance from p to a chord sampling of a boundary segment whose curve has no
// closed-form projection (an elliptic or spline rim).
func distToSampledSeg(s endSeg, p math.Point3) float64 {
	best := stdmath.Inf(1)
	prev := segPointAt(s, 0)
	for i := 1; i <= ringSampleChords; i++ {
		cur := segPointAt(s, float64(i)/ringSampleChords)
		if d := distToLineSeg(prev, cur, p); d < best {
			best = d
		}
		prev = cur
	}
	return best
}

// ringMeetOnSeg returns the parameter along chain segment s at which the chain reaches the ring, the
// meeting point, and whether one was found within tol. fromHead scans t ascending (the meeting that
// ends a LEADING overrun); otherwise it scans descending (the one that begins a TRAILING overrun) — so
// one search serves both ends of the clip and neither has to reverse the segment. That was once a
// CORRECTNESS requirement — the layer's chain reversal swapped a non-arc segment's endpoints and left
// its curve pointing the original way, so a reverse-clip-reverse round trip trimmed the WRONG sub-span
// — and is now simply the cheaper and more direct form, the defect having been fixed in reversedEndSeg.
func ringMeetOnSeg(ring []endSeg, s endSeg, tol float64, fromHead bool) (float64, math.Point3, bool) {
	d := sampledRingDistances(ring, s)
	for k := range d {
		i := k
		if !fromHead {
			i = len(d) - 1 - k
		}
		if !isSampledLocalMin(d, i) {
			continue
		}
		lo := stdmath.Max(0, float64(i-1)/ringMeetScan)
		hi := stdmath.Min(1, float64(i+1)/ringMeetScan)
		if t, p, ok := refineRingMeet(ring, s, lo, hi, tol); ok {
			return t, p, true
		}
	}
	return 0, math.Point3{}, false
}

// sampledRingDistances is the chain segment's distance to the ring at ringMeetScan+1 equal parameters.
func sampledRingDistances(ring []endSeg, s endSeg) []float64 {
	d := make([]float64, ringMeetScan+1)
	for i := range d {
		d[i] = distToRing(ring, segPointAt(s, float64(i)/ringMeetScan))
	}
	return d
}

// isSampledLocalMin reports whether sample i is a local minimum of the sampled distance — a candidate
// bracket for a real meeting. Either end counts when the run is still descending toward it (a chain
// that reaches the ring exactly at one of its own endpoints has no rising side there).
func isSampledLocalMin(d []float64, i int) bool {
	if i > 0 && d[i] > d[i-1] {
		return false
	}
	return i == len(d)-1 || d[i] <= d[i+1]
}

// refineRingMeet contracts a bracketed minimum of the chain-to-ring distance by golden section and
// accepts it only when the refined distance is within tol — so a bracket that merely APPROACHES the
// boundary (a chain passing close by without touching) is rejected rather than snapped onto it.
func refineRingMeet(ring []endSeg, s endSeg, lo, hi, tol float64) (float64, math.Point3, bool) {
	for k := 0; k < ringMeetRefine; k++ {
		a, b := lo+(hi-lo)*goldenInset, hi-(hi-lo)*goldenInset
		if distToRing(ring, segPointAt(s, a)) <= distToRing(ring, segPointAt(s, b)) {
			hi = b
			continue
		}
		lo = a
	}
	t := (lo + hi) / 2
	p := segPointAt(s, t)
	if distToRing(ring, p) > tol {
		return 0, math.Point3{}, false
	}
	return t, p, true
}
