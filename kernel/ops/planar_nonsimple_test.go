// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// selfCrossingBand builds a NON-SIMPLE (self-intersecting) closed polygon of 2*n vertices: two
// near-collinear rows joined into a crossed "bowtie band". It stands in for a transient,
// partially-constrained sketch face that gets revolved into a malformed solid and hover-picked
// mid add-in build — the input that used to drive planarTris' CDT flip-recovery to O(n·T²).
func selfCrossingBand(n int) []math.Point2 {
	pts := make([]math.Point2, 0, 2*n)
	for i := range n {
		pts = append(pts, math.P2(float64(i), 0.0001*float64(i%2)))
	}
	for i := n - 1; i >= 0; i-- {
		pts = append(pts, math.P2(float64(n-1-i), 0.5+0.0001*float64(i%2)))
	}
	return pts
}

// TestPlanarTrisNonSimpleFastPath is the regression for the pick-path frame-starvation freeze:
// planarTris on a self-intersecting ~264-vertex face must take the loopsSelfCross fast-path and
// skip the constrained-Delaunay entirely, returning a bounded best-effort mesh. Before the fix a
// single such face — hover-picked every frame while an async add-in build ran — drove the CDT flip
// recovery to O(n·T²) (~5.2 s), starving the frame-loop dispatcher and deadlocking placement. Asserted
// on the deterministic fast-path predicate rather than wall-clock time (repeatable on any CI host).
func TestPlanarTrisNonSimpleFastPath(t *testing.T) {
	t.Parallel()
	poly := selfCrossingBand(132) // 264 vertices
	if !loopsSelfCross(poly, nil) {
		t.Fatal("selfCrossingBand is not detected as non-simple; the fast-path would not engage")
	}
	if tris := planarTris(poly, nil); len(tris) == 0 {
		t.Fatal("planarTris returned no triangles for a non-simple face; want a bounded best-effort mesh")
	}
}

// TestUncoveredPlanarFaceRecordsDefect is the #3388 regression: the bounded best-effort mesh
// planarTris returns for a self-crossing boundary does NOT cover the face area, and that shortfall
// must travel with the mesh as a Defect — not ship silently as a clean face. A covering triangulation
// records nothing.
func TestUncoveredPlanarFaceRecordsDefect(t *testing.T) {
	t.Parallel()
	band := selfCrossingBand(16)
	tris := planarTris(band, nil)
	m := &Mesh{}
	recordUncoveredPlanarFace(m, tris, band, nil)
	if !hasDiag(m.Diagnostics, CodeTessellateDomainUncovered) {
		t.Fatalf("a self-crossing planar face shipped without a %q defect: %v", CodeTessellateDomainUncovered, m.Diagnostics)
	}
	if m.Diagnostics[0].Severity != diag.Defect {
		t.Errorf("domain-uncovered recorded at severity %v, want Defect", m.Diagnostics[0].Severity)
	}

	square := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)}
	clean := &Mesh{}
	recordUncoveredPlanarFace(clean, planarTris(square, nil), square, nil)
	if len(clean.Diagnostics) != 0 {
		t.Errorf("a covering square face recorded a spurious defect: %v", clean.Diagnostics)
	}
}

// TestConstrainedDelaunayNonSimpleBudget exercises the CDT's own recoverFlipWork budget in isolation
// — the backstop for every constrainedDelaunay caller (curved-surface (u,v) meshers, conformance
// repair), not just planarTris' fast-path. Fed a non-simple boundary directly, the flip-recovery
// budget must engage (m.overBudget), which is what caps the O(n·T²) spin and routes to the earcut
// fallback. This is deterministic (same input → same flip sequence), unlike a wall-clock bound.
func TestConstrainedDelaunayNonSimpleBudget(t *testing.T) {
	t.Parallel()
	poly := selfCrossingBand(132) // 264 vertices
	pts := make([][2]float64, len(poly))
	for i, p := range poly {
		pts[i] = [2]float64{float64(p.X), float64(p.Y)}
	}
	loops := [][]int{rangeIndices(0, len(poly))}

	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	m.constrain(loops)
	if !m.overBudget {
		t.Fatalf("flip-recovery budget (%d) did not engage on a 264-vertex non-simple boundary; "+
			"the O(n·T²) freeze can regress (recoverFlipWork=%d)", m.recoverBudget, m.recoverFlipWork)
	}
	if m.recoverFlipWork > m.recoverBudget {
		t.Fatalf("recoverFlipWork %d exceeded budget %d", m.recoverFlipWork, m.recoverBudget)
	}
	// A valid convex face never trips the budget (the corridor march recovers it without recoverByFlips).
	vm := newCDT(cvtPts(ngon2D(0, 0, 10, 64)))
	for i := 0; i < vm.nsup; i++ {
		vm.insert(i)
	}
	vm.constrain([][]int{rangeIndices(0, 64)})
	if vm.overBudget {
		t.Error("flip-recovery budget engaged on a valid convex n-gon; it must be invisible to valid faces")
	}
}

// cvtPts converts a Point2 loop to the CDT's [][2]float64 form.
func cvtPts(loop []math.Point2) [][2]float64 {
	pts := make([][2]float64, len(loop))
	for i, p := range loop {
		pts[i] = [2]float64{float64(p.X), float64(p.Y)}
	}
	return pts
}

// TestLoopsSelfCross checks the discriminator that routes a non-simple face past the CDT: it fires
// on a genuine transversal crossing and stays silent on valid simple boundaries (a clean n-gon and a
// holed face), so valid complex faces still reach the constrained Delaunay unchanged.
func TestLoopsSelfCross(t *testing.T) {
	t.Parallel()
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
