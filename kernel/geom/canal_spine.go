// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// canalSpine builds the ball-center SPINE of a constant-radius rolling-ball corner blend: the
// locus of centers of a sphere of radius `radius` internally tangent to BOTH roll hosts. That
// locus is the intersection of the two hosts' INNER offsets (each host offset by radius toward
// the cavity), so the spine is the SSI of the offsets, trimmed to the endpoint ball-centers
// `ends` (the reflected-family centers the corner already computes; N7: C″→C). It is the honest
// rolling-ball centre curve — a rim-rail spine misses by +75% (blend-sweep-spike-report.md).
//
// It errors (never fabricates geometry) on: a wrong host count, a point-spine (the offsets
// concur → the ends coincide → no corner exists), an offset self-intersection (radius exceeds a
// host's concave curvature → the ball cannot fit), a wrong offset sign (the ends do not lie on
// both offsets), or an SSI that does not converge through the corner pinch. Every tolerance is
// model-relative via res (ADR-0042). Consumed by the canal cross-section loft (C2): C2 MUST loft at
// the returned curve's exact VERTICES via [SpineVertices], never resample with PointAt(t) — a
// polyline's off-vertex points ride the chord sag between its on-host vertices.
//
//	spine, err := canalSpine([]Surface{wall, s10}, 5, [2]math.Point3{cPrime, c}, res)
func canalSpine(rolls []Surface, radius float64, ends [2]math.Point3, res Resolution) (Curve3, error) {
	if len(rolls) != 2 {
		return nil, fmt.Errorf("canalSpine: need exactly 2 roll hosts, got %d", len(rolls))
	}
	weld := res.Weld()
	if d := float64(ends[0].DistanceTo(ends[1])); d <= weld {
		return nil, fmt.Errorf(
			"canalSpine: point-spine — ball-center ends coincide (gap %g ≤ weld %g): the host offsets concur, no rolling-ball corner exists",
			d, weld)
	}
	offs, err := offsetHostsToward(rolls, radius, ends, weld)
	if err != nil {
		return nil, err
	}
	pts, err := traceSpineBetweenEnds(offs, ends, res)
	if err != nil {
		return nil, err
	}
	return finalizeSpine(pts, offs, weld)
}

// offsetHostsToward offsets each host inward by radius toward the ball-center ends, then asserts
// both ends lie on both offsets — the sign-selection validation (the ball center is at distance
// radius from each host by construction, i.e. ON each inner offset).
func offsetHostsToward(rolls []Surface, radius float64, ends [2]math.Point3, weld float64) ([2]OffsetSurface, error) {
	var offs [2]OffsetSurface
	for i, host := range rolls {
		off, err := offsetHostToward(host, radius, ends)
		if err != nil {
			return offs, err
		}
		offs[i] = off
	}
	if err := assertEndsOnOffsets(offs, ends, weld); err != nil {
		return offs, err
	}
	return offs, nil
}

// offsetHostToward offsets host by radius along the normal pointing TOWARD the ends (the cavity
// side the ball rolls on), so distance = cavitySide·radius. NewOffsetSurface rejects a fold
// (radius past the host's concave radius of curvature), carrying the offending distance.
func offsetHostToward(host Surface, radius float64, ends [2]math.Point3) (OffsetSurface, error) {
	s := cavitySide(host, ends)
	if s == 0 {
		return OffsetSurface{}, fmt.Errorf(
			"offsetHostToward: ends %v straddle host %T — cavity side ambiguous, expected both on one side", ends, host)
	}
	return NewOffsetSurface(host, s*radius)
}

// cavitySide returns +1 or -1: the sign of the host normal that points toward the ball-center
// ends. The ball rolls on the cavity side, so its center sits at distance radius from the host
// along the outward normal scaled by this sign. GENERAL (plane/cyl/cone/sphere/torus): it reads
// the geometry (foot→end · normal) per end and sums, so a single near-degenerate projection
// cannot flip the choice. Returns 0 when the ends straddle the host (signs cancel — ambiguous).
func cavitySide(host Surface, ends [2]math.Point3) float64 {
	sum := 0.0
	for _, e := range ends {
		u, v, foot := ClosestPointOnSurface(host, e)
		sum += float64(foot.VectorTo(e).Dot(host.NormalAt(u, v)))
	}
	if sum > 0 {
		return 1
	}
	if sum < 0 {
		return -1
	}
	return 0
}

// assertEndsOnOffsets verifies both ball-center ends lie on both offsets to weld. By construction
// the ball center is at distance radius from each host, hence ON each inner offset; a failure
// means the offset sign was wrong or `ends` are not the reflected-family centers.
func assertEndsOnOffsets(offs [2]OffsetSurface, ends [2]math.Point3, weld float64) error {
	for _, off := range offs {
		for _, e := range ends {
			if d := distanceToOffset(off, e); d > weld {
				return fmt.Errorf(
					"assertEndsOnOffsets: end %v is %g off offset(base %T) (want ≤ weld %g): wrong offset sign or non-family ends",
					e, d, off.Base, weld)
			}
		}
	}
	return nil
}

