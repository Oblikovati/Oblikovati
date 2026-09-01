// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"
)

// #1409: constraint recovery must walk the segment corridor, flipping O(crossings) edges with no
// whole-mesh rescan per flip — replacing the O(T²)–O(T³) scan-per-flip that froze imported faces (#1073).

// segmentCrossings counts the distinct live edges (excluding those touching a or b) that segment a→b
// properly crosses — the number of flips a correct recovery must perform.
func segmentCrossings(m *cdt, a, b int) int {
	pa, pb := m.pts[a], m.pts[b]
	seen := map[[2]int]bool{}
	n := 0
	for t := range m.tris {
		if m.dead[t] {
			continue
		}
		for i := range 3 {
			u, w := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
			key := conKey(u, w)
			if seen[key] {
				continue
			}
			seen[key] = true
			if u == a || u == b || w == a || w == b {
				continue
			}
			if segmentsCross(pa, pb, m.pts[u], m.pts[w]) {
				n++
			}
		}
	}
	return n
}

// liveArea sums the unsigned area of live, non-super triangles — invariant under Lawson flips, so it
// pins that recovery only re-triangulates (never gains or loses domain).
func liveArea(m *cdt) float64 {
	a := 0.0
	for t := range m.tris {
		if m.dead[t] || m.hasSuper(t) {
			continue
		}
		v := m.tris[t].v
		p, q, r := m.pts[v[0]], m.pts[v[1]], m.pts[v[2]]
		a += stdmath.Abs((q[0]-p[0])*(r[1]-p[1])-(r[0]-p[0])*(q[1]-p[1])) / 2
	}
	return a
}

// edgeManifold reports whether every undirected edge of the live triangulation is shared by at most two
// triangles — a tangle/overlap from a bad flip would push an edge past two.
func edgeManifold(m *cdt) bool {
	use := map[[2]int]int{}
	for t := range m.tris {
		if m.dead[t] {
			continue
		}
		for i := range 3 {
			use[conKey(m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3])]++
		}
	}
	for _, c := range use {
		if c > 2 {
			return false
		}
	}
	return true
}

// convexPolygonCDT triangulates n points on a circle (a convex polygon) with no constraints, returning
// the mesh ready for chord recovery. Points in convex position make every crossing quad convex, so a
// chord is recovered in exactly (crossings) flips with no deferral — the clean instrumented case.
func convexPolygonCDT(n int) *cdt {
	pts := make([][2]float64, n)
	for j := range n {
		a := 2 * stdmath.Pi * float64(j) / float64(n)
		pts[j] = [2]float64{10 * stdmath.Cos(a), 10 * stdmath.Sin(a)}
	}
	m := newCDT(pts)
	for ip := 0; ip < m.nsup; ip++ {
		m.insert(ip)
	}
	m.buildIncident()
	return m
}

// TestConstraintRecoveryFlipsExactlyCrossings is #1409 acceptance criterion 2: recovering a chord of a
// convex polygon flips EXACTLY the edges the chord crosses — the corridor walk does no wasted work and,
// critically, no whole-mesh rescan (the flip count is the local crossing count, not a function of T).
func TestConstraintRecoveryFlipsExactlyCrossings(t *testing.T) {
	t.Parallel()
	m := convexPolygonCDT(16)
	const a, b = 0, 5 // a chord skipping four vertices either side, guaranteed to cross interior diagonals
	crossings := segmentCrossings(m, a, b)
	if crossings < 3 {
		t.Fatalf("chord (%d,%d) crosses only %d edges; need a non-trivial corridor to exercise recovery", a, b, crossings)
	}
	areaBefore := liveArea(m)

	before := m.flipSteps
	if !m.recoverSegment(a, b) {
		t.Fatalf("chord (%d,%d) was not recovered", a, b)
	}
	flips := m.flipSteps - before

	if flips != crossings {
		t.Errorf("recovery flipped %d edges, want exactly %d (the crossings) — wasted flips or a rescan", flips, crossings)
	}
	if !m.hasEdge(a, b) {
		t.Errorf("recovered segment (%d,%d) is not a mesh edge", a, b)
	}
	if !edgeManifold(m) {
		t.Error("recovery left a non-manifold edge (a flip tangled the mesh)")
	}
	if got := liveArea(m); stdmath.Abs(got-areaBefore) > 1e-9 {
		t.Errorf("area changed across recovery: %.9f → %.9f (flips must preserve the domain)", areaBefore, got)
	}
}

// alternatingStripCDT builds a→b along y=0.5 with N interior points alternating below (y=0) and above
// (y=1) at x=1..N. The triangulation links consecutive interior points with edges crossing the line, so
// recovering (a,b) crosses ~N edges through a mesh of O(N) triangles — the dense-constraint corridor that
// was cubic before #1409. Returns the mesh and the (a,b) endpoints.
func alternatingStripCDT(n int) *cdt {
	pts := [][2]float64{{0, 0.5}, {float64(n + 1), 0.5}}
	for i := 1; i <= n; i++ {
		y := 0.0
		if i%2 == 1 {
			y = 1.0
		}
		pts = append(pts, [2]float64{float64(i), y})
	}
	m := newCDT(pts)
	for ip := 0; ip < m.nsup; ip++ {
		m.insert(ip)
	}
	m.buildIncident()
	return m
}

