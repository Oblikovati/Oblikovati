// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// trimUVSamples is the per-edge sampling when a loop is projected into the surface's (u, v) domain.
// A point well inside the face classifies correctly from a modest polyline; a point nearer a curved
// trim edge than the sampling error is handled by faceBoundaryBand reselection, not by more samples —
// so this is a balance (fewer reselections) rather than an accuracy floor. It is not a face tessellation.
const trimUVSamples = 32

// domainPeriodTol accepts a parameter span as a full 2π turn; the analytic surfaces return an angular
// domain of exactly [0, 2π], so this only guards floating comparison, not a model quantity.
const domainPeriodTol = 1e-9 // tol:parametric — angular domain span ≈ 2π

// pointInTrimUV reports whether p (on f's surface) lies within f's trimmed region, by classifying its
// surface parameters against the loops projected into the (u, v) domain — the parameter-space
// classification production kernels use (OCCT BRepClass_FaceClassifier). Unlike the tangent-plane
// geodesic winding, it is correct for a full periodic band (a whole cylinder/cone side bounded by two
// complete circles), because a periodic parameter is unwrapped into one continuous branch before the
// even-odd test. A boundary-less face (a whole sphere/torus) contains every surface point. When the
// surface has no non-periodic, UNBOUNDED axis to cast the even-odd ray toward (a sphere's latitude ends
// at a pole; a torus wraps both ways), it defers to the geodesic winding, which needs no exterior
// endpoint.
// It develops the loops on every call. A caller holding a topo.Face should reach it through
// [faceTrimUVOf] instead, which memoizes that development on the face — the projection does not depend
// on the query point, and on a B-spline surface it costs a NURBS inversion per loop sample (#3477).
// This entry stays for the synthesized curvedFaces the curved boolean classifies before any topo.Face
// exists for them.
func pointInTrimUV(f curvedFace, p math.Point3) bool {
	return developFaceTrim(f).contains(p)
}

// castAxis picks the parameter axis the even-odd containment ray travels along toward "outside": a
// NON-periodic axis with an EXTERIOR ENDPOINT — either unbounded (a plane's u/v, a cylinder/cone's v,
// exterior at ±∞) or bounded by a REGULAR domain edge (a NURBS patch's [uLo,uHi]×[vLo,vHi], exterior
// beyond the edge). It prefers v. ok is false when neither axis qualifies — a sphere (latitude is
// bounded by a degenerate POLE, not a boundary) or a torus (periodic both ways) — for which the caller
// uses the pole-free geodesic winding, since a ray toward a pole or around a wrap has no exterior end.
func castAxis(s geom.Surface, uPer, vPer bool) (alongV, ok bool) {
	if lo, hi := s.VDomain(); !vPer && axisHasExterior(s, lo, hi, true) {
		return true, true
	}
	if lo, hi := s.UDomain(); !uPer && axisHasExterior(s, lo, hi, false) {
		return false, true
	}
	return false, false
}

// axisHasExterior reports whether casting along this non-periodic axis reaches a point outside the
// surface: an unbounded domain reaches ±∞, and a bounded domain reaches its upper edge — provided that
// edge is a regular boundary, not a degenerate pole where the surface collapses to a point.
func axisHasExterior(s geom.Surface, lo, hi float64, alongV bool) bool {
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return true
	}
	return !edgeIsPole(s, hi, alongV)
}

// edgeIsPole reports whether the surface degenerates to a single point along the given domain edge —
// the sphere's ±π/2 latitude, where every cross-axis parameter maps to one pole. It measures the
// cross-axis span AT the edge against the span one step INTO the domain: a pole collapses the former to
// ~0 while the latter stays finite, so the ratio is the scale-free degeneracy signal.
func edgeIsPole(s geom.Surface, edge float64, alongV bool) bool {
	clo, chi := s.UDomain()
	if !alongV {
		clo, chi = s.VDomain()
	}
	if stdmath.IsInf(clo, 0) || stdmath.IsInf(chi, 0) {
		return false // an unbounded cross axis (a plane) never collapses to a point
	}
	a, b := clo+0.25*(chi-clo), clo+0.75*(chi-clo)
	inside := edge - 0.1*edgeStep(s, alongV)
	spanEdge := edgePoint(s, a, edge, alongV).DistanceTo(edgePoint(s, b, edge, alongV))
	spanIn := edgePoint(s, a, inside, alongV).DistanceTo(edgePoint(s, b, inside, alongV))
	return float64(spanEdge) < 1e-6*float64(spanIn) // tol:numeric — cross-axis span collapse ratio (dimensionless)
}