// distanceToOffset is the Euclidean distance from p to its foot on the offset surface.
func distanceToOffset(off OffsetSurface, p math.Point3) float64 {
	_, _, foot := ClosestPointOnSurface(off, p)
	return float64(foot.DistanceTo(p))
}

// traceSpineBetweenEnds intersects the two offsets and returns the polyline of the arc between the
// ends. It dispatches to the closed-form analytic SSI first; on handled==false it falls to the
// marched tracer, windowed and seeded at the endpoint ball-centers — the exact, best-conditioned
// seeds through the corner tangency (pitfall 1). For the offset-SSI the operands are OffsetSurface-
// typed, so IntersectSurfacesAnalytic (which keys on concrete Plane/Cylinder/... types) does not
// fire and the marched path carries every case here — the analytic branch is kept for future
// concrete-primitive spines.
func traceSpineBetweenEnds(offs [2]OffsetSurface, ends [2]math.Point3, res Resolution) ([]math.Point3, error) {
	weld := res.Weld()
	if curves, handled := IntersectSurfacesAnalytic(offs[0], offs[1], res); handled {
		return trimAnalyticToEnds(curves, ends, weld)
	}
	window := spineTraceWindow(offs[0], ends)
	curves := TraceSurfaceIntersection(offs[0], offs[1], window).Curves
	return trimTracedToEnds(curves, ends, weld)
}

// trimAnalyticToEnds tessellates each closed-form SSI curve into points and trims to the ends,
// sharing the marched-path trimmer. Unbounded curves (an infinite line) have no finite polyline
// form and are skipped.
func trimAnalyticToEnds(curves []Curve3, ends [2]math.Point3, weld float64) ([]math.Point3, error) {
	polys := make([][]math.Point3, 0, len(curves))
	for _, c := range curves {
		if pl, err := PolylineFromCurve3(c, spineAnalyticSamples); err == nil {
			polys = append(polys, pl.Vertices)
		}
	}
	return trimTracedToEnds(polys, ends, weld)
}

// spineAnalyticSamples tessellates a closed-form SSI curve densely enough that trimming to the ends
// lands within a chord of each family center before the exact snap.
const spineAnalyticSamples = 128

// Window padding for the marched SSI (dimensionless parameter-space, not a model length): pad each
// axis by a fraction of its span, but at least a floor so a collapsed span still opens a window.
const (
	spineWindowPadFrac  = 0.25 // tol:parametric — window pad as a fraction of the end-to-end span
	spineWindowPadFloor = 0.1  // tol:parametric — minimum window pad (radians / axial units)
)

// spineTraceWindow bounds the marched SSI to a padded box around the two endpoint ball-centers in
// base's parameter space. This localizes the trace to the corner arc (excluding the far Steinmetz
// branches of a cyl∩cyl) and seeds it at the exact family centers (pinch guard, pitfall 1).
func spineTraceWindow(base Surface, ends [2]math.Point3) SurfaceGrid {
	u0, v0 := base.ParamAt(ends[0])
	u1, v1 := base.ParamAt(ends[1])
	uMin, uMax := padRange(u0, u1)
	vMin, vMax := padRange(v0, v1)
	return SurfaceGrid{UMin: uMin, UMax: uMax, VMin: vMin, VMax: vMax}
}

// padRange returns [min-pad, max+pad] over a and b.
func padRange(a, b float64) (lo, hi float64) {
	lo, hi = stdmath.Min(a, b), stdmath.Max(a, b)
	pad := stdmath.Max(spineWindowPadFrac*(hi-lo), spineWindowPadFloor)
	return lo - pad, hi + pad
}

// trimTracedToEnds picks the traced polyline carrying BOTH endpoint ball-centers and returns its
// sub-run from ends[0] to ends[1]. Errors when the SSI produced no curve reaching both ends, or when
// the curve it did pick stops short of one — see assertTraceReachesEnd (C1 review finding 1): the
// force-snap in subRun is only safe once we know the raw trace actually arrived.
func trimTracedToEnds(curves [][]math.Point3, ends [2]math.Point3, weld float64) ([]math.Point3, error) {
	poly := polylineThroughEnds(curves, ends)
	if poly == nil {
		return nil, fmt.Errorf(
			"trimTracedToEnds: SSI produced no curve through both ball-center ends %v (non-convergent offset intersection)", ends)
	}
	i0 := nearestIndex(poly, ends[0])
	i1 := nearestIndex(poly, ends[1])
	if err := assertTraceReachesEnd(poly, i0, ends[0], weld); err != nil {
		return nil, err
	}
	if err := assertTraceReachesEnd(poly, i1, ends[1], weld); err != nil {
		return nil, err
	}
	return subRun(poly, i0, i1, ends), nil
}

