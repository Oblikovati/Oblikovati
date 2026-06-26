// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// General (u,v) arrangement trim of a ruled side (Oblikovati#1405). The analytic walk in
// curved_halfspace_ruled_boundary.go reduces the kept region to a single-valued v-interval [lo(u),hi(u)]
// per azimuth — it reads the surface only through keptV/sectionV, so it cannot trim by a MULTI-valued or
// CLOSED-island imprint (the general curved∩curved case). This file replaces that walk with a parameter-
// space arrangement: project the imprint into the side's (u,v) band, subdivide the band into cells along
// the imprint, classify each cell kept/dropped by a material predicate, and re-emit the kept region's
// boundary as exact analytic edges. A plane cut becomes the special case where the imprint is one conic.
//
// The arrangement reuses the planar subdivision engine (Arrange, arrange2d.go) on the (u,v) point cloud;
// curve identity is carried on the sampled segments so the re-emitted boundary keeps exact arcs/conics
// rather than the sampled polyline. This file builds the (u,v) projection and the tagged imprint sampling;
// the cell classification and boundary re-emission follow in the same arrangement pipeline.

// paramOf returns the (u, v) = (azimuth, axial distance) of a 3D point on the ruled side — the inverse of
// point3. v is the signed distance along the axis from the v=0 base; the radial remainder gives the azimuth
// in [0, 2π) measured from the ref direction. A point off the surface projects to its nearest (u,v) (the
// axial/azimuth components are still well defined), which is what sampling an imprint that lies ON the
// surface to tolerance needs.
func (c ruledUV) paramOf(p math.Point3) math.Point2 {
	d := c.base.VectorTo(p)
	v := float64(d.Dot(c.axis))
	radial := d.Sub(c.axis.Scale(math.Scalar(v)))
	u := stdmath.Atan2(float64(radial.Dot(c.binor)), float64(radial.Dot(c.ref)))
	if u < 0 {
		u += 2 * stdmath.Pi
	}
	return math.P2(u, v)
}

// segKind tags what a (u,v) segment re-emits to when it bounds a kept cell: an imprint sub-arc (the cut),
// a band rim (constant v, a circle arc), or the azimuth SEAM ruling (u=0≡2π — an ARTIFICIAL edge that
// closes the parameter rectangle for the arrangement and dissolves wherever the kept region wraps it).
type segKind int

