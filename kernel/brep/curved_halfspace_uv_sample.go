// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Phase 1 of the ruled-side (u,v) arrangement (#1405, split out of
// curved_halfspace_uv_arrangement.go for #2212): project the imprint curves into the side's
// (u,v) parameter space and sample them into TAGGED segments.
//
// Tagging is the point: each segment remembers which curve it came from, so the boundary the
// last phase re-emits carries the exact arc or conic rather than the sampled polyline. Seam
// handling also lives here — a segment crossing u=2pi=0 (or the tube seam v=2pi=0 on a torus)
// is split at the seam so no segment spans the wrap.

// paramOf returns the (u, v) = (azimuth, axial distance) of a 3D point on the ruled side — the inverse of
// point3. v is the signed distance along the axis from the v=0 base; the radial remainder gives the azimuth
// in [0, 2π) measured from the ref direction. A point off the surface projects to its nearest (u,v) (the
// axial/azimuth components are still well defined), which is what sampling an imprint that lies ON the
// surface to tolerance needs.
func (c ruledUV) paramOf(p math.Point3) math.Point2 {
	d := c.base.VectorTo(p)
	v := float64(d.Dot(c.axis))
	radial := d.Sub(c.axis.Scale(math.Scalar(v)))
	// atan2 ∈ [−π, π] and seamU ∈ [0, 2π), so u ∈ [−3π, π]: only a lower wrap is needed. (An upper clamp
	// would map a point at azimuth ≈2π−ε, which rounds to 2π after the wrap, down to 0 — a false seam
	// discontinuity that leaves an un-split wrap segment.)
	u := stdmath.Atan2(float64(radial.Dot(c.binor)), float64(radial.Dot(c.ref))) - c.seamU
	for u < 0 {
		u += 2 * stdmath.Pi
	}
	return math.P2(u, v)
}

// segKind tags what a (u,v) segment re-emits to when it bounds a kept cell: an imprint sub-arc (the cut),
// a band rim (constant v, a circle arc), the azimuth SEAM ruling (u=0≡2π — an ARTIFICIAL edge that
// closes the parameter rectangle for the arrangement and dissolves wherever the kept region wraps it), or a
// non-periodic face-POLYGON boundary edge (a bounded plane's real analytic frame — see planeUV, #1591).
type segKind int

const (
	segImprint segKind = iota
	segRim
	segSeam
	// segPolygon frames a non-periodic planar face by its boundary edges instead of rim circles + a seam.
	// Unlike segRim (constant v ⇒ equal-v means same run) each polygon edge is a distinct analytic curve, so
	// sameRun compares curve identity like segImprint — two polygon edges at the same v must NOT merge (#1591).
	segPolygon
)

// uvSeg is one sampled segment of an imprint curve in the side's (u,v) band, tagged with the analytic
// curve it came from and the curve parameters at its endpoints, so the arrangement can re-emit the exact
// curve sub-arc (an ellipse/hyperbola/parabola arm) for a boundary chain rather than the sampled polyline.
type uvSeg struct {
	a, b   math.Point2 // (u,v) endpoints
	curve  geom.Curve3 // the source analytic curve this segment re-emits to
	tA, tB float64     // its parameters at a and b
	kind   segKind
}

// imprintSampleCount is the number of segments one analytic imprint curve is sampled into across its
// parameter domain. It is dense enough to resolve the curve's crossings with the rims and with other
// imprint arcs to the arrangement weld (1e-7) at part scale, while keeping the planar arrangement small.
const imprintSampleCount = 256

// sampleImprintUV samples an analytic imprint curve over its whole domain into tagged (u,v) segments on
// the side. Each sample inverts the 3D curve point to (u,v) via paramOf; consecutive samples become one
// uvSeg carrying the curve and its endpoint parameters, so a later boundary walk re-emits exact sub-arcs.
// The azimuth seam (u wrapping 2π↔0) is left for the arrangement to resolve — sampleImprintUV reports the
// raw branch [0,2π); seam unwrapping is applied when segments are assembled into the band.
func (c ruledUV) sampleImprintUV(curve geom.Curve3) []uvSeg {
	var segs []uvSeg
	for _, r := range c.clipParams(curve) {
		segs = append(segs, c.sampleRange(curve, r[0], r[1])...)
	}
	return segs
}