// endSnapGapFactor bounds how far the raw traced polyline's nearest vertex to a family-center end may
// sit before subRun's force-snap is allowed to overwrite it: a small multiple of the LOCAL chord at
// that vertex (the spacing to its neighbour) — the natural yardstick for how far a HEALTHY march
// sample can land from a true on-curve point at that sampling density. This is deliberately tighter
// than polylineThroughEnds' branch-selection gate (a quarter of the end separation): that gate only
// picks which SSI curve among several carries both ends, so it must stay loose; this one guards the
// force-snap itself and must catch a tracer that stops even a couple of chords short of the real
// endpoint, which the branch gate alone would silently wave through (C1 review finding 1).
const endSnapGapFactor = 2.0 // tol:parametric — max end-snap gap as a multiple of the local chord

// assertTraceReachesEnd requires the raw traced vertex poly[idx] (the nearest sample to end) to sit
// within endSnapGapFactor local chords of end, floored at weld so a near-zero local chord (a dense
// march through a tight corner) cannot make an already-tiny, sub-weld gap look like a violation. On
// failure the error carries the measured gap and the tolerance it exceeded, per the error-message rule.
func assertTraceReachesEnd(poly []math.Point3, idx int, end math.Point3, weld float64) error {
	gap, chord := nearestVertexGap(poly, idx, end)
	tol := stdmath.Max(endSnapGapFactor*chord, weld)
	if gap > tol {
		return fmt.Errorf(
			"assertTraceReachesEnd: raw traced vertex %v is %g from family-center end %v, want <= %g "+
				"(= max(%g x local chord %g, weld %g)): the SSI tracer stopped short of the endpoint",
			poly[idx], gap, end, tol, endSnapGapFactor, chord, weld)
	}
	return nil
}

// nearestVertexGap returns the distance from poly[idx] to target and the LOCAL chord length at idx
// (the shorter of its adjacent segments, or the gap itself for a single-vertex poly) — the sampling
// density right there, used to scale assertTraceReachesEnd's tolerance to how finely the march sampled
// that stretch of the curve.
func nearestVertexGap(poly []math.Point3, idx int, target math.Point3) (gap, localChord float64) {
	gap = float64(poly[idx].DistanceTo(target))
	localChord = stdmath.Inf(1)
	if idx > 0 {
		localChord = stdmath.Min(localChord, float64(poly[idx-1].DistanceTo(poly[idx])))
	}
	if idx+1 < len(poly) {
		localChord = stdmath.Min(localChord, float64(poly[idx+1].DistanceTo(poly[idx])))
	}
	if stdmath.IsInf(localChord, 1) {
		localChord = gap
	}
	return gap, localChord
}

// polylineThroughEnds returns the traced polyline whose combined nearest approach to both ends is
// smallest AND within a gate of each end (a quarter of the end separation) — the correct SSI branch
// is the one carrying both family centers; other branches carry at most one. nil when none qualify.
func polylineThroughEnds(curves [][]math.Point3, ends [2]math.Point3) []math.Point3 {
	gate := 0.25 * float64(ends[0].DistanceTo(ends[1]))
	var best []math.Point3
	bestScore := stdmath.Inf(1)
	for _, c := range curves {
		if len(c) < 2 {
			continue
		}
		d0, d1 := minDist(c, ends[0]), minDist(c, ends[1])
		if d0 <= gate && d1 <= gate && d0+d1 < bestScore {
			bestScore, best = d0+d1, c
		}
	}
	return best
}

// subRun extracts the inclusive slice of pts between i0 and i1, oriented ends[0]→ends[1], with the
// two boundary vertices replaced by the EXACT ends (the family centers are known in closed form; the
// traced samples near them carry march noise). The overwrite is unconditional here — it is SAFE only
// because the caller (trimTracedToEnds) has already asserted via assertTraceReachesEnd that pts[i0]
// and pts[i1] are within a tight, chord-relative tolerance of ends[0]/ends[1]; this function never
// sees a pts that stopped short.
func subRun(pts []math.Point3, i0, i1 int, ends [2]math.Point3) []math.Point3 {
	lo, hi, reversed := i0, i1, false
	if i0 > i1 {
		lo, hi, reversed = i1, i0, true
	}
	run := append([]math.Point3(nil), pts[lo:hi+1]...)
	if reversed {
		reverseSpinePoints(run)
	}
	if len(run) >= 2 {
		run[0], run[len(run)-1] = ends[0], ends[1]
	}
	return run
}

