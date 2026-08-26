// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"
)

// Audit A8 (#1604): the CDT must be a TRUE constrained Delaunay triangulation. Constraint
// recovery used to flip only until the segment existed — no in-circle legalization afterwards —
// leaving the corridor non-Delaunay (arbitrarily bad triangles), and Bowyer–Watson insertion into
// the constrained mesh was seeded by an exhaustive circumcircle scan (firstBad) instead of the
// adjacency walk, hiding the damage at O(T)-per-point cost. These tests pin the repaired
// invariants on fixtures whose recovery genuinely degrades the mesh.

// cdtNonDelaunayEdges returns the interior edges violating the empty-circumcircle property:
// non-constrained edges between REAL vertices whose two adjacent triangles are also fully real
// (super-vertex circumcircles are the hull scaffold, not the domain property the CDT guarantees).
func cdtNonDelaunayEdges(m *cdt) [][2]int {
	seen := map[[2]int]bool{}
	var bad [][2]int
	for t := range m.tris {
		if m.dead[t] || m.hasSuper(t) {
			continue
		}
		for i := range 3 {
			u, w := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
			k := conKey(u, w)
			if seen[k] || m.con[k] > 0 {
				continue
			}
			seen[k] = true
			s := m.tris[t].n[i]
			if s < 0 || m.dead[s] || m.hasSuper(s) {
				continue
			}
			d := m.apex(s, u, w)
			if d < 0 {
				continue
			}
			if inCircle(m.pts[m.tris[t].v[0]], m.pts[m.tris[t].v[1]], m.pts[m.tris[t].v[2]], m.pts[d]) > 0 {
				bad = append(bad, k)
			}
		}
	}
	return bad
}

// assertCCWLive fails the test if any live non-super triangle is inverted or degenerate.
func assertCCWLive(t *testing.T, m *cdt) {
	t.Helper()
	for tr := range m.tris {
		if m.dead[tr] || m.hasSuper(tr) {
			continue
		}
		v := m.tris[tr].v
		if orient2d(m.pts[v[0]], m.pts[v[1]], m.pts[v[2]]) <= 0 {
			t.Fatalf("triangle %d is inverted/degenerate", tr)
		}
	}
}

// TestCDTLegalizedAfterRecovery pins A8 step 1 (#1604): recovering a chord through the alternating
// strip retriangulates its corridor as two fans, which are NOT Delaunay — before legalization this
// left 2 empty-circumcircle violations behind (measured). insertConstraint must legalize the
// corridor (recursive Lawson flips honoring constraints, Anglada/Sloan) after every recovery.
func TestCDTLegalizedAfterRecovery(t *testing.T) {
	for _, n := range []int{12, 40} {
		m := alternatingStripCDT(n)
		m.insertConstraint(0, 1)
		if m.con[conKey(0, 1)] == 0 {
			t.Fatalf("n=%d: chord (0,1) was not recovered", n)
		}
		if bad := cdtNonDelaunayEdges(m); len(bad) != 0 {
			t.Errorf("n=%d: %d non-constrained edges are not locally Delaunay after recovery (first: %v)", n, len(bad), bad[0])
		}
		if !edgeManifold(m) {
			t.Errorf("n=%d: legalization left a non-manifold edge", n)
		}
		assertCCWLive(t, m)
	}
}

// TestCDTInsertionIntoRecoveredCorridor pins the firstBad failure mode (#1604): points inserted
// INTO a just-recovered (previously folded) corridor must land in a mesh that legalization has
// restored to Delaunay, so the walk finds the containing triangle and the cavity is star-shaped —
// the result stays manifold, CCW, and fully Delaunay away from constraints.
func TestCDTInsertionIntoRecoveredCorridor(t *testing.T) {
	const n = 24
	pts := [][2]float64{{0, 0.5}, {float64(n + 1), 0.5}}
	for i := 1; i <= n; i++ {
		y := 0.0
		if i%2 == 1 {
			y = 1.0
		}
		pts = append(pts, [2]float64{float64(i), y})
	}
	nStrip := len(pts)
	for i := range 6 { // probes straddling the recovered chord, mid-corridor
		pts = append(pts, [2]float64{3.3 + 3*float64(i), 0.4 + 0.2*float64(i%2)})
	}
	m := newCDT(pts)
	for ip := range nStrip {
		m.insert(ip)
	}
	m.buildIncident()
	m.insertConstraint(0, 1)
	if m.con[conKey(0, 1)] == 0 {
		t.Fatal("chord (0,1) was not recovered")
	}
	area := liveArea(m)
	for ip := nStrip; ip < m.nsup; ip++ {
		m.insert(ip)
	}
	if !edgeManifold(m) {
		t.Error("insertion into the recovered corridor left a non-manifold edge")
	}
	assertCCWLive(t, m)
	if got := liveArea(m); stdmath.Abs(got-area) > 1e-9 {
		t.Errorf("insertion changed the hull area %.9f → %.9f (overlap or gap)", area, got)
	}
	if bad := cdtNonDelaunayEdges(m); len(bad) != 0 {
		t.Errorf("%d non-constrained edges non-Delaunay after corridor insertion (first: %v)", len(bad), bad[0])
	}
}

// combFixture is the end-to-end adversarial fixture: a wide slab whose top edge is a deep sawtooth
// comb (every tooth flank is an oblique constraint recovered through the unconstrained Delaunay),
// plus a jittered interior Steiner band under the teeth — insertions landing next to the recovered
// corridors. Returns points, the boundary loop, and the frontier count.
func combFixture() ([][2]float64, []int, int) {
	var pts [][2]float64
	pts = append(pts, [2]float64{0, 0}, [2]float64{12, 0}, [2]float64{12, 4})
	for x := 11.0; x >= 1.0; x -= 1.0 {
		pts = append(pts, [2]float64{x, 4}, [2]float64{x - 0.5, 0.8})
	}
	pts = append(pts, [2]float64{0, 4})
	loop := make([]int, len(pts))
	for i := range loop {
		loop[i] = i
	}
	nFrontier := len(pts)
	for x := 0.4; x <= 11.6; x += 0.4 {
		pts = append(pts, [2]float64{x, 0.2 + 0.2*stdmath.Sin(x*7)})
	}
	return pts, loop, nFrontier
}