// edgeStep is a small step into the domain from an edge, along the cast axis, for the pole probe.
func edgeStep(s geom.Surface, alongV bool) float64 {
	lo, hi := s.VDomain()
	if !alongV {
		lo, hi = s.UDomain()
	}
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return 1
	}
	return hi - lo
}

// edgePoint evaluates the surface at cross-axis param c and cast-axis param t.
func edgePoint(s geom.Surface, c, t float64, alongV bool) math.Point3 {
	if alongV {
		return s.PointAt(c, t)
	}
	return s.PointAt(t, c)
}

// loopRayCrossings counts how many times a ray from q along the chosen non-periodic axis crosses one
// trim loop. The query is first shifted into that loop's own parameter branch (loops unwrap
// independently), and a loop that WRAPS the periodic cross-axis a full turn — a tunnel-wall end curve,
// a wrapped-pad rim — is left OPEN, since closing it with a seam-spanning chord would add a spurious
// second crossing and cancel the real one. This even-odd count over each loop is why a strip face
// (two non-nested boundary loops) and a nested outer+hole face both classify correctly.
func loopRayCrossings(q math.Point2, poly []math.Point2, uPer, vPer, alongV bool) int {
	c := loopCentroid(poly)
	qu, qv := q.X, q.Y
	if uPer {
		qu = unwrapAzimuthNear(c.X, qu)
	}
	if vPer {
		qv = unwrapAzimuthNear(c.Y, qv)
	}
	closed := !loopWrapsCrossAxis(poly, uPer, vPer, alongV)
	if !closed {
		// A wrapping ring covers exactly one period of the cross axis between its samples and the
		// closing image; the query is reduced into THAT span, not to the branch nearest the centroid,
		// which leaves a one-step window on either end where the ray misses the ring (ADR-0060).
		qu, qv = reduceIntoRingSpan(poly, math.P2(qu, qv), alongV)
	}
	return rayCrossingCount(math.P2(qu, qv), poly, closed, alongV)
}

// reduceIntoRingSpan moves the query's cross-axis coordinate by whole turns into the span the wrapping
// ring covers: from the lower of its extreme sample and closing image, one period up.
func reduceIntoRingSpan(poly []math.Point2, q math.Point2, alongV bool) (u, v math.Scalar) {
	image := closingImage(poly, false, alongV)
	lo := stdmath.Inf(1)
	for _, p := range append(append([]math.Point2{}, poly...), image) {
		if alongV {
			lo = stdmath.Min(lo, float64(p.X))
		} else {
			lo = stdmath.Min(lo, float64(p.Y))
		}
	}
	reduce := func(x float64) float64 { return lo + stdmath.Mod(stdmath.Mod(x-lo, twoPi)+twoPi, twoPi) }
	if alongV {
		return math.Scalar(reduce(float64(q.X))), q.Y
	}
	return q.X, math.Scalar(reduce(float64(q.Y)))
}

// loopWrapsCrossAxis reports whether the loop makes a full 2π circuit of the periodic axis that is
// PERPENDICULAR to the cast direction (u when casting along v). Such a loop is an open curve spanning
// the whole periodic range, not a closed ring.
func loopWrapsCrossAxis(poly []math.Point2, uPer, vPer, alongV bool) bool {
	crossPeriodic := uPer
	net := poly[len(poly)-1].X - poly[0].X
	if !alongV {
		crossPeriodic = vPer
		net = poly[len(poly)-1].Y - poly[0].Y
	}
	return crossPeriodic && stdmath.Abs(stdmath.Abs(net)-2*stdmath.Pi) < 0.5 // tol:parametric — full-turn circuit
}

// rayCrossingCount counts crossings of a ray from q toward +axis (v when alongV, else u) with the
// polyline, standard crossing-number even-odd. An open polyline (closed=false) omits the last→first
// segment.
func rayCrossingCount(q math.Point2, poly []math.Point2, closed, alongV bool) int {
	n := len(poly)
	count := 0
	for i := 0; i+1 < n; i++ {
		if raySegmentCrosses(q, poly[i], poly[i+1], alongV) {
			count++
		}
	}
	if raySegmentCrosses(q, poly[n-1], closingImage(poly, closed, alongV), alongV) {
		count++
	}
	return count
}

