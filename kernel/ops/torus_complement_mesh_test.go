// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// The chart-window helpers behind torusComplementMesh: index search and clamping over the grid lines.
func TestComplementWindowHelpers(t *testing.T) {
	xs := []float64{0, 1, 2, 3, 4}
	if got := lastBelow(xs, 2.5); got != 2 {
		t.Errorf("lastBelow(2.5)=%d, want 2", got)
	}
	if got := lastBelow(xs, -1); got != 0 { // below all → 0
		t.Errorf("lastBelow(-1)=%d, want 0", got)
	}
	if got := firstAbove(xs, 2.5); got != 3 {
		t.Errorf("firstAbove(2.5)=%d, want 3", got)
	}
	if got := firstAbove(xs, 9); got != 4 { // above all → last index
		t.Errorf("firstAbove(9)=%d, want 4", got)
	}
	for _, tc := range []struct{ in, lo, hi, want int }{{-1, 0, 4, 0}, {9, 0, 4, 4}, {2, 0, 4, 2}} {
		if got := clampIndex(tc.in, tc.lo, tc.hi); got != tc.want {
			t.Errorf("clampIndex(%d,%d,%d)=%d, want %d", tc.in, tc.lo, tc.hi, got, tc.want)
		}
	}
}

// centroidUV returns the mean (u,v) of a loop, and ovalWindow brackets the oval's bbox inside the chart.
func TestComplementCentroidAndWindow(t *testing.T) {
	loop := []math.Point2{math.P2(1, 2), math.P2(3, 2), math.P2(3, 4), math.P2(1, 4)}
	if uc, vc := centroidUV(loop); uc != 2 || vc != 3 {
		t.Errorf("centroidUV = (%g,%g), want (2,3)", uc, vc)
	}
	us := []float64{0, 1, 2, 3, 4, 5, 6}
	vs := []float64{0, 1, 2, 3, 4, 5, 6}
	i0, i1, j0, j1 := ovalWindow(us, vs, loop) // oval u∈[1,3], v∈[2,4]
	if i0 < 1 || i1 > len(us)-1 || i0 >= i1 || j0 < 1 || j1 > len(vs)-1 || j0 >= j1 {
		t.Errorf("ovalWindow = [%d,%d]x[%d,%d], want a non-empty interior bracket", i0, i1, j0, j1)
	}
}