// reverseSpinePoints reverses pts in place.
func reverseSpinePoints(pts []math.Point3) {
	for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
		pts[i], pts[j] = pts[j], pts[i]
	}
}

// nearestIndex returns the index of the pts vertex closest to target.
func nearestIndex(pts []math.Point3, target math.Point3) int {
	best, bi := stdmath.Inf(1), 0
	for i, p := range pts {
		if d := float64(p.DistanceTo(target)); d < best {
			best, bi = d, i
		}
	}
	return bi
}

// minDist returns the distance from target to the nearest pts vertex.
func minDist(pts []math.Point3, target math.Point3) float64 {
	return float64(pts[nearestIndex(pts, target)].DistanceTo(target))
}

// finalizeSpine refines the interior samples onto both offsets, rejects a collapsed (point) spine,
// and returns the polyline curve.
func finalizeSpine(pts []math.Point3, offs [2]OffsetSurface, weld float64) (Curve3, error) {
	refined, err := refineInterior(pts, offs, weld)
	if err != nil {
		return nil, err
	}
	poly, err := NewPolyline(refined)
	if err != nil {
		return nil, fmt.Errorf("finalizeSpine: %w — spine collapsed to %d point(s)", err, len(refined))
	}
	if l := poly.Length(); l <= weld {
		return nil, fmt.Errorf("finalizeSpine: point-spine — arc length %g ≤ weld %g: the offsets concur", l, weld)
	}
	return poly, nil
}

// refineInterior tightens each interior spine point onto the joint offset locus with an exact-host
// corrector — Newton where the host normals are transversal, damped steepest-descent through a
// corner pinch (the tangential-offset guard, pitfall 1). Endpoints stay the exact ends. Errors
// when the corrector cannot reach weld (honest reject: non-convergent / pinched SSI).
func refineInterior(pts []math.Point3, offs [2]OffsetSurface, weld float64) ([]math.Point3, error) {
	out := append([]math.Point3(nil), pts...)
	for i := 1; i+1 < len(out); i++ {
		p, ok := correctToOffsetPair(offs[0], offs[1], out[i], weld)
		if !ok {
			return nil, fmt.Errorf(
				"refineInterior: SSI corrector did not converge at spine point %v to weld %g (tangential offsets / corner pinch)",
				out[i], weld)
		}
		out[i] = p
	}
	return out, nil
}

// correctToOffsetPair pulls p onto the joint inner-offset locus — the points whose signed distance
// to each host equals that host's offset distance — using the EXACT host projections. The generic
// OffsetSurface projection carries finite-difference normal-derivative noise (~1e-4) that floors the
// spine far above weld; measuring the offset residual through the analytic host instead reaches weld.
// It reuses the SSI corrector step: tangent-plane Newton where the host normals are transversal, a
// damped steepest-descent step where they are near-parallel (the corner pinch, pitfall 1).
func correctToOffsetPair(offA, offB OffsetSurface, p math.Point3, tol float64) (math.Point3, bool) {
	for i := 0; i < ssiCorrectIters; i++ {
		uA, vA, _ := ProjectPointToSurface(offA.Base, p)
		uB, vB, _ := ProjectPointToSurface(offB.Base, p)
		nA, nB := offA.Base.NormalAt(uA, vA), offB.Base.NormalAt(uB, vB)
		sA := float64(offA.Base.PointAt(uA, vA).VectorTo(p).Dot(nA)) - offA.Distance
		sB := float64(offB.Base.PointAt(uB, vB).VectorTo(p).Dot(nB)) - offB.Distance
		if stdmath.Abs(sA) < tol && stdmath.Abs(sB) < tol {
			return p, true
		}
		p = p.TranslateBy(correctorStep(nA, nB, sA, sB))
	}
	return p, false
}

// SpineVertices returns the exact rolling-ball-center samples of a canal spine curve — the vertices
// canalSpine traced and refined onto both offsets (on-host to weld), NOT c.PointAt(t) resampling: a
// polyline's off-vertex points ride the chord between vertices (~4e-5 sag for N7) and its TangentAt
// is piecewise-constant per segment, so a consumer that needs the true rolling-ball centers — C2's
// cross-section loft MUST use these, never PointAt(t) — reads them here rather than reaching past
// canalSpine's Curve3 return with its own type-assertion (C1 review finding 2). canalSpine keeps
// returning a plain Curve3 so it stays directly usable by C2/C3's generic curve machinery; this is
// the one seam that additionally exposes the vertices. ok is false when c is not a spine-shaped
// curve (currently: not a [Polyline] — canalSpine's concrete return type).
//
//	verts, ok := geom.SpineVertices(spine)
func SpineVertices(c Curve3) (verts []math.Point3, ok bool) {
	pl, ok := c.(Polyline)
	if !ok {
		return nil, false
	}
	return pl.Vertices, true
}
