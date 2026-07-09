// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/math"
	"oblikovati.org/math/predicate"
)

// 2D constrained Delaunay triangulation (CDT) of a planar polygon (outer loop minus holes) with
// optional interior Steiner points. Built for trimmed curved-face tessellation in (u,v): the trim
// boundary loops become hard constraints, interior grid points refine the patch, and the domain is
// extracted by flooding across the constrained edges — so a CONCAVE trim is respected exactly (no
// triangles bridge a concavity) and a slightly self-intersecting boundary still meshes (the
// boundary segments are recovered by edge flips, then walls for the flood). The orientation and
// in-circle tests use the adaptive-EXACT predicates (see [orient2d]/[inCircle]): a planar face's
// boundary is sampled into near-collinear and near-cocircular points where a naive float determinant
// mis-signs and tangles the mesh.
//
// Algorithm: Bowyer–Watson incremental Delaunay over all points (connected-cavity insertion, which
// is robust to circumcircle round-off), then each boundary segment is recovered by flipping the
// edges it crosses, then triangles are 2-coloured inside/outside by flooding triangle adjacency and
// toggling at every constrained edge.

// cdtTri is one triangle: CCW vertices v, and n[i] = the neighbour across the edge opposite v[i]
// (the edge v[(i+1)%3]–v[(i+2)%3]), or -1 at the convex-hull boundary.
type cdtTri struct {
	v [3]int
	n [3]int
}

type cdt struct {
	pts  [][2]float64
	tris []cdtTri
	dead []bool // tris[t] was deleted by an insertion
	// con maps a constrained undirected edge (sorted vertex pair) to its MULTIPLICITY: how many
	// loop passes constrained it. Degenerate trims can run a hole boundary EXACTLY along a stretch
	// of the outer boundary (the unrolled holed wall does this at its seam), making one mesh edge a
	// DOUBLE wall — the domain flood must toggle inside/outside once per wall (odd/even parity), and
	// a plain set under-toggled there, silently mislabeling everything beyond the pinch (#1604).
	con  map[[2]int]int
	nsup int // index of the first super-triangle vertex (points >= nsup are super)
	last int // a recently-live triangle, the seed hint for the next point-location walk (#1408)
	// unrecovered collects the constraint endpoint pairs whose edge the flip recovery never realized
	// (the flip cap hit, or a non-convex crossing stalled). Such a constraint is NOT recorded in con —
	// a phantom con entry has no mesh edge for floodInside to toggle at, so the domain boundary leaks
	// across the gap silently (#1410). A non-empty list means extractDomain is untrustworthy and the
	// caller must take the deterministic earcut fallback instead.
	unrecovered [][2]int
	// walkSteps counts triangles visited by the adjacency walk over this triangulation's life — a
	// per-instance counter (each cdt runs on one goroutine, so no data race), read by tests to assert
	// point location is near-linear, not the old O(N²) full scan.
	walkSteps int
	// incident[v] is a live triangle containing vertex v — the hint that lets constraint recovery find a
	// vertex's star (and an edge incident to it) in O(deg) instead of an O(T) whole-mesh scan (#1409). It is
	// built once before recovery (buildIncident) and kept current by touch() on every addTri/flip. nil
	// before recovery, so insertion (which runs first) pays nothing; a stale entry is detected and rescanned.
	incident []int
	// flipSteps counts Lawson flips performed over this triangulation's life (constraint recovery and
	// the post-recovery legalization it queues — flip is called nowhere else), read by tests to assert
	// constraint recovery flips O(crossings) edges, not the O(T²) whole-mesh rescan-per-flip the
	// corridor walk replaced (#1409).
	flipSteps int
	// pendingLegal queues the diagonal created by each flip for in-circle legalization once the
	// current segment's recovery completes (#1604): recovery alone leaves the corridor non-Delaunay,
	// which both degrades triangle quality and invalidates the walk-based point location.
	pendingLegal [][2]int
	// recoverFlipWork counts flipOneCrossing invocations across ALL recoverByFlips calls, and
	// recoverBudget caps it. recoverByFlips is the O(T) whole-mesh fallback taken only when the
	// corridor march fails on a degeneracy; on a NON-SIMPLE (self-intersecting / overlapping /
	// pinched) boundary — a transient partially-constrained face hover-picked mid-build — that
	// degeneracy hits every constraint, so each spins its per-segment cap of O(T) scans and the
	// face costs O(n·T²) (seconds on a ~250-vertex face). That is synchronous per-frame pick work,
	// so it starves the frame-loop dispatcher an async add-in build depends on → hard deadlock.
	// A GLOBAL budget stops the spin once a face proves thoroughly degenerate; overBudget then
	// forces finalizeDomain to the deterministic earcut fallback. A valid face barely touches
	// recoverByFlips (the march handles it, and splitConstraintAtVertices recovers the rare
	// on-segment vertex even after a budget bail), so the budget is invisible to it.
	recoverFlipWork int
	recoverBudget   int
	overBudget      bool
}