// closingImage is the point the polyline's last sample closes onto: its first sample for a ring that
// closes, and for a ring that WRAPS the cross axis, that sample's periodic image one turn on — the
// sampling stops one step short of the full turn, and a ray through that last step must still cross
// the ring, or a band reads its own interior as outside there (ADR-0060).
func closingImage(poly []math.Point2, closed, alongV bool) math.Point2 {
	first, last := poly[0], poly[len(poly)-1]
	if closed {
		return first
	}
	if alongV {
		return math.P2(float64(first.X)+stdmath.Copysign(2*stdmath.Pi, float64(last.X-first.X)), float64(first.Y))
	}
	return math.P2(float64(first.X), float64(first.Y)+stdmath.Copysign(2*stdmath.Pi, float64(last.Y-first.Y)))
}

// raySegmentCrosses reports whether the ray from q toward +axis crosses segment ab. The segment must
// straddle q on the cross-axis, and its interpolated axis coordinate there must lie beyond q.
func raySegmentCrosses(q, a, b math.Point2, alongV bool) bool {
	ac, bc, qc := a.X, b.X, q.X // cross-axis (u) coordinates when casting along v
	ap, bp, qp := a.Y, b.Y, q.Y // ray-axis (v) coordinates
	if !alongV {
		ac, bc, qc, ap, bp, qp = a.Y, b.Y, q.Y, a.X, b.X, q.X
	}
	if (ac > qc) == (bc > qc) {
		return false // does not straddle q on the cross-axis
	}
	t := (qc - ac) / (bc - ac)
	return ap+t*(bp-ap) > qp
}

// loopToUV walks a loop's edges into a continuous (u, v) polyline: each sample's periodic parameter is
// shifted to the branch of the previous sample, so an edge crossing the u=0≡2π seam stays monotone
// rather than jumping a full turn. The walk is then repaired across a parametric POLE, where
// continuity alone picks the wrong branch (see bridgePoleBranch).
func loopToUV(s geom.Surface, loop curvedLoop, uPer, vPer bool) []math.Point2 {
	ring := unwrapLoopRing(s, loop, uPer, vPer)
	if uPer {
		ring = bridgePoleBranch(s, ring, false)
	}
	if vPer {
		ring = bridgePoleBranch(s, ring, true)
	}
	return ring
}

// unwrapLoopRing samples every edge of the loop in traversal order and shifts each sample's periodic
// parameters to the branch of the previous sample, so the polyline stays continuous across the seam.
func unwrapLoopRing(s geom.Surface, loop curvedLoop, uPer, vPer bool) []math.Point2 {
	var ring []math.Point2
	for _, e := range loop.edges {
		for k := 0; k < trimUVSamples; k++ {
			t := e.t0 + (e.t1-e.t0)*float64(k)/trimUVSamples
			u, v := s.ParamAt(e.curve.PointAt(t))
			u, v = continueUV(ring, u, v, uPer, vPer)
			ring = append(ring, math.P2(u, v))
		}
	}
	return ring
}

// bridgePoleBranch repairs a ring that fails to close along its periodic axis because the loop runs
// THROUGH a parametric pole — a cone apex, a sphere pole — where the surface collapses to a point and
// every periodic value names it. The loop's two sides of the slit are the SAME 3-D curve there, so
// continuity pins both to one branch and the ring degenerates to a zero-area sliver. That sliver made
// a full cone's side face integrate to no volume, which flipped the whole body's derived outward
// orientation and reported every INTERIOR point of the cone as outside
// (Oblikovati/Oblikovati#3447). A ring that closes, and a genuine full-turn circuit with no pole on
// it (a rim circle bounding a band), are both returned untouched.
func bridgePoleBranch(s geom.Surface, ring []math.Point2, periodicIsV bool) []math.Point2 {
	turns := ringMissingTurns(ring, periodicIsV)
	if turns == 0 {
		return ring
	}
	pole, found := poleSampleIndex(s, ring, periodicIsV)
	if !found {
		return ring
	}
	return rebranchRingTail(ring, pole, turns*2*stdmath.Pi, periodicIsV)
}

// ringMissingTurns is the whole number of turns the ring's periodic coordinate is short of returning
// to its start. It is 0 for a ring that closes in the plane and ±1 for a loop that circles the seam
// once — the pole probe, not this count, is what tells those two apart.
func ringMissingTurns(ring []math.Point2, periodicIsV bool) float64 {
	if len(ring) < 2 {
		return 0
	}
	gap := ring[0].X - ring[len(ring)-1].X
	if periodicIsV {
		gap = ring[0].Y - ring[len(ring)-1].Y
	}
	return stdmath.Round(gap / (2 * stdmath.Pi))
}

// poleSampleIndex returns the first ring sample sitting on a parametric pole. Only the first is
// sought: a loop that crosses two poles (a full lune) closes in the plane and never reaches here.
func poleSampleIndex(s geom.Surface, ring []math.Point2, periodicIsV bool) (int, bool) {
	for i, q := range ring {
		if sampleOnPole(s, q, periodicIsV) {
			return i, true
		}
	}
	return 0, false
}