// TestCDTConstraintSplitsAtOnSegmentVertex pins the segment-through-vertex resolution (#1604): a
// boundary segment whose open interior passes EXACTLY through another mesh vertex can never be one
// edge, so it must recover PIECEWISE (split at the on-segment vertex, Shewchuk/Anglada) instead of
// being dropped as unrecovered — which is how the unrolled periodic wall silently leaked its seam:
// the flood poured through the gap and the extracted domain lost a third of its area while the old
// folded scan-seeded mesh double-covered enough of it to pass the 3D area band.
func TestCDTConstraintSplitsAtOnSegmentVertex(t *testing.T) {
	// Unit-ish square whose bottom edge (0)-(1) passes exactly through point 4 at (2,0); 4 is not
	// part of the loop, so the constraint (0,1) is structurally unrecoverable as a single edge.
	pts := [][2]float64{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {2, 0}}
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	m.constrain([][]int{{0, 1, 2, 3}})
	if len(m.unrecovered) != 0 {
		t.Fatalf("constraint through an on-segment vertex was dropped as unrecovered (%v), want piecewise recovery", m.unrecovered)
	}
	if m.con[conKey(0, 4)] == 0 || m.con[conKey(4, 1)] == 0 {
		t.Error("split constraint pieces (0,4) and (4,1) are not recorded constrained")
	}
	tris := m.extractDomain()
	if got, want := cdtAreaSum(pts, tris), 16.0; stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("domain area = %g, want %g (the split boundary leaked)", got, want)
	}
	if bad := cdtNonDelaunayEdges(m); len(bad) != 0 {
		t.Errorf("%d non-constrained edges non-Delaunay after split recovery", len(bad))
	}
}

// TestCDTPinchedHoleWallParity pins the doubled-wall parity rule (#1604): when a hole boundary
// runs EXACTLY along a stretch of the outer boundary (the unrolled holed cylinder wall does this
// at its seam — the window's seam-side edge coincides with the outer seam segment), that stretch
// is TWO coincident walls on one mesh edge. The flood must toggle inside/outside once per wall
// (odd multiplicity toggles, even cancels); the old boolean con set under-toggled and every
// triangle beyond the pinch was mislabeled — the extracted domain was ~10% off while Validate
// stayed green.
func TestCDTPinchedHoleWallParity(t *testing.T) {
	// 10×10 square with a 4×4 notch-hole whose right edge lies ON the outer right edge x=10:
	// outer CCW, hole (6,3)-(10,3)-(10,7)-(6,7). Domain area = 100 − 16 = 84.
	pts := [][2]float64{
		{0, 0}, {10, 0}, {10, 3}, {10, 7}, {10, 10}, {0, 10}, // outer chain (right edge sampled at the pinch)
		{6, 3}, {10, 3}, {10, 7}, {6, 7}, // hole; 7 and 8 duplicate outer samples 2 and 3
	}
	outer := []int{0, 1, 2, 3, 4, 5}
	hole := []int{6, 7, 8, 9}
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	m.constrain([][]int{outer, hole})
	if len(m.unrecovered) != 0 {
		t.Fatalf("pinched fixture left %d constraints unrecovered", len(m.unrecovered))
	}
	if got := m.con[conKey(2, 3)]; got != 2 {
		t.Errorf("pinched edge (2,3) multiplicity = %d, want 2 (outer wall + hole wall)", got)
	}
	tris := m.extractDomain()
	if got, want := cdtAreaSum(pts, tris), 84.0; stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("pinched domain area = %g, want %g (double wall parity broken)", got, want)
	}
	assertCCWLive(t, m)
}

// TestCDTSteinerInsertionAfterConstraintsWalks pins A8 step 2 (#1604): interior Steiner points
// inserted into the constrained mesh must locate their cavity seed by the adjacency WALK (valid
// because legalization keeps the mesh a true CDT), not the O(T)-per-point firstBad scan — and the
// refined domain must stay exact: CCW everywhere, area equal two-sided, Delaunay off-constraints.
func TestCDTSteinerInsertionAfterConstraintsWalks(t *testing.T) {
	pts, loop, nFrontier := combFixture()
	m := newCDT(pts)
	for i := range nFrontier {
		m.insert(i)
	}
	m.constrain([][]int{loop})
	if len(m.unrecovered) != 0 {
		t.Fatalf("fixture drifted: %d boundary constraints unrecovered", len(m.unrecovered))
	}
	preWalk := m.walkSteps
	for i := nFrontier; i < m.nsup; i++ {
		m.insert(i)
	}
	if m.walkSteps == preWalk {
		t.Error("interior insertion after constraints never used the adjacency walk (firstBad scan seeding remains)")
	}
	assertCCWLive(t, m)
	tris := m.extractDomain()
	want := cdtPolyArea(pts, loop)
	if got := cdtAreaSum(pts, tris); stdmath.Abs(got-want) > 1e-9*want {
		t.Errorf("triangulated area = %.9f, want %.9f (gaps or overlaps near recovered corridors)", got, want)
	}
	if bad := cdtNonDelaunayEdges(m); len(bad) != 0 {
		t.Errorf("%d non-constrained edges non-Delaunay after Steiner insertion (first: %v)", len(bad), bad[0])
	}
}