func conKey(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

// orient2d > 0 when c is left of the directed line a→b (triangle abc is CCW). It delegates to the
// adaptive-exact predicate: the naive float determinant suffers catastrophic cancellation on near-
// collinear points (e.g. a face boundary discretized into many almost-straight samples), returning the
// WRONG sign — which made the Bowyer–Watson cavity collection grab the wrong triangles and emit
// inverted/overlapping triangles. The exact predicate's sign is always correct.
func orient2d(a, b, c [2]float64) float64 {
	return predicate.Orient2D(p2(a), p2(b), p2(c))
}

// inCircle > 0 when d is strictly inside the circumcircle of CCW triangle abc. Delegates to the
// adaptive-exact predicate for the same reason as orient2d: a planar face's circular hole is sampled
// into (near-)cocircular points, where the naive float in-circle determinant (radius² terms swamping
// their tiny difference) returns the wrong sign and tangles the triangulation.
func inCircle(a, b, c, d [2]float64) float64 {
	return predicate.InCircle(p2(a), p2(b), p2(c), p2(d))
}

func p2(p [2]float64) math.Point2 { return math.Point2{X: p[0], Y: p[1]} }

// newCDT builds the initial triangulation: a single super-triangle large enough to contain every
// input point (the 3 super vertices are appended to pts at indices nsup, nsup+1, nsup+2).
func newCDT(pts [][2]float64) *cdt {
	minx, miny := pts[0][0], pts[0][1]
	maxx, maxy := minx, miny
	for _, p := range pts {
		minx, maxx = min(minx, p[0]), max(maxx, p[0])
		miny, maxy = min(miny, p[1]), max(maxy, p[1])
	}
	d := max(maxx-minx, maxy-miny)
	if d <= 0 {
		d = 1
	}
	cx, cy := (minx+maxx)/2, (miny+maxy)/2
	nsup := len(pts)
	all := append([][2]float64(nil), pts...)
	all = append(all,
		[2]float64{cx - 20*d, cy - d}, [2]float64{cx + 20*d, cy - d}, [2]float64{cx, cy + 20*d})
	m := &cdt{pts: all, con: map[[2]int]int{}, nsup: nsup}
	m.tris = []cdtTri{{v: [3]int{nsup, nsup + 1, nsup + 2}, n: [3]int{-1, -1, -1}}}
	m.dead = []bool{false}
	// recoverByFlips is a rare legacy fallback: the corridor march (#1409) plus splitConstraintAtVertices
	// recover every VALID face's constraints without it (the whole CDT/tessellation suite passes with this
	// budget at 0). So a modest global budget lets a genuinely-hard segment still get a burst of flips while
	// capping a malformed face — where every constraint falls back and each flipOneCrossing is an O(T) scan —
	// at O(nsup) total scans instead of O(nsup²), the O(n·T²) freeze fixed here. See recoverFlipWork.
	m.recoverBudget = 2*nsup + 64
	return m
}

// localEdge returns the local index i (0..2) of the edge of triangle t between vertices a and b
// (unordered), or -1 if t has no such edge.
func (m *cdt) localEdge(t, a, b int) int {
	tri := m.tris[t]
	for i := 0; i < 3; i++ {
		u, w := tri.v[(i+1)%3], tri.v[(i+2)%3]
		if (u == a && w == b) || (u == b && w == a) {
			return i
		}
	}
	return -1
}

// relinkOpposite finds the edge of s shared with t and points it back at nt (used after t is
// replaced by a new triangle nt across that shared edge).
func (m *cdt) relinkOpposite(s, a, b, nt int) {
	if s < 0 {
		return
	}
	if i := m.localEdge(s, a, b); i >= 0 {
		m.tris[s].n[i] = nt
	}
}
