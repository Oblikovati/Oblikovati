// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Placing a chart's artificial seam clear of the imprint, EXACTLY.
//
// The seam was placed in the widest gap between the imprint's SAMPLES, which is a resolution wall: two
// lens loops on a wall that nearly pinch leave a corridor between their tips narrower than one sample
// step, the widest sampled gap is then somewhere inside a lens, and the seam runs through the loop it was
// meant to avoid (ADR-0061 stage 4). What the seam must avoid is the azimuth EXTENT of every imprint
// curve, and an extent is bounded by the curve's turning points in azimuth — where its tangent runs along
// the ruling — which are found by bracketing the sampled azimuth's extrema and refining each to rounding.
// Between the extents lies the corridor, however narrow; the seam goes to the middle of its widest part,
// clear of every frame vertex. Where the imprint covers every azimuth — a rod's chart, whose loops wrap —
// there is no corridor, the seam has to cross, and it crosses as far from every turning point and frame
// vertex as it can, where the crossing is transversal and its solve is well-conditioned.

// azimuthArc is a run of azimuths an imprint curve covers, in the unwrapped coordinate (hi ≥ lo).
type azimuthArc struct{ lo, hi float64 }

// exactSeamAzimuth returns the seam coordinate for one periodic chart axis (coord reads it from a chart
// point): the middle of the widest stretch of azimuth free of the imprint and of the frame's own
// coordinates, or, when the imprint leaves no azimuth free, the middle of the widest gap between the
// imprint's turning points and the frame's coordinates. The chart's seams must still be at their
// origin when this runs, so the coordinates read are the surface's own.
func (c *loopFrame) exactSeamAzimuth(imprint []geom.Curve3, coord func(math.Point2) float64) float64 {
	forbidden := c.frameCoordinates(coord)
	var covered []azimuthArc
	for _, cv := range imprint {
		arcs, turns := c.azimuthCover(cv, coord)
		covered = append(covered, arcs...)
		forbidden = append(forbidden, turns...)
	}
	if u, ok := widestFreeGapMid(covered, forbidden); ok {
		return u
	}
	return widestGapMid(forbidden)
}

// frameCoordinates reads one chart coordinate at both ends of every frame edge.
func (c *loopFrame) frameCoordinates(coord func(math.Point2) float64) []float64 {
	var out []float64
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			out = append(out, coord(c.host.paramOf(e.start())), coord(c.host.paramOf(e.end())))
		}
	}
	return out
}

// azimuthCover returns the azimuth arcs one imprint curve covers and its turning azimuths. A straight
// curve on these charts is a ruling: it covers one azimuth, which is also where a seam must not go.
func (c *loopFrame) azimuthCover(cv geom.Curve3, coord func(math.Point2) float64) (arcs []azimuthArc, turns []float64) {
	lo, hi := cv.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return nil, nil
	}
	at := func(t float64) float64 { return coord(c.host.paramOf(cv.PointAt(t))) }
	if geom.IsStraightCurve(cv) {
		u := at((lo + hi) / 2)
		return []azimuthArc{{u, u}}, []float64{u}
	}
	ts, us := unwrappedStations(at, lo, hi)
	closed := geom.CurveIsClosed(cv)
	if closed && stdmath.Abs(us[len(us)-1]-us[0]) >= stdmath.Pi {
		// The curve winds the azimuth and covers every value; the seam then has to cross it, and must not
		// do so at its closure point, where no sign change brackets the crossing.
		return []azimuthArc{{0, twoPi}}, []float64{us[0]}
	}
	turnUs := azimuthTurns(at, ts, us, closed)
	// Between consecutive turning points the azimuth is monotone, so each run covers exactly the arc
	// between its ends; the runs from the curve's start to its first turn and from its last turn to its
	// end are arcs too. The ends themselves are forbidden to a seam as well: a seam through a curve's
	// endpoint — a frame incidence, or a closed loop's closure point — meets it where no sign change
	// brackets the crossing, so it would go unsolved.
	ends := append(append([]float64{us[0]}, turnUs...), us[len(us)-1])
	for i := 1; i < len(ends); i++ {
		arcs = append(arcs, azimuthArc{stdmath.Min(ends[i-1], ends[i]), stdmath.Max(ends[i-1], ends[i])})
	}
	return arcs, append(turnUs, us[0], us[len(us)-1])
}

// unwrappedStations samples the azimuth at crossingSpanSamples stations over [lo, hi], each carried onto
// the branch of the one before, so the sequence is continuous across the seam.
func unwrappedStations(at func(float64) float64, lo, hi float64) (ts, us []float64) {
	ts = make([]float64, crossingSpanSamples+1)
	us = make([]float64, crossingSpanSamples+1)
	for i := range ts {
		ts[i] = lo + (hi-lo)*float64(i)/crossingSpanSamples
		us[i] = at(ts[i])
		if i > 0 {
			us[i] = unwrapAzimuthNear(us[i-1], us[i])
		}
	}
	return ts, us
}

