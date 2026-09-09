// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestReplicasKeyTheSameTriangle: two covering triangles that are period images of each other share
// their 3D points exactly (addReplicas hands every translate the same math.Point3), so they key the
// same however their vertices are ordered.
func TestReplicasKeyTheSameTriangle(t *testing.T) {
	t.Parallel()
	pos := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)}
	a := weldedTriangleKey(pos, [3]int{0, 1, 2}, 1e-9)
	b := weldedTriangleKey(pos, [3]int{2, 0, 1}, 1e-9)
	if a != b {
		t.Errorf("the same 3D triangle keys as %v and %v; a replica is recognised by its points, not their order", a, b)
	}
}

// TestOneTranslateOfEachTriangleSurvives is the invariant the half-open window could not give: a
// triangle whose centroid lands ON the window's edge is offered at both ends, and exactly one ships.
func TestOneTranslateOfEachTriangleSurvives(t *testing.T) {
	t.Parallel()
	b := seamStraddlingCover(t)
	tris := [][3]int{{0, 1, 2}, {3, 4, 5}} // the same 3D triangle, one period apart
	keep := []bool{true, true}
	b.keepOneReplicaEach(tris, keep)
	if n := countTrue(keep); n != 1 {
		t.Errorf("%d of a triangle's two translates were kept; want exactly 1 (the seam ships it twice otherwise)", n)
	}
}

// TestATriangleWithOneTranslateIsKept: away from the seam a triangle has a single image, and the
// selection must not take it away.
func TestATriangleWithOneTranslateIsKept(t *testing.T) {
	t.Parallel()
	b := seamStraddlingCover(t)
	keep := []bool{true}
	b.keepOneReplicaEach([][3]int{{0, 1, 2}}, keep)
	if !keep[0] {
		t.Error("the only translate of a triangle was dropped; the selection may only choose between replicas")
	}
}

// TestTheLowerTranslateWins pins the total order the choice is made on: the INTEGER shift each vertex
// was laid at, so the same input gives the same mesh on every run and platform. The fixture is the
// configuration the corpus actually presents — a triangle whose centroid sits on the window's edge and
// its +2pi image on the other edge — where the distance-from-the-middle key the first cut used is
// EXACTLY tied and decides nothing (#3518 review I5).
func TestTheLowerTranslateWins(t *testing.T) {
	t.Parallel()
	b := seamStraddlingCover(t)
	inner, outer := b.translateOf([3]int{0, 1, 2}), b.translateOf([3]int{3, 4, 5})
	if inner >= outer {
		t.Fatalf("translate keys %d and %d: a triangle's two images must order strictly", inner, outer)
	}
	uIn, _ := b.centroid([3]int{0, 1, 2})
	uOut, _ := b.centroid([3]int{3, 4, 5})
	if lo, hi := b.r.uLo, b.r.uHi; stdmath.Abs(uIn-(lo+hi)/2) != stdmath.Abs(uOut-(lo+hi)/2) {
		t.Errorf("the fixture's two candidates are %g and %g from the window's middle; the row exists "+
			"because that distance is EXACTLY tied here, so it may not be the key", uIn, uOut)
	}
}

// seamStraddlingCover is a covering holding one 3D triangle twice: once with its centroid ON the
// window's low edge and once a whole period along, at the high edge. That is the configuration the
// near-pinch corridor presents (#3518) and the one a recomputed centroid decides inconsistently.
func seamStraddlingCover(t *testing.T) *chartCover {
	t.Helper()
	lo := stdmath.Pi
	b := &chartCover{r: chartRegion{uPer: true, uLo: lo, uHi: lo + 2*stdmath.Pi, vLo: 0, vHi: 1}}
	b.normalAt = func(float64, float64) math.Vector3 { return math.V3(0, 0, 1) }
	b.su, b.sv = 1, 1
	for at, du := range []float64{0, 2 * stdmath.Pi} {
		for _, p := range []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)} {
			b.add(p, lo+du, 0.5, at)
		}
	}
	return b
}

// countTrue is how many of the flags are set.
func countTrue(flags []bool) int {
	n := 0
	for _, f := range flags {
		if f {
			n++
		}
	}
	return n
}
