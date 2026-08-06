// SPDX-License-Identifier: GPL-2.0-only

package degenerate

import "testing"

// TestCrossedTrimLoopActuallyCrosses guards the fixture's whole reason to exist. Every #2058 test
// asserts that a diagnostic REACHES the caller; if this loop ever stopped crossing itself, the
// tessellator would have nothing to report and those tests would go green for the wrong reason.
func TestCrossedTrimLoopActuallyCrosses(t *testing.T) {
	// Edge 0→1 against edge 2→3: the pair the out-of-order corner visit crosses.
	if !segmentsCross(crossedTrimUV[0], crossedTrimUV[1], crossedTrimUV[2], crossedTrimUV[3]) {
		t.Fatalf("crossedTrimUV %v is a simple polygon — it no longer degrades any triangulation",
			crossedTrimUV)
	}
}

// TestCrossedTrimBodyCarriesTwoFaces pins the repeat that lets a consumer's per-code aggregation be
// tested on more than a single occurrence.
func TestCrossedTrimBodyCarriesTwoFaces(t *testing.T) {
	if n := len(CrossedTrimBody().Faces()); n != 2 {
		t.Errorf("CrossedTrimBody has %d faces, want 2 (one bow-tie at each height)", n)
	}
}

// segmentsCross reports whether the open segments a→b and c→d intersect, by the standard
// opposite-orientation test on both endpoint pairs.
func segmentsCross(a, b, c, d [2]float64) bool {
	return orientation(a, b, c)*orientation(a, b, d) < 0 &&
		orientation(c, d, a)*orientation(c, d, b) < 0
}

// orientation is the signed area of triangle p,q,r: positive counter-clockwise, negative clockwise.
func orientation(p, q, r [2]float64) float64 {
	return (q[0]-p[0])*(r[1]-p[1]) - (q[1]-p[1])*(r[0]-p[0])
}
