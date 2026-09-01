// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The EXACT half of the curved imprint (#3489). An imprint loop is what every downstream face
// boundary is built from, so its exactness is the ceiling on the result body's exactness: a marched
// chord loop caps a boolean's boundary — and therefore its analytic mass properties — at the trace's
// achieved deviation, whatever the integrator does afterwards. Where the surface pair has a closed
// form (geom.RuledQuadricSection: a straight-ruled surface substituted into the other's quadric) the
// imprint takes the exact section instead, and the edges stitched from it report zero achieved
// tolerance. Where it does not, the marcher runs exactly as before.

// exactImprintLoops returns base∩other as EXACT closed section curves, clipped to the same marching
// window the tracer would have used, or ok=false to leave the pair to the marcher. Its clipping
// contract is the tracer's: a loop wholly inside the window is kept, a loop wholly outside is dropped
// (the tracer never finds it), and a loop the window cuts OPEN is refused here as a whole — the
// tracer's open chain carries a diagnostic and a drop that only it can report, so the pair falls back
// rather than being silently reproduced.
func exactImprintLoops(base, other geom.Surface, window geom.SurfaceGrid, res geom.Resolution) ([]geom.Curve3, bool) {
	arcs, ok := geom.RuledQuadricSection(base, other, res)
	if !ok {
		arcs, ok = geom.RuledQuadricSection(other, base, res) // the wrap form may live on the other chart
	}
	if !ok || !windowSpansFullAzimuth(base, window) {
		return nil, false
	}
	kept := make([]geom.Curve3, 0, len(arcs))
	for _, arc := range arcs {
		inside, whole := loopInsideWindow(base, arc, window, res)
		if !whole {
			return nil, false // the window cuts this loop open: the marched path owns the clipped chain
		}
		if inside {
			kept = append(kept, arc)
		}
	}
	if len(kept) == 0 {
		return nil, false
	}
	return kept, true
}

// windowSpansFullAzimuth reports that the marching window leaves the base's periodic u untouched — the
// only case in which "the section wraps every azimuth" describes the whole clipped intersection.
// geom.SurfaceWindow only ever narrows an UNBOUNDED direction, so a ruled side's u always qualifies;
// asserting it keeps the exact path honest if that ever changes.
func windowSpansFullAzimuth(base geom.Surface, window geom.SurfaceGrid) bool {
	lo, hi := base.UDomain()
	return window.UMin == lo && window.UMax == hi
}

// loopInsideWindow reports whether a closed section loop lies inside the base's v window, and whether
// it lies WHOLLY on one side of it. It reads v through base.ParamAt, so it answers for the base's own
// chart even when the arc is evaluated on the other surface's.
func loopInsideWindow(base geom.Surface, arc geom.Curve3, window geom.SurfaceGrid, res geom.Resolution) (inside, whole bool) {
	in, out := 0, 0
	for i := range exactImprintWindowProbes {
		p := arc.PointAt(float64(i) / exactImprintWindowProbes)
		if vInWindow(base, p, window, res.Weld()) {
			in++
			continue
		}
		out++
	}
	return in > 0, in == 0 || out == 0
}

// vInWindow reports whether a point's base-chart v sits inside the window, with a weld-sized slack so a
// loop that merely grazes the band boundary counts as inside rather than as a straddle.
func vInWindow(base geom.Surface, p math.Point3, window geom.SurfaceGrid, slack float64) bool {
	_, v := base.ParamAt(p)
	return v >= window.VMin-slack && v <= window.VMax+slack
}

// exactImprintWindowProbes is how many points of a section loop the window test reads. The loops are
// smooth closed curves at this stage (the conditioning gate refused every folded one), so this resolves
// a band crossing far finer than the arrangement's own imprint sampling.
const exactImprintWindowProbes = 512

// imprintLoopPoints returns an imprint loop's vertex chain: a marched loop's OWN vertices (which the
// tracer placed adaptively — re-chording them would lose that, #1781) and a dense uniform sampling of
// an exact section curve. The imprint's conditioning gates measure geometry — the near-pinch neck, a
// stub band's rim — so they read points, and this is the one place that turns a loop into them.
//
//	pts := imprintLoopPoints(loops[0]) // closed: the last point repeats the first
func imprintLoopPoints(c geom.Curve3) []math.Point3 {
	if pl, ok := c.(geom.Polyline); ok {
		return pl.Vertices
	}
	if pl, ok := c.(*geom.Polyline); ok {
		return pl.Vertices
	}
	lo, hi := c.Domain()
	pts := make([]math.Point3, 0, imprintCurveSamples+1)
	for i := 0; i <= imprintCurveSamples; i++ {
		pts = append(pts, c.PointAt(lo+(hi-lo)*float64(i)/imprintCurveSamples))
	}
	return pts
}

// imprintCurveSamples is how many chords an exact section curve is sampled into for the vertex-space
// gates. It matches the arrangement's own imprint sampling (imprintSampleCount), so a gate reads the
// same resolution of the loop that the arrangement will subdivide.
const imprintCurveSamples = imprintSampleCount