// azimuthTurns finds every station where the sampled azimuth changes direction and refines each to the
// exact extremum between its neighbours, returning the turning azimuths in station order. A CLOSED
// curve's stations are read round the closure too — its first station is a turning point as often as
// any other — with the bracket's parameters folded back into the domain.
func azimuthTurns(at func(float64) float64, ts, us []float64, closed bool) []float64 {
	n := len(us) - 1 // stations 0..n, station n the closure of station 0
	var turns []float64
	last := n - 1
	if closed {
		last = n
	}
	for i := 1; i <= last; i++ {
		next := i + 1
		if next > n {
			next = 1 // round the closure: station n is station 0
		}
		before, after := us[i]-us[i-1], unwrapAzimuthNear(us[i], us[next])-us[i]
		if before*after < 0 {
			turns = append(turns, refineTurn(at, ts, us[i], i-1, next, before > 0))
		}
	}
	return turns
}

// refineTurn refines one bracketed turning point — the azimuth is extremal between stations prev and
// next — to rounding, on the branch of the middle station, with a bracket that may run round a closed
// curve's closure folded back into the domain.
func refineTurn(at func(float64) float64, ts []float64, branch float64, prev, next int, wantMax bool) float64 {
	n := len(ts) - 1
	span := ts[n] - ts[0]
	tPrev, tNext := ts[prev], ts[next]
	if next <= prev {
		tNext += span
	}
	g := func(t float64) float64 {
		if t > ts[n] {
			t -= span
		}
		return unwrapAzimuthNear(branch, at(t))
	}
	return g(geom.ExtremumOnBracket(g, tPrev, tNext, wantMax))
}

// widestFreeGapMid returns the middle of the widest stretch of the circle that no covered arc reaches
// and no forbidden point lies in; ok is false when the arcs cover the whole circle.
func widestFreeGapMid(covered []azimuthArc, forbidden []float64) (float64, bool) {
	free := freeArcs(covered)
	if len(free) == 0 {
		return 0, false
	}
	points := make([]float64, 0, len(forbidden))
	for _, f := range forbidden {
		points = append(points, wrapToPeriod(f))
	}
	sort.Float64s(points)
	bestLen, bestMid := -1.0, 0.0
	for _, arc := range free {
		for _, gap := range splitAzimuthArcAt(arc, points) {
			if l := gap.hi - gap.lo; l > bestLen {
				bestLen, bestMid = l, (gap.lo+gap.hi)/2
			}
		}
	}
	return wrapToPeriod(bestMid), bestLen >= 0
}

// freeArcs is the complement of the covered arcs on the circle, as arcs in [0, 2π) that may run past 2π
// when they cross the origin. Nothing covered means the whole circle is free.
func freeArcs(covered []azimuthArc) []azimuthArc {
	parts, full := arcsOnCircle(covered)
	if full {
		return nil
	}
	if len(parts) == 0 {
		return []azimuthArc{{0, twoPi}}
	}
	merged := mergeArcs(parts)
	var free []azimuthArc
	for i := range merged {
		next := merged[(i+1)%len(merged)].lo
		if i == len(merged)-1 {
			next += twoPi // the gap from the last arc round to the first crosses the origin
		}
		if next > merged[i].hi {
			free = append(free, azimuthArc{merged[i].hi, next})
		}
	}
	return free
}

// arcsOnCircle folds arcs onto [0, 2π), splitting one that crosses the origin in two; full reports an
// arc that covers the whole circle.
func arcsOnCircle(covered []azimuthArc) (parts []azimuthArc, full bool) {
	for _, a := range covered {
		if a.hi-a.lo >= twoPi {
			return nil, true
		}
		lo := wrapToPeriod(a.lo)
		hi := lo + (a.hi - a.lo)
		if hi > twoPi {
			parts = append(parts, azimuthArc{lo, twoPi}, azimuthArc{0, hi - twoPi})
		} else {
			parts = append(parts, azimuthArc{lo, hi})
		}
	}
	return parts, false
}

// mergeArcs sorts arcs by start and joins the ones that overlap or touch.
func mergeArcs(parts []azimuthArc) []azimuthArc {
	sort.Slice(parts, func(i, j int) bool { return parts[i].lo < parts[j].lo })
	merged := []azimuthArc{parts[0]}
	for _, p := range parts[1:] {
		if last := &merged[len(merged)-1]; p.lo <= last.hi {
			last.hi = stdmath.Max(last.hi, p.hi)
		} else {
			merged = append(merged, p)
		}
	}
	return merged
}

// splitArcAt cuts an arc at every forbidden point inside it, the points taken on the arc's own branch.
func splitAzimuthArcAt(arc azimuthArc, points []float64) []azimuthArc {
	cuts := []float64{arc.lo}
	for _, p := range points {
		for _, q := range []float64{p, p + twoPi} {
			if q > arc.lo && q < arc.hi {
				cuts = append(cuts, q)
			}
		}
	}
	sort.Float64s(cuts)
	cuts = append(cuts, arc.hi)
	out := make([]azimuthArc, 0, len(cuts)-1)
	for i := 1; i < len(cuts); i++ {
		out = append(out, azimuthArc{cuts[i-1], cuts[i]})
	}
	return out
}