// TestConstraintRecoveryIsLinearInCrossings is #1409 acceptance criterion 1: recovering a long boundary
// segment through a dense interior completes with flips == crossings and scales LINEARLY with the corridor
// length (doubling N doubles the flips), never approaching the O(T²) cap the historical #1073 freeze hit.
func TestConstraintRecoveryIsLinearInCrossings(t *testing.T) {
	t.Parallel()
	for _, n := range []int{40, 80} {
		m := alternatingStripCDT(n)
		crossings := segmentCrossings(m, 0, 1)
		areaBefore := liveArea(m)

		before := m.flipSteps
		if !m.recoverSegment(0, 1) {
			t.Fatalf("N=%d: long segment was not recovered (cap hit / corridor stalled?)", n)
		}
		flips := m.flipSteps - before

		// The strip crosses exactly the N-1 edges between consecutive interior points, all convex.
		if crossings != n-1 {
			t.Fatalf("N=%d: expected %d crossings, measured %d (fixture drifted)", n, n-1, crossings)
		}
		if flips != crossings {
			t.Errorf("N=%d: flipped %d edges for %d crossings — recovery is not corridor-local", n, flips, crossings)
		}
		if !m.hasEdge(0, 1) || !edgeManifold(m) {
			t.Errorf("N=%d: recovery did not produce a manifold mesh containing the constraint", n)
		}
		if got := liveArea(m); stdmath.Abs(got-areaBefore) > 1e-9 {
			t.Errorf("N=%d: area changed across recovery %.9f → %.9f", n, areaBefore, got)
		}
	}
}

// TestConstraintRecoverySurvivesStaleIncidenceHint covers the incidence-hint rescan path: after the hint
// is corrupted (pointed at a dead triangle), recovery must still find each vertex's star and recover the
// chord — incidentTri detects the stale entry and refreshes it by scan.
func TestConstraintRecoverySurvivesStaleIncidenceHint(t *testing.T) {
	t.Parallel()
	m := convexPolygonCDT(12)
	for i := range m.incident {
		m.incident[i] = 0 // point every vertex at triangle 0, which an insertion long since killed
	}
	if !m.recoverSegment(0, 5) || !m.hasEdge(0, 5) {
		t.Error("recovery failed to recover (0,5) after the incidence hint was invalidated")
	}
	if !edgeManifold(m) {
		t.Error("stale-hint recovery left a non-manifold mesh")
	}
}

// TestConstrainedDelaunayDenseConcaveLoop drives the public entry on a deep concave loop with many
// boundary segments recovered through an interior, asserting it completes with the domain cut exactly
// (area = outer − notch, not filled) and watertight — the #1073 dense-constraint regression at the API
// level, which the corridor walk recovers without hitting any cap.
func TestConstrainedDelaunayDenseConcaveLoop(t *testing.T) {
	t.Parallel()
	// A comb: a wide base with K downward slots, so the boundary zig-zags deeply and its segments cross
	// many interior diagonals when the whole point set is triangulated at once.
	pts, loop := combPolygon(6, 8.0, 5.0)
	tris, unrec, fellBack := constrainedDelaunayChecked(pts, [][]int{loop})
	if len(tris) == 0 {
		t.Fatal("comb polygon produced no triangles")
	}
	if len(unrec) != 0 {
		t.Errorf("dense concave loop left %d unrecovered constraints (corridor walk should recover all): %v", len(unrec), unrec)
	}
	if fellBack {
		t.Error("dense concave loop fell back to earcut — recovery should keep the constrained mesh")
	}
	want := cdtPolyArea(pts, loop)
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("comb area %.6f, want %.6f (the boundary was not honoured)", got, want)
	}
}

// BenchmarkConstraintRecoveryStrip documents the #1409 win: recovering a constraint that crosses N
// edges through an O(N)-triangle mesh. With the corridor walk this is O(N) (flips == crossings, no
// rescan); the previous scan-per-flip was O(N²)+ per segment. Run with -benchtime/-cpu to compare sizes.
func BenchmarkConstraintRecoveryStrip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := alternatingStripCDT(200)
		m.recoverSegment(0, 1)
	}
}

// combPolygon returns a comb-shaped (deeply concave) closed loop: a base rectangle of width w and the
// given height, with k rectangular slots cut into its top edge. The returned loop indexes pts in CCW order.
func combPolygon(k int, w, h float64) ([][2]float64, []int) {
	var pts [][2]float64
	add := func(x, y float64) int { pts = append(pts, [2]float64{x, y}); return len(pts) - 1 }
	var loop []int
	loop = append(loop, add(0, 0), add(w, 0)) // base bottom edge, left→right
	// Right wall up, then teeth across the top from right to left: each tooth is up, across, down.
	loop = append(loop, add(w, h))
	toothW := w / float64(2*k+1)
	for j := range k {
		xr := w - float64(2*j+1)*toothW
		xl := w - float64(2*j+2)*toothW
		loop = append(loop, add(xr, h*0.4)) // down into the slot
		loop = append(loop, add(xl, h*0.4)) // across the slot floor
		loop = append(loop, add(xl, h))     // back up to the top
	}
	loop = append(loop, add(0, h)) // top-left corner, closing back to (0,0)
	return pts, loop
}
