// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	"testing"

	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// TestSimpleLoop2D covers the conformance-repair guard predicate: the boundary that broke it (the
// notched-face fillet loop, projected) self-intersects and must be rejected; a plain convex quad
// and a concave-but-simple polygon must be accepted.
func TestSimpleLoop2D(t *testing.T) {
	t.Parallel()
	// The real self-intersecting loop from the notched-box fillet (edge z=90,x∈[80,90] crosses
	// edge x=85,z∈[0,100] at (85,90)).
	selfInt := []math.Point2{math.P2(80, 100), math.P2(80, 90), math.P2(90, 90), math.P2(90, 100), math.P2(85, 100), math.P2(85, 0), math.P2(0, 0), math.P2(0, 100)}
	if tessellate.SimpleLoop2D(selfInt) {
		t.Error("self-intersecting loop reported as simple")
	}
	square := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 10), math.P2(0, 10)}
	if !tessellate.SimpleLoop2D(square) {
		t.Error("convex square reported as non-simple")
	}
	// An L-shape (concave but simple) must pass.
	ell := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 4), math.P2(4, 4), math.P2(4, 10), math.P2(0, 10)}
	if !tessellate.SimpleLoop2D(ell) {
		t.Error("simple concave L reported as non-simple")
	}
}