const (
	segImprint segKind = iota
	segRim
	segSeam
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

// sampleRange samples one curve parameter interval [t0,t1] into tagged (u,v) segments.
func (c ruledUV) sampleRange(curve geom.Curve3, t0, t1 float64) []uvSeg {
	if t0 == t1 {
		return nil
	}
	segs := make([]uvSeg, 0, imprintSampleCount)
	prevT := t0
	prevP := c.paramOf(curve.PointAt(t0))
	for i := 1; i <= imprintSampleCount; i++ {
		t := t0 + (t1-t0)*float64(i)/imprintSampleCount
		p := c.paramOf(curve.PointAt(t))
		segs = append(segs, uvSeg{a: prevP, b: p, curve: curve, tA: prevT, tB: t, kind: segImprint})
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

// splitSeamCrossing splits an imprint segment whose endpoints straddle the azimuth seam (the shorter arc
// between them crosses u=0≡2π) into two segments meeting AT the seam, so no segment spans the discontinuity
// — the arrangement sees a clean parameter rectangle. v and the curve parameter are interpolated to the
// seam crossing. A segment that does not straddle the seam is returned unchanged.
func splitSeamCrossing(s uvSeg) []uvSeg {
	ub := unwrapAzimuthNear(s.a.X, s.b.X)
	if ub >= 0 && ub <= 2*stdmath.Pi {
		return []uvSeg{s} // wholly inside the band, no seam crossing
	}
	seamU, otherU := 0.0, 2*stdmath.Pi
	if ub > 2*stdmath.Pi { // the run climbs past 2π: a → 2π, then 0 → b
		seamU, otherU = 2*stdmath.Pi, 0
	}
	f := (seamU - s.a.X) / (ub - s.a.X)
	vSeam := s.a.Y + f*(s.b.Y-s.a.Y)
	tSeam := s.tA + f*(s.tB-s.tA)
	return []uvSeg{
		{a: s.a, b: math.P2(seamU, vSeam), curve: s.curve, tA: s.tA, tB: tSeam, kind: s.kind},
		{a: math.P2(otherU, vSeam), b: s.b, curve: s.curve, tA: tSeam, tB: s.tB, kind: s.kind},
	}
}

// bandFrameSegments returns the four straight (u,v) edges that bound the parameter rectangle for the
// arrangement: the bottom rim (v=vMin) and top rim (v=vMax) — each a single horizontal segment in (u,v),
// tagged to its rim circle — and the two seam verticals (u=0 and u=2π), tagged to the shared seam ruling.
// The rims are genuine surface boundaries; the seam verticals are artificial and dissolve where the kept
// region wraps them (resolved when cells are merged across the seam).
func (c ruledUV) bandFrameSegments() []uvSeg {
	twoPi := 2 * stdmath.Pi
	seam := geom.NewLineSegment(c.point3(0, c.band.vMin), c.point3(0, c.band.vMax))
	return []uvSeg{
		{a: math.P2(0, c.band.vMin), b: math.P2(twoPi, c.band.vMin), curve: c.band.bottomCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMax), b: math.P2(twoPi, c.band.vMax), curve: c.band.topCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMin), b: math.P2(0, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(twoPi, c.band.vMin), b: math.P2(twoPi, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
	}
}

// assembleBandSegments builds the full tagged (u,v) segment set the arrangement subdivides: every imprint
// segment (seam-split so none spans the azimuth discontinuity) plus the rim+seam frame that closes the
// parameter rectangle. Degenerate segments (both endpoints welded by the seam split) are dropped.
func (c ruledUV) assembleBandSegments(imprint []uvSeg) []uvSeg {
	out := make([]uvSeg, 0, len(imprint)+4)
	for _, s := range imprint {
		for _, split := range splitSeamCrossing(s) {
			// Clip each imprint segment to the band's axial range: a section can leave [vMin,vMax] (a tilted
			// cut's ellipse rises past the rim), and sampling that out-of-band part would inject a spurious
			// arc; clipping lands the imprint exactly on the rim where it crosses, the real rim split.
			for _, clipped := range c.clipSegToVBand(split) {
				if clipped.a.DistanceTo(clipped.b) > arrTol {
					out = append(out, clipped)
				}
			}
		}
	}
	return append(out, c.bandFrameSegments()...)
}

// clipSegToVBand clips a (u,v) imprint segment to the axial band [vMin,vMax], returning the in-band part or
// nothing when the segment lies wholly outside on one side. Crucially, an endpoint that leaves the band is
// snapped to the EXACT curve parameter where the imprint crosses the rim (not a linear interpolation of the
// sampled segment), so the rim-crossing 3D point equals plane∩rim and welds with the cap's matching arc —
// the section ellipse is plane∩cylinder, so its v=rim crossing is exactly plane∩rim (#1405).
func (c ruledUV) clipSegToVBand(s uvSeg) []uvSeg {
	vMin, vMax := c.band.vMin, c.band.vMax
	a, b := float64(s.a.Y), float64(s.b.Y)
	if (a < vMin && b < vMin) || (a > vMax && b > vMax) {
		return nil
	}
	sa, ta := c.clipEndToBand(s, true)
	sb, tb := c.clipEndToBand(s, false)
	if sa.DistanceTo(sb) <= arrTol {
		return nil
	}
	return []uvSeg{{a: sa, b: sb, curve: s.curve, tA: ta, tB: tb, kind: s.kind}}
}

// clipEndToBand returns one endpoint of a segment clipped to the band: the endpoint unchanged when already
// in [vMin,vMax], else the (u,v) and curve parameter where the imprint curve exactly reaches the nearer rim
// (refined on the curve, between this end's parameter and the other's).
func (c ruledUV) clipEndToBand(s uvSeg, isA bool) (math.Point2, float64) {
	p, v, t, tOther := s.a, float64(s.a.Y), s.tA, s.tB
	if !isA {
		p, v, t, tOther = s.b, float64(s.b.Y), s.tB, s.tA
	}
	if v >= c.band.vMin && v <= c.band.vMax {
		return p, t
	}
	vLim := c.band.vMin
	if v > c.band.vMax {
		vLim = c.band.vMax
	}
	tc := c.refineCurveV(s.curve, t, tOther, vLim)
	return c.paramOf(s.curve.PointAt(tc)), tc
}

// refineCurveV bisects the imprint curve parameter between tOut (outside the band) and tIn (inside) to the
// point where the curve's axial coordinate v equals vLim — the exact rim crossing.
func (c ruledUV) refineCurveV(curve geom.Curve3, tOut, tIn, vLim float64) float64 {
	out := tOut
	for i := 0; i < 50; i++ {
		tm := (tOut + tIn) / 2
		if (c.curveV(curve, out)-vLim <= 0) == (c.curveV(curve, tm)-vLim <= 0) {
			tOut = tm
		} else {
			tIn = tm
		}
	}
	return (tOut + tIn) / 2
}

// curveV is the axial coordinate v of a 3D imprint curve at parameter t.
func (c ruledUV) curveV(curve geom.Curve3, t float64) float64 {
	return float64(c.paramOf(curve.PointAt(t)).Y)
}

// materialPredicate reports whether a (u,v) point of the band is on the KEPT side of the imprint. For a
// plane cut it is the half-space membership g(u,v) = a(u)+v·b(u) < 0 (the same {g<0} the analytic walk
// keeps); a general curved∩curved imprint supplies its own inside/outside test. The arrangement is built
// once and classified through this predicate, so the two cases share the whole pipeline (#1405).
type materialPredicate func(uv math.Point2) bool

// halfSpaceMaterial is the plane-cut predicate: a band point is kept where the signed distance g(u,v) is
// negative, exactly the {g<0} region the single-valued walk traced as a v-interval.
func (c ruledUV) halfSpaceMaterial() materialPredicate {
	return func(uv math.Point2) bool { return c.aU(uv.X)+uv.Y*c.bU(uv.X) < 0 }
}

// arrangeBand runs the planar subdivision on the assembled band, returning the cells as (u,v) polygons
// (outer loop + nested holes). It is the periodic band flattened to a rectangle; cross-seam adjacency is
// reconciled later when the kept region's boundary is walked.
func (c ruledUV) arrangeBand(segs []uvSeg) []Face2D {
	in := make([][2]math.Point2, 0, len(segs))
	for _, s := range segs {
		in = append(in, [2]math.Point2{s.a, s.b})
	}
	return Arrange(in)
}

// keptCells returns the arrangement cells whose interior is on the material side of the imprint, each cell
// classified by evaluating the predicate at a point strictly inside its outer loop. A cell straddling the
// imprint cannot occur (the imprint is an arrangement edge), so any interior sample decides the whole cell.
func keptCells(cells []Face2D, material materialPredicate) []Face2D {
	var kept []Face2D
	for _, cell := range cells {
		if p, ok := interiorPointOf(cell.Outer); ok && material(p) {
			kept = append(kept, cell)
		}
	}
	return kept
}

// interiorPointOf returns a point strictly inside the simple polygon (and ok). The centroid serves for a
// convex cell; for a concave one it may fall outside, so the fallback steps a short way from each edge
// midpoint toward the centroid until a point lands inside — robust for the arrangement's simple cells.
func interiorPointOf(poly []math.Point2) (math.Point2, bool) {
	if len(poly) < 3 {
		return math.Point2{}, false
	}
	c := centroidOf(poly)
	if pointInPolygon2D(c, poly) {
		return c, true
	}
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		mid := math.P2((float64(a.X)+float64(b.X))/2, (float64(a.Y)+float64(b.Y))/2)
		for _, f := range []float64{1e-3, 1e-2, 0.1, 0.5} {
			p := math.P2(lerp(float64(mid.X), float64(c.X), f), lerp(float64(mid.Y), float64(c.Y), f))
			if pointInPolygon2D(p, poly) {
				return p, true
			}
		}
	}
	return c, false
}

// centroidOf returns the vertex average of a polygon (a cheap interior estimate, exact for the convex case).
func centroidOf(poly []math.Point2) math.Point2 {
	var sx, sy float64
	for _, p := range poly {
		sx, sy = sx+float64(p.X), sy+float64(p.Y)
	}
	n := float64(len(poly))
	return math.P2(sx/n, sy/n)
}

// lerp linearly interpolates from a to b by fraction f.
func lerp(a, b, f float64) float64 { return a + f*(b-a) }

// seamWeld grid (matches the arrangement welder, arrTol/tjTol family) used to identify (u,v) boundary
// vertices, with the azimuth seam folded: u=2π is the SAME ruling as u=0, so both weld to one vertex.
const seamWeldGrid = 1e-7

// seamWelder welds (u,v) points onto shared indices with the azimuth seam identified (u=2π≡u=0). Folding
// the seam is what turns the artificial seam edges of a wrapping kept region into reverse twins that cancel.
type seamWelder struct {
	index  map[[2]int64]int
	points []math.Point2
}

func newSeamWelder() *seamWelder { return &seamWelder{index: map[[2]int64]int{}} }

// add returns the welded index of p, normalising u=2π to u=0 first so a seam vertex on either side of the
// parameter rectangle maps to one ruling vertex.
func (w *seamWelder) add(p math.Point2) int {
	u := float64(p.X)
	if stdmath.Abs(u-2*stdmath.Pi) < seamWeldGrid {
		u = 0
	}
	k := [2]int64{int64(stdmath.Round(u / seamWeldGrid)), int64(stdmath.Round(float64(p.Y) / seamWeldGrid))}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

// dedge is one directed (u,v) boundary edge of the kept region: the seam-welded endpoint indices for
// cancellation/chaining, and the original (u,v) endpoints (a may sit at u=2π even when welded to u=0) for
// re-emitting the exact analytic curve.
type dedge struct {
	from, to int
	a, b     math.Point2
}

// keptBoundaryEdges returns the directed boundary edges of the kept region. It collects every kept cell's
// oriented boundary (outer CCW, holes CW), welds endpoints with the seam folded, then keeps only edges
// whose reverse is absent: an edge interior to the kept region (bordering two kept cells, or a seam edge a
// wrapping region traverses on both sides) appears as a reverse-twin pair and cancels, leaving exactly the
// edges between kept material and dropped material (or the band rims). This is the cross-seam merge and the
// shared-edge dissolve in one pass (#1405).
func keptBoundaryEdges(kept []Face2D) []dedge {
	w := newSeamWelder()
	var all []dedge
	add := func(poly []math.Point2) {
		for i, n := 0, len(poly); i < n; i++ {
			a, b := poly[i], poly[(i+1)%n]
			if a.DistanceTo(b) <= arrTol {
				continue // a degenerate edge
			}
			all = append(all, dedge{from: w.add(a), to: w.add(b), a: a, b: b})
		}
	}
	for _, cell := range kept {
		add(cell.Outer)
		for _, h := range cell.Holes {
			add(h)
		}
	}
	present := make(map[[2]int]bool, len(all))
	for _, e := range all {
		present[[2]int{e.from, e.to}] = true
	}
	survivors := make([]dedge, 0, len(all))
	for _, e := range all {
		// A full-wrap edge (an uncut rim/section circle: both ends weld to the seam vertex) is its own
		// closed loop and has no distinct reverse, so it is never canceled. Any other edge survives only if
		// its reverse is absent — the shared-edge dissolve and the cross-seam merge.
		if e.from == e.to || !present[[2]int{e.to, e.from}] {
			survivors = append(survivors, e)
		}
	}
	return survivors
}

// chainLoops links the surviving directed boundary edges into closed loops by following each edge's `to`
// vertex to the next edge that starts there. A valid kept-region boundary has matched in/out degree at
// every vertex, so following any unused outgoing edge closes each loop; the result is one loop per
// connected boundary component (a wrapping band yields two — a rim loop and a section loop). A full-wrap
// edge (from==to) is its own one-edge loop.
func chainLoops(edges []dedge) [][]dedge {
	out := make(map[int][]int) // vertex -> indices of edges starting there
	for i, e := range edges {
		out[e.from] = append(out[e.from], i)
	}
	used := make([]bool, len(edges))
	var loops [][]dedge
	for i := range edges {
		if !used[i] {
			loops = append(loops, walkLoop(i, edges, out, used))
		}
	}
	return loops
}

// walkLoop follows the boundary from edge i, marking edges used, until it returns to the start vertex (or a
// full-wrap singleton, or a dead end). out maps each vertex to the edges leaving it.
func walkLoop(i int, edges []dedge, out map[int][]int, used []bool) []dedge {
	start := edges[i].from
	var loop []dedge
	for cur := i; ; {
		used[cur] = true
		loop = append(loop, edges[cur])
		if edges[cur].from == edges[cur].to || edges[cur].to == start {
			break // a full-wrap singleton, or the loop closed back to its start
		}
		next := nextUnused(out[edges[cur].to], used)
		if next < 0 {
			break
		}
		cur = next
	}
	return loop
}

// nextUnused returns the first not-yet-used edge index from a vertex's outgoing list, or -1.
func nextUnused(candidates []int, used []bool) int {
	for _, i := range candidates {
		if !used[i] {
			return i
		}
	}
	return -1
}

// recoveredEdge is one boundary dedge resolved back to the analytic curve it lies on: its kind, the source
// curve, its (u,v) endpoints (giving the azimuth span for a rim/seam edge) and the curve parameters at
// those endpoints (for an imprint sub-arc). It is the bridge from the (u,v) arrangement back to exact
// 3D geometry (#1405).
type recoveredEdge struct {
	kind   segKind
	curve  geom.Curve3
	a, b   math.Point2
	tA, tB float64
}

// recoverEdge resolves a boundary dedge to the assembled segment it lies on, recovering the segment's kind
// and source curve, and (for an imprint edge) interpolating the curve parameter at each endpoint from the
// matched segment's tagged endpoints. The dedge is a sub-piece of exactly one assembled segment, found by
// its midpoint's perpendicular distance.
func (c ruledUV) recoverEdge(d dedge, segs []uvSeg) (recoveredEdge, bool) {
	mid := math.P2((float64(d.a.X)+float64(d.b.X))/2, (float64(d.a.Y)+float64(d.b.Y))/2)
	best, bestDist := -1, tjTol
	for i, s := range segs {
		if dist := perpDistToSeg(mid, s.a, s.b); dist < bestDist {
			best, bestDist = i, dist
		}
	}
	if best < 0 {
		return recoveredEdge{}, false
	}
	s := segs[best]
	re := recoveredEdge{kind: s.kind, curve: s.curve, a: d.a, b: d.b}
	if s.kind == segImprint {
		re.tA = lerp(s.tA, s.tB, projFraction(d.a, s.a, s.b))
		re.tB = lerp(s.tA, s.tB, projFraction(d.b, s.a, s.b))
	}
	return re, true
}

// perpDistToSeg returns the distance from p to the segment a→b (clamped to the segment).
func perpDistToSeg(p, a, b math.Point2) float64 {
	ab := a.VectorTo(b)
	l2 := float64(ab.LengthSquared())
	if l2 < arrTol*arrTol {
		return float64(p.DistanceTo(a))
	}
	t := clamp01(float64(a.VectorTo(p).Dot(ab)) / l2)
	return float64(p.DistanceTo(a.TranslateBy(ab.Scale(math.Scalar(t)))))
}

// projFraction returns p's projection parameter along a→b, clamped to [0,1].
func projFraction(p, a, b math.Point2) float64 {
	ab := a.VectorTo(b)
	l2 := float64(ab.LengthSquared())
	if l2 < arrTol*arrTol {
		return 0
	}
	return clamp01(float64(a.VectorTo(p).Dot(ab)) / l2)
}

// clamp01 clamps t to [0,1].
func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// emitLoopEdges re-emits a (u,v) boundary loop as a chain of exact analytic loopEdges: recover each dedge's
// analytic source, merge consecutive edges that share a curve into one run, and emit each run as a single
// loopEdge (a rim arc, an imprint sub-arc, or a seam ruling). It returns the full boundary chain and,
// separately, the imprint (section) sub-arcs alone — the cut edges the planar lid re-uses (reversed).
func (c ruledUV) emitLoopEdges(loop []dedge, segs []uvSeg) (face, section []loopEdge, ok bool) {
	rec := make([]recoveredEdge, 0, len(loop))
	for _, d := range loop {
		re, good := c.recoverEdge(d, segs)
		if !good {
			return nil, nil, false
		}
		rec = append(rec, re)
	}
	for _, run := range mergeRuns(rotateToTransition(rec)) {
		e, good := c.emitRun(run)
		if !good {
			return nil, nil, false
		}
		face = append(face, e)
		if run[0].kind == segImprint {
			section = append(section, e)
		}
	}
	return face, section, true
}

// trimByImprint is the general ruled-side split: it trims the side by the (u,v) imprint of the given
// analytic curves, keeping the cells the material predicate selects, and returns the kept curvedFace(s)
// and the section (cut) arcs that bound the planar lid. It is the arrangement replacement for the analytic
// splitSide — a plane cut passes its section conic and the half-space predicate, while a general
// curved∩curved cut passes its projected imprint and membership test (#1405).
func (c ruledUV) trimByImprint(f curvedFace, surface geom.Surface, imprint []geom.Curve3, material materialPredicate) ([]curvedFace, []loopEdge, error) {
	var imp []uvSeg
	for _, cv := range imprint {
		imp = append(imp, c.sampleImprintUV(cv)...)
	}
	segs := c.assembleBandSegments(imp)
	kept := keptCells(c.arrangeBand(segs), material)
	if len(kept) == 0 {
		return nil, nil, nil // the whole side is on the dropped side
	}
	emitted, ok := c.emitKeptLoops(chainLoops(keptBoundaryEdges(kept)), segs)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	faceLoops, lid := c.orientLoops(emitted, c.wrapsAllU())
	kf := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: faceLoops}
	return []curvedFace{kf}, lid, nil
}

// emittedLoop is one re-emitted boundary loop awaiting orientation: its full edge chain, the imprint
// (section) sub-arcs alone (for the lid), and its mean axial level (to order hi boundary first).
type emittedLoop struct {
	face    []loopEdge
	section []loopEdge
	mv      float64
}

// emitKeptLoops re-emits every (u,v) boundary loop to analytic edges, sorted with the higher (hi) boundary
// first to match the analytic split convention (loops[0] is the hi boundary / outer loop).
func (c ruledUV) emitKeptLoops(loops [][]dedge, segs []uvSeg) ([]emittedLoop, bool) {
	out := make([]emittedLoop, 0, len(loops))
	for _, lp := range loops {
		face, section, ok := c.emitLoopEdges(lp, segs)
		if !ok {
			return nil, false
		}
		out = append(out, emittedLoop{face: face, section: section, mv: meanEdgeV(c, face)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].mv > out[j].mv })
	return out, true
}

// orientLoops applies the analytic splitSide orientation convention to the ordered loops: the lo boundaries
// and the default hi boundary are forward with their section arcs reversed into the lid; but in a WRAPPING
// band the hi loop is REVERSED when it carries a rim and the source side traversed its top rim reversed
// (band.topRimReversed), the lid then taking that loop's section arcs forward — so the rebuilt rim keeps the
// sense opposite its cap and the band, lid and caps stay a consistent manifold. The reversal is gated on
// wrapping because a non-wrapping tongue is one mixed loop the analytic tongueSide leaves un-reversed
// (#1405, mirroring curved_halfspace_ruled_uv.go's splitSide vs tongueSide).
func (c ruledUV) orientLoops(loops []emittedLoop, wrapping bool) ([]curvedLoop, []loopEdge) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	for i, e := range loops {
		face, lidSec := e.face, reverseEdgeChain(e.section)
		if i == 0 && wrapping && c.band.topRimReversed && allRimEdges(e.face) {
			face, lidSec = reverseEdgeChain(e.face), e.section
		}
		faceLoops = append(faceLoops, curvedLoop{edges: face})
		lid = append(lid, lidSec...)
	}
	return faceLoops, lid
}

// allRimEdges reports whether every edge of a loop is a rim (circle/arc) — a PURE top-rim hi boundary, the
// only wrapping hi loop the topRimReversed whole-loop reversal applies to; a mixed section+rim hi boundary
// (a clips-rim annulus) keeps its arrangement orientation instead.
func allRimEdges(edges []loopEdge) bool {
	for _, e := range edges {
		switch e.curve.(type) {
		case geom.Circle, geom.Arc3d:
		default:
			return false
		}
	}
	return len(edges) > 0
}

// meanEdgeV is the mean axial level of an edge chain (the average v of its endpoints), used to order the
// hi boundary loop first.
func meanEdgeV(c ruledUV, edges []loopEdge) float64 {
	if len(edges) == 0 {
		return 0
	}
	sum := 0.0
	for _, e := range edges {
		sum += float64(c.paramOf(e.start()).Y) + float64(c.paramOf(e.end()).Y)
	}
	return sum / float64(2*len(edges))
}

// rotateToTransition rotates the recovered loop so it begins at a curve boundary (where the kind/curve
// changes), so every run mergeRuns forms is ONE contiguous arc. Without it a loop that happens to start in
// the middle of a curve splits that curve into two runs around the loop seam — and a closed conic run that
// wraps re-emits as a spurious full curve. A loop on a single curve (no transition) is returned unchanged.
func rotateToTransition(rec []recoveredEdge) []recoveredEdge {
	n := len(rec)
	for i := 0; i < n; i++ {
		if !sameRun(rec[(i-1+n)%n], rec[i]) {
			return append(append([]recoveredEdge{}, rec[i:]...), rec[:i]...)
		}
	}
	return rec
}

// mergeRuns groups consecutive recovered edges that lie on the same analytic curve into runs, so a chain of
// many sampled sub-edges along one conic (or many sub-arcs of one rim) re-emits as a single loopEdge.
func mergeRuns(rec []recoveredEdge) [][]recoveredEdge {
	var runs [][]recoveredEdge
	for _, e := range rec {
		if n := len(runs); n > 0 && sameRun(runs[n-1][0], e) {
			runs[n-1] = append(runs[n-1], e)
		} else {
			runs = append(runs, []recoveredEdge{e})
		}
	}
	return runs
}

// emitRun re-emits one run of recovered edges (all on the same curve) as a single exact loopEdge.
func (c ruledUV) emitRun(run []recoveredEdge) (loopEdge, bool) {
	switch run[0].kind {
	case segRim:
		return c.emitRimRun(run)
	case segImprint:
		return c.emitImprintRun(run)
	default:
		return c.emitSeamRun(run)
	}
}

// emitRimRun re-emits a run along a band rim as one circular edge: the full rim circle when the run wraps
// the whole azimuth, else a geom.Arc3d over the run's azimuth span in the surface's own frame (so it lands
// exactly on the rim). The span sums per-edge azimuth deltas, which stays correct across the seam and for
// a single full-wrap edge (where the raw endpoints alone would read as a zero span).
func (c ruledUV) emitRimRun(run []recoveredEdge) (loopEdge, bool) {
	bottom := stdmath.Abs(float64(run[0].a.Y)-c.band.vMin) <= stdmath.Abs(float64(run[0].a.Y)-c.band.vMax)
	center, radius, circle := c.band.top, c.band.rTop, c.band.topCirc
	if bottom {
		center, radius, circle = c.band.bottom, c.band.rBot, c.band.bottomCirc
	}
	span := 0.0
	for _, e := range run {
		span += float64(e.b.X) - float64(e.a.X)
	}
	if stdmath.Abs(span) >= 2*stdmath.Pi-1e-6 {
		return loopEdge{curve: circle, t0: 0, t1: 1}, true // a full rim circle, reused whole
	}
	arc, err := geom.NewArc3d(center, c.axis, c.ref, radius, float64(run[0].a.X), span)
	if err != nil {
		return loopEdge{}, false
	}
	return loopEdge{curve: arc, t0: 0, t1: 1}, true
}

// emitImprintRun re-emits a run along one imprint curve as a single loopEdge over the recovered parameter
// span. For a CLOSED conic (ellipse/circle) the parameters wrap, so the run's parameter sequence is
// unwrapped to a monotone span (handling the param seam, like the analytic sectionArm); a run that covers
// the whole closed curve re-emits it over its full domain. Open conic arms (hyperbola/parabola/line) carry
// monotone parameters already.
func (c ruledUV) emitImprintRun(run []recoveredEdge) (loopEdge, bool) {
	curve := run[0].curve
	t0 := run[0].tA
	tEnd := run[len(run)-1].tB
	if isClosedCurve(curve) {
		prev := t0
		for _, e := range run {
			prev = unwrapParamNear(prev, e.tB)
		}
		tEnd = prev
		if lo, hi := curve.Domain(); stdmath.Abs(tEnd-t0) >= (hi-lo)-1e-6 {
			return loopEdge{curve: curve, t0: lo, t1: hi}, true // the whole closed curve
		}
	}
	return loopEdge{curve: curve, t0: t0, t1: tEnd}, true
}

// emitSeamRun re-emits a (rare, surviving) seam-ruling run as a straight edge between its 3D endpoints — a
// ruling of the side from the run's first to last (u,v) vertex.
func (c ruledUV) emitSeamRun(run []recoveredEdge) (loopEdge, bool) {
	p0 := c.point3(float64(run[0].a.X), float64(run[0].a.Y))
	p1 := c.point3(float64(run[len(run)-1].b.X), float64(run[len(run)-1].b.Y))
	return loopEdge{curve: geom.NewLineSegment(p0, p1), t0: 0, t1: 1}, true
}

// isClosedCurve reports whether a conic closes on itself (an ellipse or circle), so its parameter wraps and
// a boundary run along it must be unwrapped to a monotone span before re-emission.
func isClosedCurve(curve geom.Curve3) bool {
	switch curve.(type) {
	case geom.EllipseFull, geom.Circle:
		return true
	}
	return false
}

// sameRun reports whether two recovered edges belong to one re-emittable run: the same kind, and for an
// imprint the same source curve, for a rim the same rim level (a rim sits at constant v, so equal v).
func sameRun(a, b recoveredEdge) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case segImprint:
		return a.curve == b.curve
	case segRim:
		return stdmath.Abs(float64(a.a.Y)-float64(b.a.Y)) < 1e-6
	default:
		return true
	}
}
