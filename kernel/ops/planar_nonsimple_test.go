// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"
	"time"

	"oblikovati.org/math"
)

// selfCrossingBand builds a NON-SIMPLE (self-intersecting) closed polygon of 2*n vertices: two
// near-collinear rows joined into a crossed "bowtie band". It stands in for a transient,
// partially-constrained sketch face that gets revolved into a malformed solid and hover-picked
// mid add-in build — the input that used to drive planarTris' CDT flip-recovery to O(n·T²).
func selfCrossingBand(n int) []math.Point2 {
	pts := make([]math.Point2, 0, 2*n)
	for i := 0; i < n; i++ {
		pts = append(pts, math.P2(float64(i), 0.0001*float64(i%2)))
	}
	for i := n - 1; i >= 0; i-- {
		pts = append(pts, math.P2(float64(n-1-i), 0.5+0.0001*float64(i%2)))
	}
	return pts
}

// TestPlanarTrisNonSimpleBounded is the regression for the pick-path frame-starvation freeze:
// planarTris on a self-intersecting ~264-vertex face must return a bounded triangulation quickly.
// Before the fix (loopsSelfCross fast-path + the CDT recoverFlipWork budget) this took ~5.2 s — a
// single such face, hover-picked every frame, starved the frame-loop dispatcher an async add-in
// build blocks on, deadlocking placement. The ceiling is ~40x the fixed cost and ~20x under the
// pre-fix cost, so it distinguishes fixed from broken without flaking on a slow CI host.
func TestPlanarTrisNonSimpleBounded(t *testing.T) {
	poly := selfCrossingBand(132) // 264 vertices
	start := time.Now()
	tris := planarTris(poly, nil)
	elapsed := time.Since(start)
	if elapsed > 250*time.Millisecond {
		t.Fatalf("planarTris on a 264-vertex self-intersecting face took %v; want < 250ms "+
			"(the O(n·T²) CDT flip-recovery freeze regressed)", elapsed)
	}
	if len(tris) == 0 {
		t.Fatal("planarTris returned no triangles for a non-simple face; want a bounded best-effort mesh")
	}
}

// TestConstrainedDelaunayNonSimpleBounded exercises the CDT's own recoverFlipWork budget in
// isolation — the backstop for every constrainedDelaunay caller (curved-surface (u,v) meshers,
// conformance repair), not just planarTris' fast-path. Fed a non-simple boundary directly, the
// triangulation must terminate quickly instead of spinning flip-recovery to O(n·T²).
func TestConstrainedDelaunayNonSimpleBounded(t *testing.T) {
	poly := selfCrossingBand(132) // 264 vertices
	pts := make([][2]float64, len(poly))
	for i, p := range poly {
		pts[i] = [2]float64{float64(p.X), float64(p.Y)}
	}
	loops := [][]int{rangeIndices(0, len(poly))}
	start := time.Now()
	tris := constrainedDelaunay(pts, loops)
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("constrainedDelaunay on a 264-vertex non-simple boundary took %v; want < 250ms "+
			"(flip-recovery budget regressed)", elapsed)
	}
	_ = tris // may be empty for a fully-degenerate domain; the point is bounded time, not coverage
}

// TestLoopsSelfCross checks the discriminator that routes a non-simple face past the CDT: it fires
// on a genuine transversal crossing and stays silent on valid simple boundaries (a clean n-gon and a
// holed face), so valid complex faces still reach the constrained Delaunay unchanged.
func TestLoopsSelfCross(t *testing.T) {
	if !loopsSelfCross(selfCrossingBand(20), nil) {
		t.Error("loopsSelfCross missed a self-intersecting boundary")
	}
	if loopsSelfCross(ngon2D(0, 0, 10, 48), nil) {
		t.Error("loopsSelfCross flagged a clean convex n-gon as non-simple")
	}
	outer := ngon2D(0, 0, 10, 32)
	hole := ngon2D(0, 0, 3, 24)
	if loopsSelfCross(outer, [][]math.Point2{hole}) {
		t.Error("loopsSelfCross flagged a valid holed face as non-simple")
	}
}
