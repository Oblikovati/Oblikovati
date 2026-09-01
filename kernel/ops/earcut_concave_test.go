// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestEarcutConcaveOuterTwoHoles is the regression for the concave-outer + multi-hole bug: a
// face whose OUTER boundary is concave (an L, reflex corner) carrying TWO holes used to
// triangulate to the wrong area (9.514 vs 9.030), so a part's volume (a divergence sum over
// tessellated faces) was wrong after a second hole was cut into a concave body. Root cause: a
// flipped comparison in findHoleBridge picked the larger-x edge endpoint as the bridge
// candidate instead of the smaller-x one (Mapbox: m = p.x < p.next.x ? p : p.next), placing a
// hole bridge that crossed the concave notch. A convex outer with many holes (veroboard) and a
// concave outer with one hole both worked; only concave + multiple holes exposed it.
func TestEarcutConcaveOuterTwoHoles(t *testing.T) {
	t.Parallel()
	// L outer (CCW), area 4·4 − 2.5·2.5 = 9.75.
	outer := []math.Point2{
		math.P2(0, 0), math.P2(4, 0), math.P2(4, 1.5),
		math.P2(1.5, 1.5), math.P2(1.5, 4), math.P2(0, 4),
	}
	// Two 0.6×0.6 square holes (CW), one per arm, area 0.36 each.
	holes := [][]math.Point2{
		{math.P2(2.7, 0.45), math.P2(2.7, 1.05), math.P2(3.3, 1.05), math.P2(3.3, 0.45)},
		{math.P2(0.45, 2.7), math.P2(0.45, 3.3), math.P2(1.05, 3.3), math.P2(1.05, 2.7)},
	}
	combined := append([]math.Point2(nil), outer...)
	for _, h := range holes {
		combined = append(combined, h...)
	}
	const want = 9.75 - 0.36 - 0.36
	if got := triAreaSum(combined, earcut(outer, holes)); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("earcut(concave L, 2 holes) area = %.6f, want %.6f (Δ=%.6f) — concave outer + multi-hole mis-triangulated", got, want, got-want)
	}
}