// poleTangentRatio accepts a sample as a pole when its tangent along the PERIODIC axis has collapsed
// against the transverse one — the same scale-free degeneracy signal edgeIsPole reads off a domain
// edge, taken here at an interior parameter (a cone's apex is not a domain edge: its v is unbounded).
const poleTangentRatio = 1e-9 // tol:numeric — periodic:transverse tangent ratio at a pole (dimensionless)

// sampleOnPole reports whether the surface degenerates at (u, v) along its periodic axis, so that
// axis names no direction there and the loop's branch across the point is free.
func sampleOnPole(s geom.Surface, q math.Point2, periodicIsV bool) bool {
	du, dv := s.DerivativesAt(q.X, q.Y)
	along, across := du, dv
	if periodicIsV {
		along, across = dv, du
	}
	return float64(along.Length()) < poleTangentRatio*float64(across.Length())
}

// rebranchRingTail shifts the ring from the pole sample onward by delta, and keeps the pole sample on
// BOTH branches so the traverse along the pole isoline becomes an explicit polygon edge instead of a
// chord cutting across the region.
func rebranchRingTail(ring []math.Point2, pole int, delta float64, periodicIsV bool) []math.Point2 {
	out := make([]math.Point2, 0, len(ring)+1)
	out = append(out, ring[:pole+1]...)
	for _, q := range ring[pole:] {
		if periodicIsV {
			out = append(out, math.P2(q.X, q.Y+delta))
			continue
		}
		out = append(out, math.P2(q.X+delta, q.Y))
	}
	return out
}

// continueUV shifts (u, v) by whole turns so each periodic coordinate stays within half a turn of the
// previous sample, keeping the projected loop continuous across the seam.
func continueUV(ring []math.Point2, u, v float64, uPer, vPer bool) (float64, float64) {
	if len(ring) == 0 {
		return u, v
	}
	last := ring[len(ring)-1]
	if uPer {
		u = unwrapAzimuthNear(last.X, u)
	}
	if vPer {
		v = unwrapAzimuthNear(last.Y, v)
	}
	return u, v
}

// surfacePeriodic reports which of the surface's parameter directions wrap around a full 2π turn.
func surfacePeriodic(s geom.Surface) (uPer, vPer bool) {
	return domainPeriodic(s.UDomain()), domainPeriodic(s.VDomain())
}

// domainPeriodic reports whether a parameter domain is a finite full turn [lo, lo+2π].
func domainPeriodic(lo, hi float64) bool {
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return false
	}
	return stdmath.Abs((hi-lo)-2*stdmath.Pi) < domainPeriodTol
}

// loopCentroid returns the average of a ring's (u, v) vertices — the branch reference the query point
// is unwrapped toward, so it lands in the same turn as the loop rather than a neighbouring one.
func loopCentroid(ring []math.Point2) math.Point2 {
	var su, sv float64
	for _, q := range ring {
		su += q.X
		sv += q.Y
	}
	n := float64(len(ring))
	return math.P2(su/n, sv/n)
}

// faceBoundaryBand is twice the largest gap between a trim edge and its sampled polyline (the chord
// sagitta over one trimUVSamples segment). Inside this band of a curved trim edge the projected
// polygon may misclassify a point, so the classifier reselects the ray rather than trust the polygon;
// beyond it the sampled trim is reliable. A face with only straight edges (a planar polygon, a
// cylinder/cone band whose bounds are circles-as-v-const) reports 0.
func faceBoundaryBand(f curvedFace) float64 {
	worst := 0.0
	for _, loop := range f.loops {
		for _, e := range loop.edges {
			if s := edgeChordSagitta(e); s > worst {
				worst = s
			}
		}
	}
	return 2 * worst
}

// edgeChordSagitta is the largest distance from an edge's true midpoint to its chord midpoint over
// the trimUVSamples segments the trim polygon uses — the polygon's worst deviation from the edge.
func edgeChordSagitta(e loopEdge) float64 {
	worst := 0.0
	for k := 0; k < trimUVSamples; k++ {
		t0 := e.t0 + (e.t1-e.t0)*float64(k)/trimUVSamples
		t1 := e.t0 + (e.t1-e.t0)*float64(k+1)/trimUVSamples
		chordMid := e.curve.PointAt(t0).Midpoint(e.curve.PointAt(t1))
		if s := float64(e.curve.PointAt((t0 + t1) / 2).DistanceTo(chordMid)); s > worst {
			worst = s
		}
	}
	return worst
}