// sampleRange samples one curve parameter interval [t0,t1] into tagged (u,v) segments. An SSI imprint
// arrives as a *geom.Polyline whose vertices the tracer already placed adaptively (dense where the curve
// bends — e.g. a near-pinch neck); RE-sampling it at a fixed imprintSampleCount would DECIMATE that
// adaptive detail and misplace the arrangement's rim/seam crossings near the pinch, collapsing a lens cap
// (Oblikovati#1781). So for a polyline dense enough to already meet the sampling contract, the arrangement
// consumes its OWN vertices (spacing-independent); a sparse/synthetic polyline still densifies to the fixed
// count so consumers that assume a dense azimuth cover (chooseSeamU, imprintWrapsAzimuth) are unaffected.
func (c ruledUV) sampleRange(curve geom.Curve3, t0, t1 float64) []uvSeg {
	if t0 == t1 {
		return nil
	}
	if pl, ok := curve.(*geom.Polyline); ok && len(pl.Vertices) >= imprintSampleCount {
		return c.sampleVertices(pl, t0, t1)
	}
	return c.emitSegments(curve, curveOwnParams(curve, t0, t1))
}

// curveOwnParams is an analytic curve's OWN sampling grid, restricted to (t0, t1) and closed with that
// range's endpoints — the same shape sampleVertices gives a polyline, and for the same reason.
//
// The two sides of an imprint clip the SAME curve to DIFFERENT parameter ranges (each to its own band),
// so spreading a fixed count of samples across each range independently places the shared edge's points
// somewhere different on each side, and curvedStitch cannot fuse two boundaries that do not share
// vertices: a crossing-cylinder intersect came back as three faces in THREE shells rather than one
// (Oblikovati/Oblikovati#3489). Anchoring the grid to the curve's whole domain makes the sample set a
// function of the CURVE, so wherever the two ranges overlap the points coincide exactly.
func curveOwnParams(curve geom.Curve3, t0, t1 float64) []float64 {
	lo, hi := curve.Domain()
	ts := []float64{t0}
	if !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
		for i := 0; i <= imprintSampleCount; i++ {
			if t := lo + (hi-lo)*float64(i)/imprintSampleCount; t > t0 && t < t1 {
				ts = append(ts, t)
			}
		}
	}
	if len(ts) == 1 { // an unbounded curve has no domain grid to anchor to: space the range itself
		for i := 1; i < imprintSampleCount; i++ {
			ts = append(ts, t0+(t1-t0)*float64(i)/imprintSampleCount)
		}
	}
	return append(ts, t1)
}

// emitSegments turns an ordered parameter list into the tagged (u, v) segments the arrangement reads.
func (c ruledUV) emitSegments(curve geom.Curve3, ts []float64) []uvSeg {
	segs := make([]uvSeg, 0, len(ts))
	prevT, prevP := ts[0], c.paramOf(curve.PointAt(ts[0]))
	for _, t := range ts[1:] {
		p := c.paramOf(curve.PointAt(t))
		segs = append(segs, uvSeg{a: prevP, b: p, curve: curve, tA: prevT, tB: t, kind: segImprint})
		prevT, prevP = t, p
	}
	return segs
}

// sampleVertices emits one uvSeg per ACTUAL polyline vertex in [t0,t1], preserving the tracer's adaptive
// point placement instead of re-chording it at a fixed stride (Oblikovati#1781). The polyline is uniform in
// index, so vertex j sits at t = j/(n−1); vertices strictly inside (t0,t1) are emitted between the range
// endpoints. The endpoints are kept exact so a clipped sub-range still starts/ends where the caller asked.
func (c ruledUV) sampleVertices(pl *geom.Polyline, t0, t1 float64) []uvSeg {
	n := len(pl.Vertices)
	ts := []float64{t0}
	for j := range n {
		if vt := float64(j) / float64(n-1); vt > t0 && vt < t1 {
			ts = append(ts, vt)
		}
	}
	ts = append(ts, t1)
	segs := make([]uvSeg, 0, len(ts))
	prevT, prevP := ts[0], c.paramOf(pl.PointAt(ts[0]))
	for _, t := range ts[1:] {
		p := c.paramOf(pl.PointAt(t))
		segs = append(segs, uvSeg{a: prevP, b: p, curve: pl, tA: prevT, tB: t, kind: segImprint})
		prevT, prevP = t, p
	}
	return segs
}

// clipParams returns the parameter sub-ranges over which an imprint curve should be sampled: its whole
// domain when finite (an ellipse / clipped arc — clipSegToVBand then trims any out-of-band part), but for an
// UNBOUNDED curve (a ruling line, or an open hyperbola/parabola of a cone cut) only the sub-ranges where it
// lies within the band's axial extent. A cone cut's hyperbola has TWO arms in the band (the joining vertex
// is outside it), so this can return more than one range — each a separate boundary arc.
func (c ruledUV) clipParams(curve geom.Curve3) [][2]float64 {
	lo, hi := curve.Domain()
	if !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
		return [][2]float64{{lo, hi}}
	}
	if line, isLine := curve.(geom.Line); isLine {
		d := float64(line.Dir.AsVector().Dot(c.axis))
		if stdmath.Abs(d) < 1e-12 {
			return nil // a ruling perpendicular to the axis cannot bound a v-band
		}
		v0 := float64(c.base.VectorTo(line.Origin).Dot(c.axis))
		return [][2]float64{{(c.band.vMin - v0) / d, (c.band.vMax - v0) / d}}
	}
	return c.inBandRanges(curve)
}

