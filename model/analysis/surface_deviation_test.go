// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestDeviationOffsetCopyMatchesOffset (M36-F14 acceptance): the deviation of an exact offset copy
// equals the offset everywhere — the map's min/max/RMS all converge on the known distance.
func TestDeviationOffsetCopyMatchesOffset(t *testing.T) {
	const dz = 0.3
	target := acceptancePatch(t, 0, func(i, j int) float64 { return 0 })   // flat z=0
	src := acceptancePatch(t, 0, func(i, j int) float64 { return dz / 1 }) // same patch lifted +dz
	r := SurfaceDeviationToSurface(src, target, 9, 9)
	if stdmath.Abs(r.Min-dz) > 1e-6 || stdmath.Abs(r.Max-dz) > 1e-6 {
		t.Errorf("offset-copy deviation Min/Max = %g/%g, want both ≈ %g", r.Min, r.Max, dz)
	}
	if stdmath.Abs(r.RMS-dz) > 1e-6 {
		t.Errorf("offset-copy RMS = %g, want ≈ %g", r.RMS, dz)
	}
}

// TestDeviationFlagsGap: two panels a known distance apart report that distance as the max deviation
// and flag it out of a tight tolerance (but not a loose one).
func TestDeviationFlagsGap(t *testing.T) {
	const gap = 0.5
	a := acceptancePatch(t, 0, func(i, j int) float64 { return 0 })
	b := acceptancePatch(t, 0, func(i, j int) float64 { return gap / 1 })
	r := SurfaceDeviationToSurface(a, b, 8, 8)
	if stdmath.Abs(r.AbsMax-gap) > 1e-6 {
		t.Errorf("gap AbsMax = %g, want ≈ %g", r.AbsMax, gap)
	}
	if r.OutOfTolerance(0.1) == 0 {
		t.Errorf("a %g gap should be flagged out of a 0.1 tolerance", gap)
	}
	if n := r.OutOfTolerance(1.0); n != 0 {
		t.Errorf("a %g gap is within a 1.0 tolerance, got %d out-of-tol samples", gap, n)
	}
}

// TestDeviationSignedSides: a sample on the target's +normal side is positive, the −normal side
// negative (so a deviation map can colour over/under separately).
func TestDeviationSignedSides(t *testing.T) {
	target := acceptancePatch(t, 0, func(i, j int) float64 { return 0 })
	above := acceptancePatch(t, 0, func(i, j int) float64 { return 0.2 / 1 })
	below := acceptancePatch(t, 0, func(i, j int) float64 { return -0.2 })
	if r := SurfaceDeviationToSurface(above, target, 4, 4); r.Min < 0 {
		t.Errorf("above-target deviation should be positive, Min = %g", r.Min)
	}
	if r := SurfaceDeviationToSurface(below, target, 4, 4); r.Max > 0 {
		t.Errorf("below-target deviation should be negative, Max = %g", r.Max)
	}
}

// TestDeviationToPoints: deviation to a point set offset along the normal recovers the offset.
func TestDeviationToPoints(t *testing.T) {
	src := acceptancePatch(t, 0, func(i, j int) float64 { return 0 })
	// a dense grid of reference points 0.2 below the surface (z = -0.2).
	var pts []math.Point3
	for i := 0; i <= 8; i++ {
		for j := 0; j <= 8; j++ {
			pts = append(pts, math.P3(math.Scalar(float64(i)/8), math.Scalar(float64(j)/8), -0.2))
		}
	}
	r := SurfaceDeviationToPoints(src, pts, 5, 5)
	if stdmath.Abs(r.AbsMax-0.2) > 0.05 {
		t.Errorf("deviation-to-points AbsMax = %g, want ≈ 0.2", r.AbsMax)
	}
	if r.Max <= 0 {
		t.Errorf("surface above the points should read positive, Max = %g", r.Max)
	}
}
