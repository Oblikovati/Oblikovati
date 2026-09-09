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

// TestTheTranslateNearestTheWindowMiddleWins pins the total order the choice is made on, so the same
// input gives the same mesh on every run and platform.
func TestTheTranslateNearestTheWindowMiddleWins(t *testing.T) {
	t.Parallel()
	b := seamStraddlingCover(t)
	inner, outer := b.centreOffset([3]int{0, 1, 2}), b.centreOffset([3]int{3, 4, 5})
	if !(inner < outer) {
		t.Fatalf("centre offsets %g and %g: the translate inside the window must read nearer", inner, outer)
	}
	if !b.nearerTheWindowCentre([3]int{0, 1, 2}, [3]int{3, 4, 5}) {
		t.Error("nearerTheWindowCentre chose the translate outside the window")
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
	for _, du := range []float64{0, 2 * stdmath.Pi} {
		for i, p := range []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)} {
			b.add(p, lo+du+float64(i)*1e-9, 0.5)
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