// inBandRanges scans an unbounded curve over a finite bracket (proportional to the band's 3D size) and
// returns every maximal sub-range whose axial coordinate v lies within the band. Each range straddles its
// band crossings by one sample so clipSegToVBand can refine the exact crossing; multiple ranges arise when
// the curve enters the band more than once (the two arms of a cone-cut hyperbola).
func (c ruledUV) inBandRanges(curve geom.Curve3) [][2]float64 {
	b := 4 * (c.band.vMax - c.band.vMin + 2*stdmath.Max(c.band.rBot, c.band.rTop) + 1)
	const scan = 2048
	inBand := func(t float64) bool {
		v := c.curveV(curve, t)
		return v >= c.band.vMin && v <= c.band.vMax
	}
	var ranges [][2]float64
	open, start, prev := false, 0.0, -b
	for i := 0; i <= scan; i++ {
		t := -b + 2*b*float64(i)/scan
		switch in := inBand(t); {
		case in && !open:
			start, open = prev, true
		case !in && open:
			ranges = append(ranges, [2]float64{start, t})
			open = false
		}
		prev = t
	}
	if open {
		ranges = append(ranges, [2]float64{start, b})
	}
	return ranges
}

// unwrapAzimuthNear shifts x by whole turns (2π) so it lands within ±π of ref, turning a wrapped azimuth
// pair into a monotone run — so a segment's true u-extent (and whether it crosses the seam) is unambiguous.
func unwrapAzimuthNear(ref, x float64) float64 {
	for x-ref > stdmath.Pi {
		x -= 2 * stdmath.Pi
	}
	for x-ref < -stdmath.Pi {
		x += 2 * stdmath.Pi
	}
	return x
}

// splitSeamCrossing splits an imprint segment whose endpoints straddle the AZIMUTH seam (u=0≡2π) into two
// segments meeting AT the seam, so no segment spans the discontinuity — the arrangement sees a clean
// parameter rectangle. A segment that does not straddle the seam is returned unchanged.
func splitSeamCrossing(s uvSeg) []uvSeg { return splitPeriodicSeam(s, true) }

// splitVSeamCrossing is the v (tube-angle) analogue for a torus: it splits a segment straddling the TUBE seam
// (v=0≡2π), needed when the imprint wraps the tube period (the two-oval band's ovals span all v, so the v-seam
// is crossed and cannot be placed clear of it). The ruled sides are bounded in v, so they never use it (#1406).
func splitVSeamCrossing(s uvSeg) []uvSeg { return splitPeriodicSeam(s, false) }

// splitPeriodicSeam splits an imprint segment that straddles a periodic seam on the chosen coordinate (onU:
// the azimuth u=X, else the tube angle v=Y) into two segments meeting at the seam. The OTHER coordinate and
// the curve parameter are interpolated to the crossing; a non-straddling segment is returned unchanged.
func splitPeriodicSeam(s uvSeg, onU bool) []uvSeg {
	ca, oa := float64(s.a.X), float64(s.a.Y)
	cb, ob := float64(s.b.X), float64(s.b.Y)
	if !onU {
		ca, oa, cb, ob = oa, ca, ob, cb // wrap on Y; the other coordinate is X
	}
	cu := unwrapAzimuthNear(ca, cb)
	if cu >= 0 && cu <= 2*stdmath.Pi {
		return []uvSeg{s} // no seam crossing on this coordinate
	}
	seam, other := 0.0, 2*stdmath.Pi
	if cu > 2*stdmath.Pi { // the run climbs past 2π: a → 2π, then 0 → b
		seam, other = 2*stdmath.Pi, 0
	}
	f := (seam - ca) / (cu - ca)
	oSeam := oa + f*(ob-oa)
	tSeam := s.tA + f*(s.tB-s.tA)
	return []uvSeg{
		{a: s.a, b: seamPoint(seam, oSeam, onU), curve: s.curve, tA: s.tA, tB: tSeam, kind: s.kind},
		{a: seamPoint(other, oSeam, onU), b: s.b, curve: s.curve, tA: tSeam, tB: s.tB, kind: s.kind},
	}
}

// seamPoint builds the (u,v) split point from the seam coordinate and the interpolated other coordinate,
// placing them on the right axes (onU: seam is u=X; else seam is v=Y).
func seamPoint(seamCoord, otherCoord float64, onU bool) math.Point2 {
	if onU {
		return math.P2(seamCoord, otherCoord)
	}
	return math.P2(otherCoord, seamCoord)
}
