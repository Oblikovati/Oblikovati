// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestUnwrapPeriodRemovesSeamJump pins that a parameter that wraps the period (0.9→0.1 across the seam)
// becomes contiguous (0.9→1.1), so a rim's u advances monotonically instead of folding back.
func TestUnwrapPeriodRemovesSeamJump(t *testing.T) {
	t.Parallel()
	got := unwrapPeriod([]float64{0.8, 0.9, 0.0, 0.1, 0.2}, 1.0)
	want := []float64{0.8, 0.9, 1.0, 1.1, 1.2}
	for i := range want {
		if stdmath.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("unwrapPeriod[%d] = %.4f, want %.4f", i, got[i], want[i])
		}
	}
}

// TestCanonUWrapsIntoPeriod pins that canonU maps any u into [ulo, ulo+P).
func TestCanonUWrapsIntoPeriod(t *testing.T) {
	t.Parallel()
	cases := []struct{ u, want float64 }{{-0.2, 0.8}, {0.3, 0.3}, {1.4, 0.4}, {2.0, 0.0}}
	for _, c := range cases {
		if got := canonU(c.u, 0, 1); stdmath.Abs(got-c.want) > 1e-9 {
			t.Errorf("canonU(%.2f) = %.4f, want %.4f", c.u, got, c.want)
		}
	}
}

// TestAnyMouthStraddlesSeam pins the gate that selects the covering path: a mouth whose unwrapped u range
// crosses a period boundary straddles the seam; one wholly inside a period does not.
func TestAnyMouthStraddlesSeam(t *testing.T) {
	t.Parallel()
	inside := cylLoop{u: []float64{0.40, 0.46, 0.51, 0.46}}
	straddling := cylLoop{u: []float64{0.98, 1.01, 1.03, 0.99}}
	if anyMouthStraddlesSeam([]cylLoop{inside}, 0, 1) {
		t.Error("a mouth inside one period must not be reported as straddling")
	}
	if !anyMouthStraddlesSeam([]cylLoop{inside, straddling}, 0, 1) {
		t.Error("a mouth crossing a period boundary must be reported as straddling")
	}
}

// TestTraceClosedRingsRejoinsAcrossDroppedSeam pins that oriented segments chain head-to-tail by welded
// endpoints into a closed ring — the mechanism that rejoins a seam-split mouth's two arcs into one loop.
func TestTraceClosedRingsRejoinsAcrossDroppedSeam(t *testing.T) {
	t.Parallel()
	arcA := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)}
	arcB := []math.Point3{math.P3(1, 1, 0), math.P3(0, 1, 0), math.P3(0, 0, 0)}
	rings, ok := traceClosedRings([][]math.Point3{arcA, arcB})
	if !ok || len(rings) != 1 {
		t.Fatalf("traceClosedRings ok=%v rings=%d, want one ring", ok, len(rings))
	}
	if len(rings[0]) != 4 { // 3+3 points minus the two shared join points
		t.Errorf("ring has %d points, want 4", len(rings[0]))
	}
}

// TestTraceClosedRingsRejectsOpenChain pins that a dangling segment (no continuation) is reported as not
// closed, so the mesher defers rather than meshing a torn boundary.
func TestTraceClosedRingsRejectsOpenChain(t *testing.T) {
	t.Parallel()
	open := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}
	if _, ok := traceClosedRings([][]math.Point3{open}); ok {
		t.Error("an open chain must not trace to a closed ring")
	}
}

// TestPointInPolyUVRespectsShift pins the even-odd test used for the material/mouth region check, including
// the period shift that lets a canonical query match a seam-straddling mouth's other branch.
func TestPointInPolyUVRespectsShift(t *testing.T) {
	t.Parallel()
	lu := []float64{0.0, 0.2, 0.2, 0.0}
	lv := []float64{0.0, 0.0, 0.2, 0.2}
	if !pointInPolyUV(0.1, 0.1, lu, lv, 0) {
		t.Error("centre point should be inside the unit-shift-0 square")
	}
	if pointInPolyUV(0.1, 0.1, lu, lv, 1) {
		t.Error("point should be outside the square shifted +1 in u")
	}
	if !pointInPolyUV(1.1, 0.1, lu, lv, 1) {
		t.Error("point should be inside the square shifted +1 in u")
	}
}
