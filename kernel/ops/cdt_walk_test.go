// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

// denseGridCDT inserts an n×n grid of points (spatially coherent row order, like a patch's boundary
// plus its grid Steiner nodes) into a fresh triangulation with no constraints — the unconstrained
// dense case the adjacency walk accelerates — and returns the built cdt.
func denseGridCDT(n int) *cdt {
	pts := make([][2]float64, 0, n*n)
	for i := range n {
		for j := range n {
			pts = append(pts, [2]float64{float64(i), float64(j)})
		}
	}
	m := newCDT(pts)
	for ip := 0; ip < m.nsup; ip++ {
		m.insert(ip)
	}
	return m
}

// gridArea sums the absolute area of a cdt's live, non-super triangles.
func gridArea(m *cdt) float64 {
	a := 0.0
	for t := range m.tris {
		if m.dead[t] || m.hasSuper(t) {
			continue
		}
		v := m.tris[t].v
		p, q, r := m.pts[v[0]], m.pts[v[1]], m.pts[v[2]]
		s := 0.5 * ((q[0]-p[0])*(r[1]-p[1]) - (r[0]-p[0])*(q[1]-p[1]))
		if s < 0 {
			s = -s
		}
		a += s
	}
	return a
}

// TestCDTPointLocationIsNearLinear is acceptance criterion 1 of #1408: with the adjacency walk, point
// location costs a small, BOUNDED number of triangle visits per insertion that does NOT grow with N —
// so the whole insertion is near-linear, not the O(N²) the old firstBad full scan made it (≈ N²/2
// inCircle calls for location alone). Quadrupling N (40×40 → 80×80) must leave the per-insertion visit
// count essentially flat; an O(N²) location would roughly quadruple it. The mesh must also still cover
// each grid's convex hull exactly (a valid triangulation).
func TestCDTPointLocationIsNearLinear(t *testing.T) {
	small, large := denseGridCDT(40), denseGridCDT(80)
	for _, m := range []*cdt{small, large} {
		n := 40
		if m == large {
			n = 80
		}
		if got, want := gridArea(m), float64((n-1)*(n-1)); got != want {
			t.Errorf("n=%d: triangulated area %g, want %g (convex hull) — invalid triangulation", n, got, want)
		}
	}
	perSmall := float64(small.walkSteps) / float64(small.nsup)
	perLarge := float64(large.walkSteps) / float64(large.nsup)
	if perLarge > 2*perSmall {
		t.Errorf("per-insertion walk visits grew %.1f→%.1f when N grew 4× — location is not near-linear", perSmall, perLarge)
	}
	t.Logf("per-insertion visits: N=%d %.1f, N=%d %.1f (flat ⇒ near-linear; the old O(N²) scan ≈ %d inCircle calls at N=%d)",
		small.nsup, perSmall, large.nsup, perLarge, large.nsup*large.nsup/2, large.nsup)
}

// TestCDTLocateRecoversFromDeadHint covers the walk's stale-hint path: when the cached seed triangle
// has been deleted by an earlier insertion, liveSeed must skip it and restart from a live triangle, and
// locate must still land on the triangle containing the query point.
func TestCDTLocateRecoversFromDeadHint(t *testing.T) {
	m := denseGridCDT(8)
	for d := range m.dead {
		if m.dead[d] {
			m.last = d // point the hint at a dead triangle
			break
		}
	}
	q := [2]float64{3.5, 3.5} // interior of the 8×8 grid
	v := m.tris[m.locate(q)].v
	if orient2d(m.pts[v[1]], m.pts[v[2]], q) < 0 ||
		orient2d(m.pts[v[2]], m.pts[v[0]], q) < 0 ||
		orient2d(m.pts[v[0]], m.pts[v[1]], q) < 0 {
		t.Error("locate with a dead seed hint did not find a triangle containing the query point")
	}
}

// BenchmarkCDTDenseGridLocation measures the dense unconstrained triangulation the walk targets, for
// the #1408 speedup record. Reported ns/op falls roughly linearly with N where the old scan grew with
// N² (run with -benchtime to compare grid sizes).
func BenchmarkCDTDenseGridLocation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		denseGridCDT(40)
	}
}
