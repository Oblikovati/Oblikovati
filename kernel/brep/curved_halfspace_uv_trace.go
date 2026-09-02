// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Phase 4 of the ruled-side (u,v) arrangement (#2212): extract the kept region's boundary and
// chain it into loops.
//
// A boundary edge is one a kept cell owns and its neighbour does not. The seam welder folds
// u=2pi onto u=0 (and, on a torus, v=2pi onto v=0) so a region that wraps the surface cancels
// its artificial seam edges as reverse twins instead of emitting them. Chaining walks the
// directed edges by turning angle, which is what keeps a self-touching boundary correct.

// seamWeld grid (matches the arrangement welder, arrTol/tjTol family) used to identify (u,v) boundary
// vertices, with the azimuth seam folded: u=2π is the SAME ruling as u=0, so both weld to one vertex.
const seamWeldGrid = 1e-7

// seamWelder welds (u,v) points onto shared indices with the azimuth seam identified (u=2π≡u=0). Folding
// the seam is what turns the artificial seam edges of a wrapping kept region into reverse twins that cancel.
// On a v-periodic surface (a torus) the tube seam is folded too (v=2π≡v=0), so a kept region wrapping the
// tube cancels its v-seam edges the same way a wrapping band cancels its u-seam (Oblikovati#1406).
type seamWelder struct {
	index     map[[2]int64]int
	points    []math.Point2
	uPeriodic bool
	vPeriodic bool
}

func newSeamWelder(uPeriodic, vPeriodic bool) *seamWelder {
	return &seamWelder{index: map[[2]int64]int{}, uPeriodic: uPeriodic, vPeriodic: vPeriodic}
}

// add returns the welded index of p, normalising u=2π to u=0 (and, on a v-periodic surface, v=2π to v=0)
// first so a seam vertex on either side of the parameter rectangle maps to one ruling/tube vertex. On a
// NON-periodic side (a bounded plane, planeUV) u is a real world distance, not an azimuth: folding u≈2π
// would silently weld a genuine face vertex onto u=0, so the u-fold is gated on uPeriodic (#1591).
func (w *seamWelder) add(p math.Point2) int {
	u, v := float64(p.X), float64(p.Y)
	if w.uPeriodic && stdmath.Abs(u-2*stdmath.Pi) < seamWeldGrid {
		u = 0
	}
	if w.vPeriodic && stdmath.Abs(v-2*stdmath.Pi) < seamWeldGrid {
		v = 0
	}
	k := [2]int64{int64(stdmath.Round(u / seamWeldGrid)), int64(stdmath.Round(v / seamWeldGrid))}
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
func keptBoundaryEdges(kept []Face2D, uPeriodic, vPeriodic bool) []dedge {
	w := newSeamWelder(uPeriodic, vPeriodic)
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
// full-wrap singleton, or a dead end). out maps each vertex to the edges leaving it. At a vertex it takes
// the angular successor (nextByAngle), so a degree-4 self-touch vertex (the Steinmetz pinch) is traced as a
// turn into the adjacent lobe rather than a straight crossing into the other lobe (#1403).
func walkLoop(i int, edges []dedge, out map[int][]int, used []bool) []dedge {
	start := edges[i].from
	var loop []dedge
	for cur := i; ; {
		used[cur] = true
		loop = append(loop, edges[cur])
		if edges[cur].from == edges[cur].to || edges[cur].to == start {
			break // a full-wrap singleton, or the loop closed back to its start
		}
		next := nextByAngle(cur, edges, out, used)
		if next < 0 {
			break
		}
		cur = next
	}
	return loop
}

// nextByAngle picks the boundary successor of edge cur at its head vertex: the unused outgoing half-edge
// first in CLOCKWISE order from the reversed arrival direction (de Berg §2.3, the DCEL face-traversal rule).
// With kept material on the left of every directed edge (keptBoundaryEdges orients them so), this is the
// tightest right turn, which hugs the face boundary — so two lobes meeting at one degree-4 vertex trace as
// two loops instead of crossing through. At a degree-2 vertex there is a single outgoing edge and the rule
// reduces to nextUnused, so every existing (non-self-touching) arrangement is unchanged. Returns -1 on a
// dead end (no unused outgoing).
func nextByAngle(cur int, edges []dedge, out map[int][]int, used []bool) int {
	arrival := edges[cur]
	// The reversed arrival ray, pointing from the head vertex back along the edge we came in on.
	back := arrival.b.VectorTo(arrival.a)
	best, bestCW := -1, 0.0
	for _, j := range out[arrival.to] {
		if used[j] {
			continue
		}
		leaving := edges[j].a.VectorTo(edges[j].b)
		cw := clockwiseAngle(back, leaving)
		if best < 0 || cw < bestCW {
			best, bestCW = j, cw
		}
	}
	return best
}

// clockwiseAngle returns the angle in [0, 2π) swept CLOCKWISE from vector a to vector b. A near-zero result
// means b points back along a (the reverse edge we arrived on); the caller never offers that edge, but the
// ordering is total either way.
func clockwiseAngle(a, b math.Vector2) float64 {
	angA := stdmath.Atan2(float64(a.Y), float64(a.X))
	angB := stdmath.Atan2(float64(b.Y), float64(b.X))
	cw := angA - angB
	for cw < 0 {
		cw += 2 * stdmath.Pi
	}
	for cw >= 2*stdmath.Pi {
		cw -= 2 * stdmath.Pi
	}
	return cw
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
func recoverEdge(d dedge, segs []uvSeg) (recoveredEdge, bool) {
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
	// An imprint arc AND a non-periodic face-polygon edge (planeUV, #1591) both re-emit through emitImprintRun,
	// which needs the curve parameters at the dedge endpoints; a rim recomputes its own span in emitRimRun, so
	// it is left out. Interpolate the parameter from the matched segment's tagged endpoints.
	if s.kind == segImprint || s.kind == segPolygon {
		re.tA = math.Lerp(s.tA, s.tB, projFraction(d.a, s.a, s.b))
		re.tB = math.Lerp(s.tA, s.tB, projFraction(d.b, s.a, s.b))
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
	t := math.Clamp01(float64(a.VectorTo(p).Dot(ab)) / l2)
	return float64(p.DistanceTo(a.TranslateBy(ab.Scale(math.Scalar(t)))))
}

// projFraction returns p's projection parameter along a→b, clamped to [0,1].
func projFraction(p, a, b math.Point2) float64 {
	ab := a.VectorTo(b)
	l2 := float64(ab.LengthSquared())
	if l2 < arrTol*arrTol {
		return 0
	}
	return math.Clamp01(float64(a.VectorTo(p).Dot(ab)) / l2)
}

// emitLoopEdges re-emits a (u,v) boundary loop as a chain of exact analytic loopEdges: recover each dedge's
// analytic source, merge consecutive edges that share a curve into one run, and emit each run as a single
// loopEdge (a rim arc, an imprint sub-arc, or a seam ruling). It returns the full boundary chain and,
// separately, the imprint (section) sub-arcs alone — the cut edges the planar lid re-uses (reversed).
func emitLoopEdges(c uvSide, loop []dedge, segs []uvSeg) (face, section []loopEdge, ok bool) {
	rec := make([]recoveredEdge, 0, len(loop))
	for _, d := range loop {
		re, good := recoverEdge(d, segs)
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
