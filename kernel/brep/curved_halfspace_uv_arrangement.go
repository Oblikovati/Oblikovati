// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

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
	t0, t1 := curve.Domain()
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
			if split.a.DistanceTo(split.b) > arrTol {
				out = append(out, split)
			}
		}
	}
	return append(out, c.bandFrameSegments()...)
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
